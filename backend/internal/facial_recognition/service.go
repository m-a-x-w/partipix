package facial_recognition

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/Kagami/go-face"
	"github.com/disintegration/imaging"

	"partipix/backend/internal/database"
	"partipix/backend/internal/storage"
)

// Service handles facial recognition operations
type Service struct {
	rec *face.Recognizer
}

// FaceMatch represents a face match with coordinates
type FaceMatch struct {
	UserID     string
	Confidence float64
	Rectangle  image.Rectangle
}

// NewService creates a new facial recognition service
func NewService() (*Service, error) {
	// Initialize the face recognizer
	rec, err := face.NewRecognizer("models")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize face recognizer: %v", err)
	}

	return &Service{
		rec: rec,
	}, nil
}

// DetectFaces detects faces in an image and returns their coordinates
func (s *Service) DetectFaces(img image.Image) ([]face.Face, error) {
	// Save image temporarily since dlib requires file input
	tmpPath := filepath.Join(os.TempDir(), "face_detect.jpg")
	if err := imaging.Save(img, tmpPath); err != nil {
		return nil, fmt.Errorf("failed to save temp image: %v", err)
	}
	defer os.Remove(tmpPath)

	// Detect faces
	faces, err := s.rec.RecognizeFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect faces: %v", err)
	}

	return faces, nil
}

// ExtractFaces extracts face regions from an image and returns cropped face images and their coordinates
func (s *Service) ExtractFaces(img image.Image) ([]image.Image, []image.Rectangle, error) {
	// Save image temporarily since dlib requires file input
	tmpPath := filepath.Join(os.TempDir(), "face_extract.jpg")
	if err := imaging.Save(img, tmpPath); err != nil {
		return nil, nil, fmt.Errorf("failed to save temp image: %v", err)
	}
	defer os.Remove(tmpPath)

	faces, err := s.rec.RecognizeFile(tmpPath)
	if err != nil {
		return nil, nil, err
	}

	faceImages := make([]image.Image, 0, len(faces))
	faceRects := make([]image.Rectangle, 0, len(faces))

	for i, face := range faces {
		rect := face.Rectangle
		// Add padding around the face
		padding := 0.2 // 20% padding
		width := rect.Max.X - rect.Min.X
		height := rect.Max.Y - rect.Min.Y

		paddedRect := image.Rectangle{
			Min: image.Point{
				X: max(0, rect.Min.X-int(float64(width)*padding)),
				Y: max(0, rect.Min.Y-int(float64(height)*padding)),
			},
			Max: image.Point{
				X: min(img.Bounds().Max.X, rect.Max.X+int(float64(width)*padding)),
				Y: min(img.Bounds().Max.Y, rect.Max.Y+int(float64(height)*padding)),
			},
		}

		// Log face detection details
		log.Printf("Face #%d detected at coordinates: x=%d-%d, y=%d-%d (size: %dx%d)",
			i+1, rect.Min.X, rect.Max.X, rect.Min.Y, rect.Max.Y,
			rect.Max.X-rect.Min.X, rect.Max.Y-rect.Min.Y)

		// Crop and standardize the face
		faceCrop := imaging.Crop(img, paddedRect)
		faceCrop = imaging.Fit(faceCrop, 150, 150, imaging.Lanczos)
		faceImages = append(faceImages, faceCrop)
		faceRects = append(faceRects, rect)
	}

	return faceImages, faceRects, nil
}

