package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"partipix/backend/internal/database"
	"partipix/backend/internal/storage"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/bcrypt"
)

type UpdateProfileRequest struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Password string `json:"password,omitempty"`
}

type ProfileResponse struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Email           string           `json:"email"`
	Avatar          string           `json:"avatar"`
	IsAdmin         bool             `json:"isAdmin"`
	Events          []Event          `json:"events"`
	Photos          []Photo          `json:"photos"`
	ReferencePhotos []ReferencePhoto `json:"referencePhotos"`
}

type Event struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Date         string `json:"date"`
	Location     string `json:"location,omitempty"`
	Participants int    `json:"participants"`
	Code         string `json:"code"`
}

type ReferencePhoto struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	UploadedAt string `json:"uploadedAt"`
}

type ReferencePhotoUploadRequest struct {
	Files [][]byte `json:"files"`
}

type ReferencePhotoResponse struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	UploadedAt string `json:"uploadedAt"`
}

type DetectedFace struct {
	ID              string `json:"id"`
	URL             string `json:"url"`
	OriginalPhotoID string `json:"originalPhotoId"`
}

type ConfirmReferencePhotoRequest struct {
	PhotoID string `json:"photoId"`
	FaceID  string `json:"faceId"`
}

func (r *Router) handleGetProfile(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	// Get user
	user, err := database.GetUserByID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "User not found"}`)
		return
	}

	// Get user's events
	events, err := database.GetEventsByUserID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch events"}`)
		return
	}

	// Get user's photos
	photos, err := database.GetPhotosByUserID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch photos"}`)
		return
	}

	// Get user's reference photos
	referencePhotos, err := database.GetReferencePhotosByUserID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch reference photos"}`)
		return
	}

	// Convert database models to response models
	eventResponses := make([]Event, len(events))
	for i, e := range events {
		eventResponses[i] = Event{
			ID:           e.ID,
			Title:        e.Title,
			Description:  e.Description,
			Date:         e.Date.Format(time.RFC3339),
			Location:     e.Location,
			Participants: 0, // TODO: Count participants
			Code:         e.Code,
		}
	}

	photoResponses := make([]Photo, len(photos))
	for i, p := range photos {
		// Get people in photo
		peopleInPhoto, err := database.GetPhotoPeople(p.ID)
		if err != nil {
			log.Printf("Warning: Failed to get people in photo %s: %v", p.ID, err)
			peopleInPhoto = []string{}
		}

		// Get uploader's name
		uploader, err := database.GetUserByID(p.UploadedBy)
		uploaderName := p.UploadedBy // Default to ID if user not found
		if err == nil {
			uploaderName = uploader.Name
		}

		photoResponses[i] = Photo{
			ID:             p.ID,
			URL:            p.URL,
			UploadedBy:     p.UploadedBy,
			UploadedByName: uploaderName,
			UploadedAt:     p.UploadedAt.Format(time.RFC3339),
			EventID:        p.EventID,
			PeopleInPhoto:  peopleInPhoto,
		}
	}

	// Filter out reference photos with missing files
	validReferencePhotos := make([]ReferencePhoto, 0)
	for _, rp := range referencePhotos {
		// Remove the /uploads prefix to get the filesystem path
		filePath := strings.TrimPrefix(rp.URL, "/uploads/")
		filePath = filepath.Join(storage.UploadsDir, filePath)

		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			validReferencePhotos = append(validReferencePhotos, ReferencePhoto{
				ID:         rp.ID,
				URL:        rp.URL,
				UploadedAt: rp.UploadedAt.Format(time.RFC3339),
			})
		} else {
			// If file doesn't exist, delete the database record
			database.DeleteReferencePhoto(rp.ID)
		}
	}

	response := ProfileResponse{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		Avatar:          user.Avatar,
		IsAdmin:         user.IsAdmin,
		Events:          eventResponses,
		Photos:          photoResponses,
		ReferencePhotos: validReferencePhotos,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleUpdateProfile(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	var req UpdateProfileRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid request body"}`)
		return
	}

	// Get user
	user, err := database.GetUserByID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "User not found"}`)
		return
	}

	// Update fields if provided
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Password != "" {
		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Failed to process password"}`)
			return
		}
		user.Password = string(hashedPassword)
	}

	// Save changes
	if err := database.UpdateUser(user); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to update profile"}`)
		return
	}

	response := ProfileResponse{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		Avatar:          user.Avatar,
		IsAdmin:         user.IsAdmin,
		Events:          []Event{},          // Empty for update response
		Photos:          []Photo{},          // Empty for update response
		ReferencePhotos: []ReferencePhoto{}, // Empty for update response
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleUploadReferencePhotos(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	// Check existing photos count
	existingPhotos, err := database.GetReferencePhotosByUserID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to check existing photos"}`)
		return
	}

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
		ctx.SetBodyString(`{"error": "Invalid form data: ` + err.Error() + `"}`)
		return
	}

	files := form.File["photos"]
	if len(files) == 0 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "No photos provided"}`)
		return
	}

	// Check if total photos would exceed limit
	if len(existingPhotos)+len(files) > 5 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Maximum 5 photos allowed. You currently have ` + fmt.Sprint(len(existingPhotos)) + ` photos."}`)
		return
	}

	responses := make([]ReferencePhotoResponse, 0)
	for _, fileHeader := range files {
		// Save the file using our storage utility
		filePath, err := storage.SaveUploadedFile(fileHeader, storage.UploadsDir)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Failed to save photo"}`)
			return
		}

		// Create database record
		photo := &database.ReferencePhoto{
			UserID: userID,
			URL:    filePath,
		}

		if err := database.CreateReferencePhoto(photo); err != nil {
			// Clean up the file if database save fails
			storage.DeleteFile(filePath)
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Failed to save photo metadata"}`)
			return
		}

		responses = append(responses, ReferencePhotoResponse{
			ID:         photo.ID,
			URL:        photo.URL,
			UploadedAt: photo.UploadedAt.Format(time.RFC3339),
		})
	}

	ctx.SetStatusCode(fasthttp.StatusCreated)
	json.NewEncoder(ctx).Encode(responses)
}

