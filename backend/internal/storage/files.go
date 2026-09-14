package storage

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	pigo "github.com/esimov/pigo/core"
	"github.com/google/uuid"
)

const (
	// Store paths relative to backend root
	RootDir     = "."              // Root directory for all storage
	UploadsDir  = "uploads"        // Directory for uploaded photos
	EventsDir   = "uploads/events" // Directory for event photos
	PfpsDir     = "uploads/pfps"   // Directory for profile pictures
	TmpDir      = "tmp/faces"      // Directory for face crops
	maxWidth    = 1200             // Maximum width for reference photos
	maxHeight   = 1200             // Maximum height for reference photos
	pfpSize     = 400              // Size for profile pictures (square)
	jpegQuality = 85               // JPEG quality for optimized images
)

var (
	validMimeTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/heic": true,
		"image/heif": true,
	}
	classifier *pigo.Pigo
)

// FaceScanInfo tracks when a face scan was created
type FaceScanInfo struct {
	CreatedAt time.Time
	Cleaned   bool
}

var (
	faceScanTimestamps = make(map[string]*FaceScanInfo)
	faceScanMutex      sync.RWMutex
)

func init() {
	// Create required directories if they don't exist
	dirs := []string{
		UploadsDir,
		EventsDir,
		PfpsDir,
		TmpDir,
	}

	for _, dir := range dirs {
		absPath := path.Join(RootDir, dir)
		if err := os.MkdirAll(absPath, 0755); err != nil {
			log.Printf("Failed to create directory %s: %v", absPath, err)
		}
	}

	// Initialize face detection classifier with absolute path
	cascadeFile, err := os.ReadFile(path.Join(RootDir, "cascade/facefinder"))
	if err != nil {
		log.Printf("Warning: Failed to load Pigo cascade file: %v", err)
		return
	}

	classifier, err = pigo.NewPigo().Unpack(cascadeFile)
	if err != nil {
		log.Printf("Warning: Failed to initialize Pigo classifier: %v", err)
		return
	}
}

// SaveUploadedFile saves and optimizes a file to the local filesystem and returns its path
func SaveUploadedFile(file *multipart.FileHeader, eventID string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer src.Close()

	// Check file extension first
	filename := file.Filename
	ext := strings.ToLower(filepath.Ext(filename))
	isHeic := ext == ".heic" || ext == ".heif"

	// Check file type
	buffer := make([]byte, 512)
	if _, err := src.Read(buffer); err != nil {
		return "", fmt.Errorf("failed to read file header: %v", err)
	}
	if _, err := src.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to reset file reader: %v", err)
	}

	contentType := http.DetectContentType(buffer)

	// Accept the file if either the MIME type is valid OR it has a .heic/.heif extension
	if !validMimeTypes[contentType] && !isHeic {
		return "", fmt.Errorf("invalid file type: %s", contentType)
	}

	// Read entire file data to process HEIC if needed
	fileData, err := io.ReadAll(src)
	if err != nil {
		return "", fmt.Errorf("failed to read file data: %v", err)
	}

	var img image.Image

	// Special handling for HEIC/HEIF format - detect by extension or content type
	if isHeic || contentType == "image/heic" || contentType == "image/heif" {
		tempFile, err := os.CreateTemp("", "heic_conversion_*.heic")
		if err != nil {
			log.Printf("ERROR: Failed to create temporary file for HEIC conversion: %v", err)
			return "", fmt.Errorf("failed to create temporary file for HEIC conversion: %v", err)
		}
		defer os.Remove(tempFile.Name())

		if _, err := tempFile.Write(fileData); err != nil {
			log.Printf("ERROR: Failed to write HEIC data to temporary file: %v", err)
			return "", fmt.Errorf("failed to write HEIC data to temporary file: %v", err)
		}
		tempFile.Close()

		convertedFile := tempFile.Name() + ".jpg"

		// Check if heif-convert is available
		_, err = exec.LookPath("heif-convert")
		if err == nil {
			cmd := exec.Command("heif-convert", tempFile.Name(), convertedFile)
			var output bytes.Buffer
			var errOutput bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &errOutput
			if err := cmd.Run(); err != nil {
				log.Printf("ERROR: heif-convert failed: %v, stdout: %s, stderr: %s", err, output.String(), errOutput.String())
				return "", fmt.Errorf("failed to convert HEIC to JPEG: %v - %s", err, errOutput.String())
			}

			convertedData, err := os.ReadFile(convertedFile)
			if err != nil {
				log.Printf("ERROR: Failed to read converted JPEG file: %v", err)
				return "", fmt.Errorf("failed to read converted JPEG file: %v", err)
			}
			defer os.Remove(convertedFile)

			img, _, err = image.Decode(bytes.NewReader(convertedData))
			if err != nil {
				log.Printf("ERROR: Failed to decode converted JPEG image: %v", err)
				return "", fmt.Errorf("failed to decode converted JPEG image: %v", err)
			}
		} else {
			// Fallback: try direct decoding in case a decoder is registered
			img, _, err = image.Decode(bytes.NewReader(fileData))
			if err != nil {
				log.Printf("ERROR: Failed to decode HEIC image directly: %v", err)
				return "", fmt.Errorf("failed to decode HEIC image (and heif-convert not available): %v", err)
			}
		}
	} else {
		// For other formats use standard image decoder
		img, _, err = image.Decode(bytes.NewReader(fileData))
		if err != nil {
			log.Printf("ERROR: Failed to decode image: %v", err)
			return "", fmt.Errorf("failed to decode image: %v", err)
		}
	}

	// Resize and optimize image
	resized := imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)

	// Generate unique filename with proper extension
	filename = uuid.New().String() + ".jpg" // Always save as JPEG for consistency

	// If eventID is provided, store in event-specific directory
	var uploadPath string
	if eventID != "" {
		eventPath := path.Join(RootDir, EventsDir, eventID)
		if err := os.MkdirAll(eventPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create event directory: %v", err)
		}
		uploadPath = path.Join(eventPath, filename)
	} else {
		// For non-event photos (like profile pictures), use uploads dir
		uploadPath = path.Join(RootDir, UploadsDir, filename)
		if err := os.MkdirAll(path.Dir(uploadPath), 0755); err != nil {
			return "", fmt.Errorf("failed to create uploads directory: %v", err)
		}
	}

	// Save the optimized image first
	err = imaging.Save(resized, uploadPath, imaging.JPEGQuality(jpegQuality))
	if err != nil {
		return "", fmt.Errorf("failed to save optimized image: %v", err)
	}

	// Return relative path for database storage
	relativePath := strings.TrimPrefix(uploadPath, path.Join(RootDir, ""))
	return relativePath, nil
}