// LogEmbedding prints a human-readable version of a face descriptor
func logEmbedding(desc face.Descriptor, prefix string) {
	// Print first few dimensions as sample
	log.Printf("%s Embedding preview (first 10/128 dimensions):", prefix)
	for i := 0; i < 10 && i < len(desc); i++ {
		log.Printf("  Dim %d: %.4f", i, desc[i])
	}

	// Calculate and print some statistics
	var sum, min, max float32 = 0, desc[0], desc[0]
	for _, v := range desc {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	avg := sum / float32(len(desc))
	log.Printf("%s Embedding stats - Avg: %.4f, Min: %.4f, Max: %.4f", prefix, avg, min, max)
}

// GetFaceDescriptor returns a face descriptor
func (s *Service) GetFaceDescriptor(img image.Image) (face.Descriptor, error) {
	// Save image temporarily since dlib requires file input
	tmpPath := filepath.Join(os.TempDir(), "face_desc.jpg")
	if err := imaging.Save(img, tmpPath); err != nil {
		var emptyDesc face.Descriptor
		return emptyDesc, fmt.Errorf("failed to save temp image: %v", err)
	}
	defer os.Remove(tmpPath)

	faces, err := s.rec.RecognizeFile(tmpPath)
	if err != nil || len(faces) == 0 {
		var emptyDesc face.Descriptor
		return emptyDesc, fmt.Errorf("failed to get face descriptor: %v", err)
	}

	// Log the embedding for reference photos
	logEmbedding(faces[0].Descriptor, "New Reference Photo")
	return faces[0].Descriptor, nil
}

// CompareFaces compares two face descriptors and returns a similarity score (0-1)
func (s *Service) CompareFaces(desc1, desc2 face.Descriptor) float64 {
	similarity := 1.0 - face.SquaredEuclideanDistance(desc1, desc2)

	// Log comparison details
	log.Printf("Face Comparison - Similarity Score: %.4f", similarity)
	logEmbedding(desc1, "Face 1")
	logEmbedding(desc2, "Face 2")

	return similarity
}

// FindMatches finds matching faces in reference photos
func (s *Service) FindMatches(faceDesc face.Descriptor, referencePhotos []string) ([]string, error) {
	matches := make([]string, 0)
	matchThreshold := 0.6 // Minimum similarity score to consider a match

	for _, refPath := range referencePhotos {
		refFaces, err := s.rec.RecognizeFile(refPath)
		if err != nil || len(refFaces) == 0 {
			log.Printf("Warning: Failed to process reference photo %s: %v", refPath, err)
			continue
		}

		// Check each face in the reference photo
		for _, refFace := range refFaces {
			similarity := s.CompareFaces(faceDesc, refFace.Descriptor)
			if similarity >= matchThreshold {
				// Extract user ID from reference photo path
				fileName := filepath.Base(refPath)
				userID := strings.Split(fileName, "_")[0]
				matches = append(matches, userID)
				break // One match per reference photo is enough
			}
		}
	}

	return matches, nil
}

// FindMatchesWithConfidence finds matching faces in reference photos and returns user IDs and confidence scores
func (s *Service) FindMatchesWithConfidence(faceDesc face.Descriptor, referencePhotos []string) ([]string, []float64, error) {
	matches := make([]string, 0)
	confidences := make([]float64, 0)
	matchThreshold := 0.6 // Minimum similarity score to consider a match

	// Keep track of best match for logging
	var bestMatch struct {
		UserID     string
		Confidence float64
		PhotoPath  string
	}
	bestMatch.Confidence = -1

	// Log total number of reference photos to process
	log.Printf("Processing %d reference photos for face matching", len(referencePhotos))

	// Calculate and log input face descriptor statistics for comparison
	logEmbedding(faceDesc, "Input Face")

	for _, refPath := range referencePhotos {
		// Log the reference photo being processed
		log.Printf("Trying reference photo: %s", refPath)

		// Check if file exists before processing
		if _, err := os.Stat(refPath); os.IsNotExist(err) {
			log.Printf("Warning: Reference photo does not exist: %s", refPath)
			continue
		}

		refFaces, err := s.rec.RecognizeFile(refPath)
		if err != nil || len(refFaces) == 0 {
			log.Printf("Warning: Failed to process reference photo %s: %v", refPath, err)
			continue
		}

		// Check each face in the reference photo
		for _, refFace := range refFaces {
			similarity := s.CompareFaces(faceDesc, refFace.Descriptor)

			// Extract user ID from reference photo path
			fileName := filepath.Base(refPath)
			userID := strings.Split(fileName, ".")[0] // Use first part before extension

			// Log all attempts with scores, not just matches
			log.Printf("Reference photo comparison - Path: %s, User: %s, Confidence: %.2f%%",
				refPath, userID, similarity*100)

			if similarity >= matchThreshold {
				// Update best match if this is the highest confidence so far
				if similarity > bestMatch.Confidence {
					bestMatch.UserID = userID
					bestMatch.Confidence = similarity
					bestMatch.PhotoPath = refPath
				}

				matches = append(matches, userID)
				confidences = append(confidences, similarity)
				break // One match per reference photo is enough
			}
		}
	}

	// Log the best match details
	if bestMatch.Confidence > -1 {
		log.Printf("Best match found - User ID: %s, Confidence: %.2f%%, Reference Photo: %s",
			bestMatch.UserID, bestMatch.Confidence*100, bestMatch.PhotoPath)
	} else {
		log.Printf("No matches found above threshold of %.2f%%", matchThreshold*100)
	}

	// Log summary of results
	log.Printf("Face matching complete - Found %d matches above threshold", len(matches))

	return matches, confidences, nil
}

// SerializeDescriptor converts a face descriptor to a byte array for storage
func (s *Service) SerializeDescriptor(desc face.Descriptor) []byte {
	// Each float32 is 4 bytes, and there are 128 values
	data := make([]byte, 128*4)
	for i, val := range desc {
		// Convert float32 to 4 bytes
		floatBytes := (*[4]byte)(unsafe.Pointer(&val))[:]
		// Copy to the right position in our byte array
		copy(data[i*4:i*4+4], floatBytes)
	}
	return data
}

// DeserializeDescriptor converts a byte array back to a face descriptor
func (s *Service) DeserializeDescriptor(data []byte) face.Descriptor {
	if len(data) != 128*4 {
		log.Printf("Warning: Invalid embedding data length: %d bytes", len(data))
		return face.Descriptor{}
	}

	var descriptor face.Descriptor
	for i := 0; i < 128; i++ {
		// Convert 4 bytes back to float32
		val := (*float32)(unsafe.Pointer(&data[i*4]))
		descriptor[i] = *val
	}
	return descriptor
}

// StoreEmbeddingForReferencePhoto computes and stores the embedding for a reference photo
func (s *Service) StoreEmbeddingForReferencePhoto(photoPath string, referencePhotoID string, userID string) error {
	// Ensure we have the full path to the reference photo
	fullPath := photoPath

	// If the path doesn't already include the uploads directory, prepend it
	if !strings.HasPrefix(fullPath, storage.RootDir) {
		// If path starts with "pfps/" or "/pfps/", make sure we get the full path
		if strings.HasPrefix(fullPath, "pfps/") || strings.HasPrefix(fullPath, "/pfps/") {
			baseName := filepath.Base(fullPath)
			fullPath = filepath.Join(storage.RootDir, "uploads", "pfps", baseName)
		} else if !strings.HasPrefix(fullPath, "/") {
			// If it doesn't start with "/", add it
			fullPath = filepath.Join(storage.RootDir, fullPath)
		} else {
			// If it starts with "/", prepend the root dir
			fullPath = filepath.Join(storage.RootDir, fullPath[1:])
		}
	}

	log.Printf("Looking for reference photo at: %s", fullPath)

	// Check if the file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("reference photo does not exist: %s", fullPath)
	}

	// Detect faces in the reference photo
	faces, err := s.rec.RecognizeFile(fullPath)
	if err != nil || len(faces) == 0 {
		return fmt.Errorf("failed to detect faces in reference photo: %v", err)
	}

	// Get the embedding of the first face (assuming reference photos have 1 clear face)
	descriptor := faces[0].Descriptor

	// Serialize the embedding
	embeddingData := s.SerializeDescriptor(descriptor)

	// Store in the database
	err = database.SaveFaceEmbedding(referencePhotoID, userID, embeddingData)
	if err != nil {
		return fmt.Errorf("failed to save face embedding: %v", err)
	}

	log.Printf("Stored face embedding for reference photo %s (user %s)", referencePhotoID, userID)
	return nil
}