func (r *Router) handleUploadReferencePhotosForVerification(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	// Check existing photos count
	existingPhotos, err := database.GetReferencePhotosByUserID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to check existing photos"}`)
		return
	}

	// Check if user already has 5 photos
	if len(existingPhotos) >= 5 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Maximum 5 photos allowed"}`)
		return
	}

	// Parse the multipart form
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid form data: ` + err.Error() + `"}`)
		return
	}

	files := form.File["photos"]
	if len(files) == 0 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "No photos provided"}`)
		return
	}

	responses := make([]DetectedFace, 0)
	for _, fileHeader := range files {
		// Save the file and get its ID from the filename
		filePath, err := storage.SaveUploadedFile(fileHeader, storage.UploadsDir)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Failed to save photo"}`)
			return
		}

		// Extract the ID from the saved file path (it's the filename without extension)
		photoID := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

		// Get the detected face crops from storage
		faceCrops, err := storage.GetFaceCrops(filePath)
		if err != nil {
			storage.DeleteFile(filePath) // Clean up the original file
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Failed to process faces"}`)
			return
		}

		// Clean up the original file since we only need the face crops
		storage.DeleteFile(filePath)

		// Start tracking these face scans
		storage.TrackFaceScan(photoID)

		// Add each detected face to the response
		for i, facePath := range faceCrops {
			faceID := fmt.Sprintf("%s_face_%d", photoID, i+1)
			responses = append(responses, DetectedFace{
				ID:              faceID,
				URL:             "/tmp/faces/" + filepath.Base(facePath),
				OriginalPhotoID: photoID,
			})
		}
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(responses)
}