// GetFaceCrops returns the paths to all detected face crops for a given photo
func GetFaceCrops(originalPath string) ([]string, error) {
	img, err := imaging.Open(path.Join(RootDir, originalPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %v", err)
	}

	// Convert image to grayscale for face detection
	grayImg := imaging.Grayscale(img)

	// Prepare image for Pigo
	pixels := pigo.RgbToGrayscale(grayImg)
	imgParams := &pigo.ImageParams{
		Pixels: pixels,
		Rows:   grayImg.Bounds().Dy(),
		Cols:   grayImg.Bounds().Dx(),
		Dim:    grayImg.Bounds().Dx(),
	}

	// Initialize Pigo classifier if not already initialized
	if classifier == nil {
		cascadeFile, err := os.ReadFile(path.Join(RootDir, "cascade/facefinder"))
		if err != nil {
			return nil, fmt.Errorf("failed to load Pigo cascade file: %v", err)
		}
		classifier, err = pigo.NewPigo().Unpack(cascadeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Pigo classifier: %v", err)
		}
	}

	// Run face detection
	cascadeParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     imgParams.Dim,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}

	faces := classifier.RunCascade(cascadeParams, 0.0)
	faces = classifier.ClusterDetections(faces, 0.1)

	// Extract the base filename without extension
	baseID := strings.TrimSuffix(filepath.Base(originalPath), filepath.Ext(originalPath))
	facePaths := make([]string, 0, len(faces))

	// Save face crops to temporary directory
	for i, face := range faces {
		scale := float64(face.Scale) / 100.0
		rect := image.Rect(
			face.Col-int(scale*float64(face.Scale)),
			face.Row-int(scale*float64(face.Scale)),
			face.Col+int(scale*float64(face.Scale)),
			face.Row+int(scale*float64(face.Scale)),
		)

		// Crop and save the face
		faceCrop := imaging.Crop(img, rect)
		// Resize to standard profile picture size
		faceCrop = imaging.Fit(faceCrop, pfpSize, pfpSize, imaging.Lanczos)
		cropFilename := fmt.Sprintf("%s_face_%d.jpg", baseID, i+1)
		cropPath := path.Join(RootDir, TmpDir, cropFilename)

		if err := imaging.Save(faceCrop, cropPath, imaging.JPEGQuality(jpegQuality)); err != nil {
			log.Printf("Warning: Failed to save face crop %d: %v", i+1, err)
			continue
		}

		facePaths = append(facePaths, cropPath)
	}

	return facePaths, nil
}