// FindMatchesWithEmbeddings finds matching faces using pre-computed embeddings
func (s *Service) FindMatchesWithEmbeddings(faceDesc face.Descriptor) ([]string, []float64, error) {
	matches := make([]string, 0)
	confidences := make([]float64, 0)
	matchThreshold := 0.5 // Minimum similarity score to consider a match (lowered from 0.6)

	// Log total number of embeddings we're comparing against
	log.Printf("Loading all stored face embeddings for matching")

	// Get all stored embeddings
	embeddings, err := database.GetAllFaceEmbeddings()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve face embeddings: %v", err)
	}

	log.Printf("Comparing face against %d stored embeddings", len(embeddings))

	// Calculate and log input face descriptor statistics for comparison
	logEmbedding(faceDesc, "Input Face")

	// Track best match for logging
	var bestMatch struct {
		UserID           string
		ReferencePhotoID string
		Confidence       float64
	}
	bestMatch.Confidence = -1

	// Compare against each stored embedding
	for _, embedding := range embeddings {
		// Deserialize the stored embedding
		storedDesc := s.DeserializeDescriptor(embedding.EmbeddingVector)

		// Compare with the input face
		similarity := s.CompareFaces(faceDesc, storedDesc)

		// Log all comparisons
		log.Printf("Embedding comparison - User: %s, Reference Photo: %s, Confidence: %.2f%%",
			embedding.UserID, embedding.ReferencePhotoID, similarity*100)

		if similarity >= matchThreshold {
			// Update best match if this is the highest confidence so far
			if similarity > bestMatch.Confidence {
				bestMatch.UserID = embedding.UserID
				bestMatch.ReferencePhotoID = embedding.ReferencePhotoID
				bestMatch.Confidence = similarity
			}

			matches = append(matches, embedding.UserID)
			confidences = append(confidences, similarity)
		}
	}

	// Log the best match details
	if bestMatch.Confidence > -1 {
		log.Printf("Best match found - User ID: %s, Confidence: %.2f%%, Reference Photo: %s",
			bestMatch.UserID, bestMatch.Confidence*100, bestMatch.ReferencePhotoID)
	} else {
		log.Printf("No matches found above threshold of %.2f%%", matchThreshold*100)
	}

	// Log summary of results
	log.Printf("Face matching complete - Found %d matches above threshold", len(matches))

	return matches, confidences, nil
}

