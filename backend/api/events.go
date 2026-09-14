package api

import (
	"encoding/json"
	"log"
	"partipix/backend/internal/database"
	"partipix/backend/internal/storage"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	"github.com/valyala/fasthttp"
)

type CreateEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Location    string `json:"location,omitempty"`
}

type JoinEventRequest struct {
	Code string `json:"code"`
}

type EventResponse struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Date         string  `json:"date"`
	Location     string  `json:"location,omitempty"`
	Participants int     `json:"participants"`
	Photos       []Photo `json:"photos"`
	Code         string  `json:"code"`
	Guests       []Guest `json:"guests"`
}

type Guest struct {
	Name   string `json:"name"`
	Status string `json:"status"` // 'uploaded' | 'not_uploading' | 'will_upload'
	IsHost bool   `json:"isHost"`
}

func (r *Router) handleCreateEvent(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	var req CreateEventRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid request body"}`)
		return
	}

	// Parse date
	date, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid date format"}`)
		return
	}

	// Create event
	event := &database.Event{
		Title:       req.Title,
		Description: req.Description,
		Date:        date,
		Location:    req.Location,
		CreatedBy:   userID,
	}

	if err := database.CreateEvent(event); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to create event"}`)
		return
	}

	// Add creator as participant
	participant := &database.EventParticipant{
		EventID: event.ID,
		UserID:  userID,
		Status:  "uploaded",
	}

	if err := database.AddEventParticipant(participant); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to add participant"}`)
		return
	}

	response := EventResponse{
		ID:           event.ID,
		Title:        event.Title,
		Description:  event.Description,
		Date:         event.Date.Format(time.RFC3339),
		Location:     event.Location,
		Participants: 1,
		Photos:       []Photo{},
		Code:         event.Code,
		Guests: []Guest{
			{
				Name:   "Test User", // TODO: Get actual user name
				Status: "uploaded",
				IsHost: true,
			},
		},
	}

	ctx.SetStatusCode(fasthttp.StatusCreated)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleGetEvents(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	events, err := database.GetEventsByUserID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch events"}`)
		return
	}

	response := make([]EventResponse, len(events))
	for i, event := range events {
		// Get participants with user info
		participants, users, err := database.GetEventParticipantsWithUsers(event.ID)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Failed to fetch participants"}`)
			return
		}

		// Get photos
		photos, err := database.GetPhotosByEventID(event.ID)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Failed to fetch photos"}`)
			return
		}

		// Convert to response format
		photoResponses := make([]Photo, len(photos))
		for j, photo := range photos {
			// Try to get uploader's name from users map first (more efficient)
			uploaderName := photo.UploadedBy // Default to ID if user not found
			if user, ok := users[photo.UploadedBy]; ok {
				uploaderName = user.Name
			} else {
				// If not in users map, try to get from database
				if uploader, err := database.GetUserByID(photo.UploadedBy); err == nil {
					uploaderName = uploader.Name
				}
			}

			photoResponses[j] = Photo{
				ID:             photo.ID,
				URL:            photo.URL,
				UploadedBy:     photo.UploadedBy,
				UploadedByName: uploaderName,
				UploadedAt:     photo.UploadedAt.Format(time.RFC3339),
				EventID:        photo.EventID,
				PeopleInPhoto:  []string{}, // TODO: Get people in photo
			}
		}

		guestResponses := make([]Guest, len(participants))
		for j, p := range participants {
			userName := "Unknown User"
			if user, ok := users[p.UserID]; ok {
				userName = user.Name
			}
			guestResponses[j] = Guest{
				Name:   userName,
				Status: p.Status,
				IsHost: p.UserID == event.CreatedBy,
			}
		}

		response[i] = EventResponse{
			ID:           event.ID,
			Title:        event.Title,
			Description:  event.Description,
			Date:         event.Date.Format(time.RFC3339),
			Location:     event.Location,
			Participants: len(participants),
			Photos:       photoResponses,
			Code:         event.Code,
			Guests:       guestResponses,
		}
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleGetEvent(ctx *fasthttp.RequestCtx) {
	eventID := ctx.UserValue("id").(string)

	event, err := database.GetEventByID(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "Event not found"}`)
		return
	}

	// Get participants with user info
	participants, users, err := database.GetEventParticipantsWithUsers(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch participants"}`)
		return
	}

	// Get photos
	photos, err := database.GetPhotosByEventID(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch photos"}`)
		return
	}

	// Convert to response format
	photoResponses := make([]Photo, len(photos))
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
		matchUsers, err := database.GetUsersFromMatches(matches)
		if err != nil {
			log.Printf("Warning: Failed to get user details for matches: %v", err)
			matchUsers = make(map[string]database.User)
		}

		// Convert matches to FaceMatch response objects
		faceMatches := make([]FaceMatch, 0)
		for _, match := range matches {
			userName := match.UserID // Default to ID if user not found
			if user, ok := matchUsers[match.UserID]; ok {
				userName = user.Name
			}

			// Get face location from the database
			location, fetchErr := database.GetFaceLocation(photo.ID, match.UserID)
			if fetchErr != nil {
				log.Printf("Warning: Failed to get face location for photo %s, user %s: %v",
					photo.ID, match.UserID, fetchErr)
				continue
			}

			faceMatches = append(faceMatches, FaceMatch{
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
			})
		}

		// Try to get uploader's name from users map first (more efficient)
		uploaderName := photo.UploadedBy // Default to ID if user not found
		if user, ok := users[photo.UploadedBy]; ok {
			uploaderName = user.Name
		} else {
			// If not in users map, try to get from database
			if uploader, err := database.GetUserByID(photo.UploadedBy); err == nil {
				uploaderName = uploader.Name
			}
		}

		photoResponses[i] = Photo{
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

	guestResponses := make([]Guest, len(participants))
	for i, participant := range participants {
		userName := "Unknown User"
		if user, ok := users[participant.UserID]; ok {
			userName = user.Name
		}
		guestResponses[i] = Guest{
			Name:   userName,
			Status: participant.Status,
			IsHost: participant.UserID == event.CreatedBy,
		}
	}

	response := EventResponse{
		ID:           event.ID,
		Title:        event.Title,
		Description:  event.Description,
		Date:         event.Date.Format(time.RFC3339),
		Location:     event.Location,
		Participants: len(participants),
		Photos:       photoResponses,
		Code:         event.Code,
		Guests:       guestResponses,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleJoinEvent(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	var req JoinEventRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid request body"}`)
		return
	}

	// Get event by code
	event, err := database.GetEventByCode(req.Code)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "Event not found"}`)
		return
	}

	// Get all photos in the event before we add the participant
	photos, err := database.GetPhotosByEventID(event.ID)
	if err != nil {
		log.Printf("Warning: Failed to get event photos: %v", err)
	}

	// Get all reference photos for the new participant
	_, err = database.GetReferencePhotosByUserID(userID)
	if err != nil {
		log.Printf("Warning: Failed to get reference photos for new participant %s: %v", userID, err)
	}

	// Add participant
	participant := &database.EventParticipant{
		EventID: event.ID,
		UserID:  userID,
		Status:  "not_uploading",
	}

	if err := database.AddEventParticipant(participant); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to join event"}`)
		return
	}

	userObj, err := database.GetUserByID(userID)
	if err != nil {
		log.Printf("Warning: Failed to get user details for %s: %v", userID, err)
		userObj = &database.User{ID: userID, Name: userID}
	}

	// Process each photo
	for _, photo := range photos {
		// Open the photo for face processing
		imgPath := filepath.Join(storage.RootDir, photo.URL)
		img, err := imaging.Open(imgPath)
		if err != nil {
			log.Printf("Warning: Failed to open photo %s for face detection: %v", photo.ID, err)
			continue
		}

		// Extract faces from the photo
		faces, faceRects, err := faceService.ExtractFaces(img)
		if err != nil {
			log.Printf("Warning: Failed to extract faces from photo %s: %v", photo.ID, err)
			continue
		}

		// Get existing matches to avoid reprocessing already matched faces
		existingMatches, err := database.GetPhotoMatches(photo.ID)
		if err != nil {
			log.Printf("Warning: Failed to get existing matches for photo %s: %v", photo.ID, err)
			continue
		}

		// Create a map to track which face rectangles are already matched
		matchedFaceIndices := make(map[int]bool)
		for _, match := range existingMatches {
			// Get face location to determine which face this match corresponds to
			loc, err := database.GetFaceLocation(photo.ID, match.UserID)
			if err != nil {
				continue
			}
			// Find which face index this corresponds to
			for i, rect := range faceRects {
				if rect.Min.X == loc.X && rect.Min.Y == loc.Y &&
					(rect.Max.X-rect.Min.X) == loc.Width &&
					(rect.Max.Y-rect.Min.Y) == loc.Height {
					matchedFaceIndices[i] = true
					break
				}
			}
		}

		// Process each unmatched face
		for i, face := range faces {
			// Skip if this face is already matched
			if matchedFaceIndices[i] {
				continue
			}

			// Get face descriptor
			descriptor, err := faceService.GetFaceDescriptor(face)
			if err != nil {
				log.Printf("Warning: Failed to get face descriptor for face #%d in photo %s: %v", i+1, photo.ID, err)
				continue
			}

			// Try to match against new participant's reference photos
			matchedIDs, confidences, err := faceService.FindMatchesWithEmbeddingsForUsers(descriptor, []string{userID})
			if err != nil {
				log.Printf("Warning: Failed to find matches for face #%d in photo %s: %v", i+1, photo.ID, err)
				continue
			}

			// Process any matches found
			for j, mUserID := range matchedIDs {
				confidence := confidences[j]
				rect := faceRects[i]

				// Store match in database
				dbMatch := &database.PhotoPersonMatch{
					PhotoID:    photo.ID,
					UserID:     mUserID,
					Confidence: confidence,
				}
				if err := database.AddPhotoPersonMatch(dbMatch); err != nil {
					log.Printf("Warning: Failed to save face match: %v", err)
					continue
				}

				// Store face location
				faceLocation := database.FaceLocation{
					PhotoID: photo.ID,
					UserID:  mUserID,
					X:       rect.Min.X,
					Y:       rect.Min.Y,
					Width:   rect.Max.X - rect.Min.X,
					Height:  rect.Max.Y - rect.Min.Y,
				}
				if err := database.SaveFaceLocation(faceLocation); err != nil {
					log.Printf("Warning: Failed to save face location: %v", err)
					continue
				}

				// Add to photo_people table
				photoPerson := &database.PhotoPerson{
					PhotoID: photo.ID,
					UserID:  mUserID,
				}
				if err := database.AddPhotoPerson(photoPerson); err != nil {
					log.Printf("Warning: Failed to save photo person: %v", err)
				}

				log.Printf("Found new match in photo %s for joining user %s (%s) with confidence %.2f%%",
					photo.ID, mUserID, userObj.Name, confidence*100)
			}
		}
	}

	// Get updated event details
	participants, users, err := database.GetEventParticipantsWithUsers(event.ID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch participants"}`)
		return
	}

	guestResponses := make([]Guest, len(participants))
	for i, p := range participants {
		userName := "Unknown User"
		if user, ok := users[p.UserID]; ok {
			userName = user.Name
		}
		guestResponses[i] = Guest{
			Name:   userName,
			Status: p.Status,
			IsHost: p.UserID == event.CreatedBy,
		}
	}

	response := EventResponse{
		ID:           event.ID,
		Title:        event.Title,
		Description:  event.Description,
		Date:         event.Date.Format(time.RFC3339),
		Location:     event.Location,
		Participants: len(participants),
		Photos:       []Photo{}, // Photos will be fetched by client
		Code:         event.Code,
		Guests:       guestResponses,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleUpdateEventStatus(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)
	eventID := ctx.UserValue("id").(string)

	var req struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid request body"}`)
		return
	}

	// Validate status
	if req.Status != "uploaded" && req.Status != "not_uploading" && req.Status != "will_upload" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid status"}`)
		return
	}

	// Update participant status
	participant := &database.EventParticipant{
		EventID: eventID,
		UserID:  userID,
		Status:  req.Status,
	}

	if err := database.UpdateEventParticipantStatus(participant); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to update status"}`)
		return
	}

	// Get updated event details
	event, err := database.GetEventByID(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch updated event"}`)
		return
	}

	// Get participants
	participants, users, err := database.GetEventParticipantsWithUsers(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch participants"}`)
		return
	}

	// Format response
	guestResponses := make([]Guest, len(participants))
	for i, p := range participants {
		userName := "Unknown User"
		if user, ok := users[p.UserID]; ok {
			userName = user.Name
		}
		guestResponses[i] = Guest{
			Name:   userName,
			Status: p.Status,
			IsHost: p.UserID == event.CreatedBy,
		}
	}

	response := EventResponse{
		ID:           event.ID,
		Title:        event.Title,
		Description:  event.Description,
		Date:         event.Date.Format(time.RFC3339),
		Location:     event.Location,
		Participants: len(participants),
		Photos:       []Photo{},
		Code:         event.Code,
		Guests:       guestResponses,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleDeleteUserEvent(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)
	eventID := ctx.UserValue("id").(string)

	// Get the event to check permissions
	event, err := database.GetEventByID(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "Event not found"}`)
		return
	}

	// Check if user is the event host
	if event.CreatedBy != userID {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		ctx.SetBodyString(`{"error": "Not authorized to delete this event"}`)
		return
	}

	if err := database.DeleteEvent(eventID); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to delete event"}`)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(`{"message": "Event deleted successfully"}`)
}