// MoveFaceToPfps moves a face crop from temporary storage to the profile pictures directory
func MoveFaceToPfps(tmpPath string) (string, error) {
	// Ensure uploads/pfps directory exists
	pfpsPath := path.Join(RootDir, PfpsDir)
	if err := os.MkdirAll(pfpsPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create profile pictures directory: %v", err)
	}

	// Read the face crop from temp directory
	fullTmpPath := path.Join(RootDir, tmpPath)
	img, err := imaging.Open(fullTmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to open face crop at %s: %v", tmpPath, err)
	}

	// Generate a new filename
	filename := uuid.New().String() + ".jpg"

	// Save to profile pictures directory
	fullPfpPath := path.Join(pfpsPath, filename)
	if err := imaging.Save(img, fullPfpPath, imaging.JPEGQuality(jpegQuality)); err != nil {
		return "", fmt.Errorf("failed to save profile picture to %s: %v", fullPfpPath, err)
	}

	// Delete the temporary file
	if err := os.Remove(fullTmpPath); err != nil {
		// Log but don't fail if we can't delete the temp file
		log.Printf("Warning: Failed to delete temporary face crop %s: %v", tmpPath, err)
	}

	// Return path that will work with our static file serving
	return path.Join("pfps", filename), nil
}

// CleanupFaceCrops removes all face crops associated with a photo ID
func CleanupFaceCrops(photoID string) error {
	if photoID == "" {
		return nil
	}

	pattern := path.Join(RootDir, TmpDir, photoID+"_face_*.jpg")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list face crops: %v", err)
	}

	log.Printf("Cleaning up face crops for photo %s, found %d files", photoID, len(matches))

	for _, match := range matches {
		log.Printf("Deleting face crop: %s", match)
		if err := os.Remove(match); err != nil {
			log.Printf("Warning: Failed to delete face crop %s: %v", match, err)
			// Continue trying to delete other files even if one fails
		}
	}

	return nil
}

// TrackFaceScan records when a face scan was created and starts a cleanup timer
func TrackFaceScan(photoID string) {
	faceScanMutex.Lock()
	faceScanTimestamps[photoID] = &FaceScanInfo{
		CreatedAt: time.Now(),
		Cleaned:   false,
	}
	faceScanMutex.Unlock()

	// Start cleanup timer
	go func() {
		time.Sleep(60 * time.Second)
		CleanupExpiredFaceScan(photoID)
	}()
}

// CleanupExpiredFaceScan removes face scans for a photo if they haven't been cleaned up already
func CleanupExpiredFaceScan(photoID string) {
	faceScanMutex.Lock()
	defer faceScanMutex.Unlock()

	info, exists := faceScanTimestamps[photoID]
	if !exists || info.Cleaned {
		return
	}

	// Mark as cleaned and remove from tracking
	info.Cleaned = true
	delete(faceScanTimestamps, photoID)

	// Clean up the face scans
	if err := CleanupFaceCrops(photoID); err != nil {
		log.Printf("Warning: Failed to clean up expired face scans for photo %s: %v", photoID, err)
	} else {
		log.Printf("Cleaned up expired face scans for photo %s", photoID)
	}
}

// MarkFaceScansProcessed marks a photo's face scans as processed so they won't be cleaned up by timeout
func MarkFaceScansProcessed(photoID string) {
	faceScanMutex.Lock()
	defer faceScanMutex.Unlock()

	info, exists := faceScanTimestamps[photoID]
	if !exists {
		return
	}

	info.Cleaned = true
	delete(faceScanTimestamps, photoID)
}

func DeleteFile(filePath string) error {
	if filePath == "" {
		return nil
	}

	filePath = strings.TrimPrefix(filePath, "/")
	return os.Remove(path.Join(RootDir, filePath))
}

// DeleteEventFolder removes the event's folder and all its contents
func DeleteEventFolder(eventID string) error {
	if eventID == "" {
		return nil
	}

	eventPath := path.Join(RootDir, EventsDir, eventID)
	return os.RemoveAll(eventPath)
}