// FindMatchesWithEmbeddingsForUsers finds matching faces using pre-computed embeddings, but only for specific users
func (s *Service) FindMatchesWithEmbeddingsForUsers(faceDesc face.Descriptor, userIDs []string) ([]string, []float64, error) {
	matches := make([]string, 0)
	confidences := make([]float64, 0)
	matchThreshold := 0.5 // Minimum similarity score to consider a match

	// Log total number of users we're comparing against
	log.Printf("Loading stored face embeddings for matching against %d users", len(userIDs))

	// Get embeddings only for specified users
	embeddings, err := database.GetFaceEmbeddingsByUserIDs(userIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve face embeddings: %v", err)
	}

	log.Printf("Comparing face against %d stored embeddings from specified users", len(embeddings))

	// Calculate and log input face descriptor statistics for comparison
	logEmbedding(faceDesc, "Input Face")

	// Track best match for logging
	var bestMatch struct {
		UserID           string
		ReferencePhotoID string
		Confidence       float64
	}
	bestMatch.Confidence = -1

	// Compare against each stored embedding
	for _, embedding := range embeddings {
		// Deserialize the stored embedding
		storedDesc := s.DeserializeDescriptor(embedding.EmbeddingVector)

		// Compare with the input face
		similarity := s.CompareFaces(faceDesc, storedDesc)

		// Log all comparisons
		log.Printf("Embedding comparison - User: %s, Reference Photo: %s, Confidence: %.2f%%",
			embedding.UserID, embedding.ReferencePhotoID, similarity*100)

		if similarity >= matchThreshold {
			// Update best match if this is the highest confidence so far
			if similarity > bestMatch.Confidence {
				bestMatch.UserID = embedding.UserID
				bestMatch.ReferencePhotoID = embedding.ReferencePhotoID
				bestMatch.Confidence = similarity
			}

			matches = append(matches, embedding.UserID)
			confidences = append(confidences, similarity)
		}
	}

	// Log the best match details
	if bestMatch.Confidence > -1 {
		log.Printf("Best match found - User ID: %s, Confidence: %.2f%%, Reference Photo: %s",
			bestMatch.UserID, bestMatch.Confidence*100, bestMatch.ReferencePhotoID)
	} else {
		log.Printf("No matches found above threshold of %.2f%%", matchThreshold*100)
	}

	// Log summary of results
	log.Printf("Face matching complete - Found %d matches above threshold", len(matches))

	return matches, confidences, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
