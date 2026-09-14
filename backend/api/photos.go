package api

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"

	"partipix/backend/internal/database"
	"partipix/backend/internal/storage"
)

type UploadPhotoRequest struct {
	EventID       string   `json:"eventId"`
	File          []byte   `json:"file"`
	PeopleInPhoto []string `json:"peopleInPhoto,omitempty"`
}

type FaceMatch struct {
	UserID     string  `json:"userId"`
	UserName   string  `json:"userName"`
	Confidence float64 `json:"confidence"`
	Location   struct {
		X      int `json:"x"`
		Y      int `json:"y"`
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"location"`
}

type Photo struct {
	ID             string      `json:"id"`
	URL            string      `json:"url"`
	UploadedBy     string      `json:"uploadedBy"`
	UploadedByName string      `json:"uploadedByName"`
	UploadedAt     string      `json:"uploadedAt"`
	EventID        string      `json:"eventId"`
	PeopleInPhoto  []string    `json:"peopleInPhoto"`
	FaceMatches    []FaceMatch `json:"faceMatches"`
}

func (r *Router) handleUploadPhoto(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	// Parse the multipart form
	contentType := string(ctx.Request.Header.ContentType())
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Request must be multipart/form-data"}`)
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid form data"}`)
		return
	}

	// Get event ID from form
	eventID := string(ctx.FormValue("eventId"))
	if eventID == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Event ID is required"}`)
		return
	}

	files := form.File["photos"]
	if len(files) == 0 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "No photos provided"}`)
		return
	}

	// Validate event exists and user has access
	_, err = database.GetEventByID(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "Event not found"}`)
		return
	}

	// Get reference photos for all event participants
	participants, users, err := database.GetEventParticipantsWithUsers(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to get event participants"}`)
		return
	}

	// Get reference photos only from event participants
	var allReferencePhotos []string
	referencePhotoToUser := make(map[string]database.User) // Map to store user info for each reference photo
	for _, participant := range participants {
		// Skip if participant is not in users map (shouldn't happen but just in case)
		user, ok := users[participant.UserID]
		if !ok {
			continue
		}

		photos, err := database.GetReferencePhotosByUserID(participant.UserID)
		if err != nil {
			log.Printf("Warning: Failed to get reference photos for participant %s: %v", participant.UserID, err)
			continue
		}

		// Only add reference photos from this participant
		for _, photo := range photos {
			photoPath := filepath.Join(storage.PfpsDir, filepath.Base(photo.URL))
			// Store the user info for this reference photo
			referencePhotoToUser[filepath.Base(photo.URL)] = user
			log.Printf("Adding reference photo %s for participant %s (%s)",
				filepath.Base(photo.URL),
				user.Name,
				participant.UserID)
			allReferencePhotos = append(allReferencePhotos, photoPath)
		}
	}

	if len(allReferencePhotos) == 0 {
		log.Printf("Warning: No reference photos found for any participants in event %s", eventID)
	} else {
		log.Printf("Found %d total reference photos from event participants", len(allReferencePhotos))
	}

	responses := make([]Photo, 0)
	for _, fileHeader := range files {
		// Save the uploaded photo with event ID for proper organization
		log.Printf("Processing uploaded file: %s (Content-Type: %s)", fileHeader.Filename, fileHeader.Header.Get("Content-Type"))
		filePath, err := storage.SaveUploadedFile(fileHeader, eventID)
		if err != nil {
			log.Printf("ERROR: Failed to save uploaded file %s: %v", fileHeader.Filename, err)
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(fmt.Sprintf(`{"error": "Failed to save photo: %v"}`, err))
			return
		}

		// Create photo record in database with the path as is
		photo := &database.Photo{
			EventID:    eventID,
			URL:        "/" + filePath, // Just ensure it starts with / for consistency
			UploadedBy: userID,
		}

		if err := database.CreatePhoto(photo); err != nil {
			storage.DeleteFile(filePath)
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Failed to save photo metadata"}`)
			return
		}

		// Open the saved image for face detection
		img, err := imaging.Open(filepath.Join(storage.RootDir, filePath))
		if err != nil {
			log.Printf("Warning: Failed to open photo for face detection: %v", err)
			continue
		}

		// Extract faces from the photo
		faces, faceRects, err := faceService.ExtractFaces(img)
		if err != nil {
			log.Printf("Warning: Failed to extract faces from photo: %v", err)
			continue
		}

		// Get descriptors and find matches for each face
		peopleInPhoto := make([]string, 0)
		var faceMatches []FaceMatch

		// Track best match per user to avoid duplicates
		type userMatch struct {
			UserID     string
			UserName   string
			Confidence float64
			RefPhoto   string
		}
		bestMatchPerUser := make(map[string]userMatch)

		// Track users for whom we've already saved face locations
		faceLocationSaved := make(map[string]bool)

		// Initialize 'seen' map to track unique people in peopleInPhoto
		seen := make(map[string]bool)

		for i, face := range faces {
			descriptor, err := faceService.GetFaceDescriptor(face)
			if err != nil {
				log.Printf("Warning: Failed to get face descriptor for face #%d: %v", i+1, err)
				continue
			}

			// Get list of participant user IDs for matching
			participantIDs := make([]string, len(participants))
			for j, p := range participants {
				participantIDs[j] = p.UserID
			}

			// Use stored embeddings for matching, but only check against event participants
			matchedUserIDs, confidences, err := faceService.FindMatchesWithEmbeddingsForUsers(descriptor, participantIDs)
			if err != nil {
				log.Printf("Warning: Failed to find matches with embeddings for face #%d: %v", i+1, err)
				continue
			}

			// Process matches and keep only the best match per user
			rect := faceRects[i]
			for j, userID := range matchedUserIDs {
				confidence := confidences[j]
				// Look up the user details
				user, ok := users[userID]
				if !ok {
					// If not found in our cached map, try to fetch from database
					userObj, err := database.GetUserByID(userID)
					if err != nil {
						log.Printf("Warning: Failed to get user details for ID %s: %v", userID, err)
						continue
					}
					user = *userObj
					users[userID] = user // Cache for future use
				}

				// Check if this is a better match for this user
				if existing, exists := bestMatchPerUser[userID]; !exists || confidence > existing.Confidence {
					bestMatchPerUser[userID] = userMatch{
						UserID:     userID,
						UserName:   user.Name,
						Confidence: confidence,
						RefPhoto:   "", // We don't have this info when using embeddings directly
					}
				}
			}

			// Convert map to slice for sorting
			var userMatches []userMatch
			for _, match := range bestMatchPerUser {
				userMatches = append(userMatches, match)
			}

			// Sort matches by confidence (highest first)
			sort.Slice(userMatches, func(i, j int) bool {
				return userMatches[i].Confidence > userMatches[j].Confidence
			})

			// Store the best matches in database and response
			for _, match := range userMatches {
				// Store match in database
				dbMatch := &database.PhotoPersonMatch{
					PhotoID:    photo.ID,
					UserID:     match.UserID,
					Confidence: match.Confidence,
				}
				if err := database.AddPhotoPersonMatch(dbMatch); err != nil {
					log.Printf("Warning: Failed to save face match: %v", err)
					continue
				}

				// Only save face location once per user per photo
				if !faceLocationSaved[match.UserID] {
					// Store face location
					faceLocation := database.FaceLocation{
						PhotoID: photo.ID,
						UserID:  match.UserID,
						X:       rect.Min.X,
						Y:       rect.Min.Y,
						Width:   rect.Max.X - rect.Min.X,
						Height:  rect.Max.Y - rect.Min.Y,
					}
					if err := database.SaveFaceLocation(faceLocation); err != nil {
						log.Printf("Warning: Failed to save face location: %v", err)
					} else {
						// Mark that we've saved a face location for this user
						faceLocationSaved[match.UserID] = true
					}
				}

				// Also store in legacy photo_people table for backward compatibility
				// But only if we haven't already added this user to the photo
				if !seen[match.UserID] {
					photoPerson := &database.PhotoPerson{
						PhotoID: photo.ID,
						UserID:  match.UserID,
					}
					if err := database.AddPhotoPerson(photoPerson); err != nil {
						log.Printf("Warning: Failed to save photo person: %v", err)
						continue
					}

					// Add to peopleInPhoto only if not already added
					peopleInPhoto = append(peopleInPhoto, match.UserID)
					seen[match.UserID] = true
				}

				// Add face match to response array regardless
				faceMatches = append(faceMatches, FaceMatch{
					UserID:     match.UserID,
					UserName:   match.UserName,
					Confidence: match.Confidence,
					Location: struct {
						X      int `json:"x"`
						Y      int `json:"y"`
						Width  int `json:"width"`
						Height int `json:"height"`
					}{
						X:      rect.Min.X,
						Y:      rect.Min.Y,
						Width:  rect.Max.X - rect.Min.X,
						Height: rect.Max.Y - rect.Min.Y,
					},
				})
			}
		}

		// Remove duplicates from peopleInPhoto
		uniquePeople := make([]string, 0)
		for _, person := range peopleInPhoto {
			if !seen[person] {
				seen[person] = true
				uniquePeople = append(uniquePeople, person)
			}
		}

		// Get uploader's name from users map first (more efficient)
		uploaderName := userID // Default to ID if user not found
		if user, ok := users[userID]; ok {
			uploaderName = user.Name
		}

		responses = append(responses, Photo{
			ID:             photo.ID,
			URL:            photo.URL,
			UploadedBy:     photo.UploadedBy,
			UploadedByName: uploaderName,
			UploadedAt:     photo.UploadedAt.Format(time.RFC3339),
			EventID:        photo.EventID,
			PeopleInPhoto:  uniquePeople,
			FaceMatches:    faceMatches,
		})
	}

	// Return success response
	ctx.SetStatusCode(fasthttp.StatusCreated)
	json.NewEncoder(ctx).Encode(responses)
}