func (r *Router) handleConfirmReferencePhoto(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	var req ConfirmReferencePhotoRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid request body"}`)
		return
	}

	log.Printf("Confirming face - Face ID: %s, Photo ID: %s", req.FaceID, req.PhotoID)

	// Find the face crop file in tmp/faces directory
	facePath := filepath.Join(storage.TmpDir, req.FaceID+".jpg")
	log.Printf("Looking for face crop at: %s", facePath)

	// Check if file exists
	if _, err := os.Stat(path.Join(storage.RootDir, facePath)); err != nil {
		log.Printf("Error finding face crop at %s: %v", facePath, err)

		// List available files to help debug
		if files, err := filepath.Glob(path.Join(storage.RootDir, storage.TmpDir, "*")); err == nil {
			log.Printf("Available face crops: %v", files)
		}

		// Clean up any remaining face crops
		if err := storage.CleanupFaceCrops(req.PhotoID); err != nil {
			log.Printf("Warning: Failed to clean up face crops: %v", err)
		}

		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf(`{"error": "Face crop not found at %s"}`, facePath))
		return
	}

	// Get face descriptor for logging and validation
	faceImg, err := imaging.Open(path.Join(storage.RootDir, facePath))
	if err != nil {
		log.Printf("Warning: Failed to open face image for descriptor: %v", err)

		// Clean up all face crops before returning error
		if err := storage.CleanupFaceCrops(req.PhotoID); err != nil {
			log.Printf("Warning: Failed to clean up face crops: %v", err)
		}

		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Failed to process face image. Please try uploading a clearer photo of your face."}`)
		return
	}

	descriptor, err := faceService.GetFaceDescriptor(faceImg)
	if err != nil {
		log.Printf("Warning: Failed to get face descriptor: %v", err)

		// Clean up all face crops before returning error
		if err := storage.CleanupFaceCrops(req.PhotoID); err != nil {
			log.Printf("Warning: Failed to clean up face crops: %v", err)
		}

		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Could not extract facial features. Please try uploading a clearer photo where your face is more visible."}`)
		return
	}

	// Calculate and log statistics about the descriptor
	var sum, min, max float32 = 0, descriptor[0], descriptor[0]
	for _, v := range descriptor {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	avg := sum / float32(len(descriptor))
	log.Printf("Face descriptor stats for %s - Avg: %.4f, Min: %.4f, Max: %.4f", req.PhotoID, avg, min, max)
	log.Printf("First 5 dimensions: %.4f, %.4f, %.4f, %.4f, %.4f",
		descriptor[0], descriptor[1], descriptor[2], descriptor[3], descriptor[4])

	// Move the face crop to profile pictures directory
	pfpPath, err := storage.MoveFaceToPfps(facePath)
	if err != nil {
		log.Printf("Failed to move face crop to profile pictures: %v", err)

		// Clean up all face crops before returning error
		if err := storage.CleanupFaceCrops(req.PhotoID); err != nil {
			log.Printf("Warning: Failed to clean up face crops: %v", err)
		}

		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf(`{"error": "Failed to save profile picture: %v"}`, err))
		return
	}
	log.Printf("Successfully moved face to profile pictures: %s", pfpPath)

	// Clean up all face crops for this photo
	if err := storage.CleanupFaceCrops(req.PhotoID); err != nil {
		log.Printf("Warning: Failed to clean up remaining face crops: %v", err)
		// Continue anyway since the main operation succeeded
	}

	// Mark the face scans as processed so they won't be cleaned up by timeout
	storage.MarkFaceScansProcessed(req.PhotoID)

	// Create database record
	photo := &database.ReferencePhoto{
		UserID: userID,
		URL:    pfpPath, // Use the path returned by MoveFaceToPfps directly
	}

	if err := database.CreateReferencePhoto(photo); err != nil {
		storage.DeleteFile(pfpPath)
		log.Printf("Failed to create reference photo record: %v", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to save photo metadata"}`)
		return
	}
	log.Printf("Successfully created reference photo record with ID: %s", photo.ID)

	// Compute and store face embedding for the reference photo
	fullPath := filepath.Join(storage.RootDir, pfpPath)
	if err := faceService.StoreEmbeddingForReferencePhoto(fullPath, photo.ID, userID); err != nil {
		log.Printf("Warning: Failed to store face embedding for reference photo %s: %v", photo.ID, err)
		// Continue anyway - the embedding storage is not critical for the API response
	} else {
		log.Printf("Successfully stored face embedding for reference photo %s", photo.ID)
	}

	response := ReferencePhotoResponse{
		ID:         photo.ID,
		URL:        photo.URL,
		UploadedAt: photo.UploadedAt.Format(time.RFC3339),
	}

	ctx.SetStatusCode(fasthttp.StatusCreated)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleDeleteReferencePhoto(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)
	photoID := ctx.UserValue("photoId").(string)

	// Get the photo to get its URL for file deletion
	photo, err := database.GetReferencePhotoByID(photoID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "Photo not found"}`)
		return
	}

	// Ensure the photo belongs to the user
	if photo.UserID != userID {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		ctx.SetBodyString(`{"error": "Not authorized to delete this photo"}`)
		return
	}

	// Get full path by prefixing with uploads if not already prefixed
	photoPath := photo.URL
	if !strings.HasPrefix(photoPath, "uploads/") && !strings.HasPrefix(photoPath, "/uploads/") {
		photoPath = path.Join("uploads", photoPath)
	}

	// Delete the file first
	if err := storage.DeleteFile(photoPath); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to delete photo file"}`)
		return
	}

	// Delete the database record
	if err := database.DeleteReferencePhoto(photoID); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to delete photo record"}`)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(`{"message": "Photo deleted successfully"}`)
}