func (r *Router) handleGetEventPhotos(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)
	eventID := ctx.UserValue("eventId").(string)

	// Get photos separated by uploader
	photosByUploader, err := database.GetEventPhotosByUploader(eventID, userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch photos"}`)
		return
	}

	// Process photos by category
	response := struct {
		Uploaded []Photo `json:"uploaded"`
		Others   []Photo `json:"others"`
	}{
		Uploaded: make([]Photo, 0),
		Others:   make([]Photo, 0),
	}

	// Process uploaded photos
	for _, photo := range photosByUploader["uploaded"] {
		// Get people in photo
		peopleInPhoto, err := database.GetPhotoPeople(photo.ID)
		if err != nil {
			log.Printf("Warning: Failed to get people in photo %s: %v", photo.ID, err)
			peopleInPhoto = []string{}
		}

		// Get face matches with confidence scores
		matches, err := database.GetPhotoMatches(photo.ID)
		if err != nil {
			log.Printf("Warning: Failed to get face matches for photo %s: %v", photo.ID, err)
			matches = []database.PhotoPersonMatch{}
		}

		// Get user details for matches
		users, err := database.GetUsersFromMatches(matches)
		if err != nil {
			log.Printf("Warning: Failed to get user details for matches: %v", err)
			users = make(map[string]database.User)
		}

		// Convert matches to FaceMatch response objects
		faceMatches := make([]FaceMatch, len(matches))
		for i, match := range matches {
			userName := match.UserID // Default to ID if user not found
			if user, ok := users[match.UserID]; ok {
				userName = user.Name
			}
			// Get face location from the database
			location, fetchErr := database.GetFaceLocation(photo.ID, match.UserID)
			if fetchErr != nil {
				log.Printf("Warning: Failed to get face location for photo %s, user %s: %v",
					photo.ID, match.UserID, fetchErr)
				continue
			}
			faceMatches[i] = FaceMatch{
				UserID:     match.UserID,
				UserName:   userName,
				Confidence: match.Confidence,
				Location: struct {
					X      int `json:"x"`
					Y      int `json:"y"`
					Width  int `json:"width"`
					Height int `json:"height"`
				}{
					X:      location.X,
					Y:      location.Y,
					Width:  location.Width,
					Height: location.Height,
				},
			}
		}

		// Get uploader's name
		uploader, err := database.GetUserByID(photo.UploadedBy)
		uploaderName := photo.UploadedBy // Default to ID if user not found
		if err == nil {
			uploaderName = uploader.Name
		}

		photoResp := Photo{
			ID:             photo.ID,
			URL:            photo.URL,
			UploadedBy:     photo.UploadedBy,
			UploadedByName: uploaderName,
			UploadedAt:     photo.UploadedAt.Format(time.RFC3339),
			EventID:        photo.EventID,
			PeopleInPhoto:  peopleInPhoto,
			FaceMatches:    faceMatches,
		}
		response.Uploaded = append(response.Uploaded, photoResp)
	}

	// Process others' photos
	for _, photo := range photosByUploader["others"] {
		peopleInPhoto, err := database.GetPhotoPeople(photo.ID)
		if err != nil {
			log.Printf("Warning: Failed to get people in photo %s: %v", photo.ID, err)
			peopleInPhoto = []string{}
		}

		// Get face matches with confidence scores
		matches, err := database.GetPhotoMatches(photo.ID)
		if err != nil {
			log.Printf("Warning: Failed to get face matches for photo %s: %v", photo.ID, err)
			matches = []database.PhotoPersonMatch{}
		}

		// Get user details for matches
		users, err := database.GetUsersFromMatches(matches)
		if err != nil {
			log.Printf("Warning: Failed to get user details for matches: %v", err)
			users = make(map[string]database.User)
		}

		// Convert matches to FaceMatch response objects
		faceMatches := make([]FaceMatch, len(matches))
		for i, match := range matches {
			userName := match.UserID // Default to ID if user not found
			if user, ok := users[match.UserID]; ok {
				userName = user.Name
			}
			// Get face location from the database
			location, fetchErr := database.GetFaceLocation(photo.ID, match.UserID)
			if fetchErr != nil {
				log.Printf("Warning: Failed to get face location for photo %s, user %s: %v",
					photo.ID, match.UserID, fetchErr)
				continue
			}
			faceMatches[i] = FaceMatch{
				UserID:     match.UserID,
				UserName:   userName,
				Confidence: match.Confidence,
				Location: struct {
					X      int `json:"x"`
					Y      int `json:"y"`
					Width  int `json:"width"`
					Height int `json:"height"`
				}{
					X:      location.X,
					Y:      location.Y,
					Width:  location.Width,
					Height: location.Height,
				},
			}
		}

		uploader, err := database.GetUserByID(photo.UploadedBy)
		uploaderName := photo.UploadedBy
		if err == nil {
			uploaderName = uploader.Name
		}

		photoResp := Photo{
			ID:             photo.ID,
			URL:            photo.URL,
			UploadedBy:     photo.UploadedBy,
			UploadedByName: uploaderName,
			UploadedAt:     photo.UploadedAt.Format(time.RFC3339),
			EventID:        photo.EventID,
			PeopleInPhoto:  peopleInPhoto,
			FaceMatches:    faceMatches,
		}
		response.Others = append(response.Others, photoResp)
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleGetUserPhotos(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	photos, err := database.GetPhotosWithUser(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch photos"}`)
		return
	}

	responses := make([]Photo, len(photos))
	for i, photo := range photos {
		// Get people in photo
		peopleInPhoto, err := database.GetPhotoPeople(photo.ID)
		if err != nil {
			log.Printf("Warning: Failed to get people in photo %s: %v", photo.ID, err)
			peopleInPhoto = []string{}
		}

		// Get face matches with confidence scores
		matches, err := database.GetPhotoMatches(photo.ID)
		if err != nil {
			log.Printf("Warning: Failed to get face matches for photo %s: %v", photo.ID, err)
			matches = []database.PhotoPersonMatch{}
		}

		// Get user details for matches
		users, err := database.GetUsersFromMatches(matches)
		if err != nil {
			log.Printf("Warning: Failed to get user details for matches: %v", err)
			users = make(map[string]database.User)
		}

		// Convert matches to FaceMatch response objects
		faceMatches := make([]FaceMatch, len(matches))
		for i, match := range matches {
			userName := match.UserID // Default to ID if user not found
			if user, ok := users[match.UserID]; ok {
				userName = user.Name
			}
			// Get face location from the database
			location, fetchErr := database.GetFaceLocation(photo.ID, match.UserID)
			if fetchErr != nil {
				log.Printf("Warning: Failed to get face location for photo %s, user %s: %v",
					photo.ID, match.UserID, fetchErr)
				continue
			}
			faceMatches[i] = FaceMatch{
				UserID:     match.UserID,
				UserName:   userName,
				Confidence: match.Confidence,
				Location: struct {
					X      int `json:"x"`
					Y      int `json:"y"`
					Width  int `json:"width"`
					Height int `json:"height"`
				}{
					X:      location.X,
					Y:      location.Y,
					Width:  location.Width,
					Height: location.Height,
				},
			}
		}

		// Get uploader's name
		uploader, err := database.GetUserByID(photo.UploadedBy)
		uploaderName := photo.UploadedBy // Default to ID if user not found
		if err == nil {
			uploaderName = uploader.Name
		}

		responses[i] = Photo{
			ID:             photo.ID,
			URL:            photo.URL,
			UploadedBy:     photo.UploadedBy,
			UploadedByName: uploaderName,
			UploadedAt:     photo.UploadedAt.Format(time.RFC3339),
			EventID:        photo.EventID,
			PeopleInPhoto:  peopleInPhoto,
			FaceMatches:    faceMatches,
		}
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(responses)
}

func (r *Router) handleGetPhotosWithUser(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	// Get photos where the user appears using facial recognition matches
	photos, err := database.GetPhotosWithPerson(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch photos"}`)
		return
	}

	responses := make([]Photo, len(photos))
	for i, photo := range photos {
		// Get people in photo
		peopleInPhoto, err := database.GetPhotoPeople(photo.ID)
		if err != nil {
			log.Printf("Warning: Failed to get people in photo %s: %v", photo.ID, err)
			peopleInPhoto = []string{}
		}

		// Get face matches with confidence scores
		matches, err := database.GetPhotoMatches(photo.ID)
		if err != nil {
			log.Printf("Warning: Failed to get face matches for photo %s: %v", photo.ID, err)
			matches = []database.PhotoPersonMatch{}
		}

		// Get user details for matches
		users, err := database.GetUsersFromMatches(matches)
		if err != nil {
			log.Printf("Warning: Failed to get user details for matches: %v", err)
			users = make(map[string]database.User)
		}

		// Convert matches to FaceMatch response objects
		faceMatches := make([]FaceMatch, len(matches))
		for i, match := range matches {
			userName := match.UserID // Default to ID if user not found
			if user, ok := users[match.UserID]; ok {
				userName = user.Name
			}
			// Get face location from the database
			location, fetchErr := database.GetFaceLocation(photo.ID, match.UserID)
			if fetchErr != nil {
				log.Printf("Warning: Failed to get face location for photo %s, user %s: %v",
					photo.ID, match.UserID, fetchErr)
				continue
			}
			faceMatches[i] = FaceMatch{
				UserID:     match.UserID,
				UserName:   userName,
				Confidence: match.Confidence,
				Location: struct {
					X      int `json:"x"`
					Y      int `json:"y"`
					Width  int `json:"width"`
					Height int `json:"height"`
				}{
					X:      location.X,
					Y:      location.Y,
					Width:  location.Width,
					Height: location.Height,
				},
			}
		}

		// Get uploader's name
		uploader, err := database.GetUserByID(photo.UploadedBy)
		uploaderName := photo.UploadedBy // Default to ID if user not found
		if err == nil {
			uploaderName = uploader.Name
		}

		responses[i] = Photo{
			ID:             photo.ID,
			URL:            photo.URL,
			UploadedBy:     photo.UploadedBy,
			UploadedByName: uploaderName,
			UploadedAt:     photo.UploadedAt.Format(time.RFC3339),
			EventID:        photo.EventID,
			PeopleInPhoto:  peopleInPhoto,
			FaceMatches:    faceMatches,
		}
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(responses)
}

// handleDeletePhoto handles the deletion of a photo
func (r *Router) handleDeletePhoto(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)
	photoID := ctx.UserValue("id").(string)

	// Get the photo to check permissions and get its URL for file deletion
	photo, err := database.GetPhotoByID(photoID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "Photo not found"}`)
		return
	}

	// Get the event to check if user is the host
	event, err := database.GetEventByID(photo.EventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to check event permissions"}`)
		return
	}

	// Check if user has permission to delete the photo
	if photo.UploadedBy != userID && event.CreatedBy != userID {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		ctx.SetBodyString(`{"error": "Not authorized to delete this photo"}`)
		return
	}

	// Delete all associated data first
	if err := database.Transaction(func(tx *gorm.DB) error {
		// Delete face matches
		if err := tx.Where("photo_id = ?", photoID).Delete(&database.PhotoPersonMatch{}).Error; err != nil {
			return err
		}

		// Delete face locations
		if err := tx.Where("photo_id = ?", photoID).Delete(&database.FaceLocation{}).Error; err != nil {
			return err
		}

		// Delete photo tags
		if err := tx.Where("photo_id = ?", photoID).Delete(&database.PhotoPerson{}).Error; err != nil {
			return err
		}

		// Delete the photo record
		if err := tx.Delete(&database.Photo{}, "id = ?", photoID).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to delete photo data"}`)
		return
	}

	// Delete the actual file
	if err := storage.DeleteFile(photo.URL); err != nil {
		log.Printf("Warning: Failed to delete photo file %s: %v", photo.URL, err)
		// Continue anyway since the database records are gone
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(`{"message": "Photo deleted successfully"}`)
}
