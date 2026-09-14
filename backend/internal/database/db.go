package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"partipix/backend/internal/storage"
)

var DB *gorm.DB

func InitDB() error {
	var err error
	DB, err = gorm.Open(sqlite.Open("partipix.db"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Auto-migrate the schema for GORM models
	err = DB.AutoMigrate(&User{}, &Event{}, &EventParticipant{}, &Photo{}, &PhotoPerson{}, &ReferencePhoto{}, &PhotoPersonMatch{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	// Get raw SQL database connection
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database: %v", err)
	}

	// Run SQL migrations
	migrationFiles := []string{
		"internal/database/migrations/001_create_face_locations.sql",
		"internal/database/migrations/002_create_face_embeddings.sql",
	}

	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %v", file, err)
		}

		_, err = sqlDB.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to run migration %s: %v", file, err)
		}
	}

	return nil
}

// Transaction executes a function within a database transaction
func Transaction(fc func(*gorm.DB) error) error {
	return DB.Transaction(fc)
}

// UserRepository
func CreateUser(user *User) error {
	user.ID = uuid.New().String()
	user.Token = uuid.New().String() // Set initial token
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return DB.Create(user).Error
}

func GetUserByID(id string) (*User, error) {
	var user User
	err := DB.First(&user, "id = ?", id).Error
	return &user, err
}

func GetUserByEmail(email string) (*User, error) {
	var user User
	err := DB.First(&user, "email = ?", email).Error
	return &user, err
}

func GetUserByToken(token string) (*User, error) {
	var user User
	err := DB.First(&user, "token = ?", token).Error
	return &user, err
}

func UpdateUser(user *User) error {
	user.UpdatedAt = time.Now()
	return DB.Save(user).Error
}

// EventRepository
func CreateEvent(event *Event) error {
	event.ID = uuid.New().String()
	event.Code = generateEventCode()
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()
	return DB.Create(event).Error
}

func GetEventByID(id string) (*Event, error) {
	var event Event
	err := DB.First(&event, "id = ?", id).Error
	return &event, err
}

func GetEventsByUserID(userID string) ([]Event, error) {
	var events []Event
	err := DB.Joins("JOIN event_participants ON events.id = event_participants.event_id").
		Where("event_participants.user_id = ?", userID).
		Find(&events).Error
	return events, err
}

func GetEventByCode(code string) (*Event, error) {
	var event Event
	// Convert the input code to uppercase to ensure case-insensitive matching
	code = strings.ToUpper(code)
	err := DB.First(&event, "UPPER(code) = ?", code).Error
	return &event, err
}

func ListEvents() ([]Event, error) {
	var events []Event
	err := DB.Find(&events).Error
	return events, err
}

func UpdateEvent(event *Event) error {
	event.UpdatedAt = time.Now()
	return DB.Save(event).Error
}

func DeleteEvent(id string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Get all photos first so we can delete the files
		var photos []Photo
		if err := tx.Where("event_id = ?", id).Find(&photos).Error; err != nil {
			return err
		}

		// Delete photo files
		for _, photo := range photos {
			if err := storage.DeleteFile(photo.URL); err != nil {
				log.Printf("Warning: Failed to delete photo file %s: %v", photo.URL, err)
				// Continue deleting other files even if one fails
			}
		}

		// Delete all photos from database
		if err := tx.Where("event_id = ?", id).Delete(&Photo{}).Error; err != nil {
			return err
		}

		// Delete all event participants
		if err := tx.Where("event_id = ?", id).Delete(&EventParticipant{}).Error; err != nil {
			return err
		}

		// Delete the event record
		if err := tx.Delete(&Event{}, "id = ?", id).Error; err != nil {
			return err
		}

		// Delete the event folder
		if err := storage.DeleteEventFolder(id); err != nil {
			log.Printf("Warning: Failed to delete event folder for event %s: %v", id, err)
			// Continue anyway since the database records are gone
		}

		return nil
	})
}

// EventParticipantRepository
func AddEventParticipant(participant *EventParticipant) error {
	participant.ID = uuid.New().String()
	participant.CreatedAt = time.Now()
	participant.UpdatedAt = time.Now()
	return DB.Create(participant).Error
}

func GetEventParticipants(eventID string) ([]EventParticipant, error) {
	var participants []EventParticipant
	err := DB.Where("event_id = ?", eventID).Find(&participants).Error
	return participants, err
}

// GetEventParticipantsWithUsers returns both participants and a map of user IDs to users
func GetEventParticipantsWithUsers(eventID string) ([]EventParticipant, map[string]User, error) {
	participants, err := GetEventParticipants(eventID)
	if err != nil {
		return nil, nil, err
	}

	// Create a set of unique user IDs
	userIDs := make(map[string]bool)
	for _, p := range participants {
		userIDs[p.UserID] = true
	}

	// Get users for all participants
	users := make(map[string]User)
	for userID := range userIDs {
		user, err := GetUserByID(userID)
		if err != nil {
			continue // Skip users that can't be found
		}
		users[userID] = *user
	}

	return participants, users, nil
}

// UpdateEventParticipantStatus updates the status of a participant in an event
func UpdateEventParticipantStatus(participant *EventParticipant) error {
	return DB.Model(&EventParticipant{}).
		Where("event_id = ? AND user_id = ?", participant.EventID, participant.UserID).
		Updates(map[string]interface{}{
			"status":     participant.Status,
			"updated_at": time.Now(),
		}).Error
}

// PhotoRepository
func CreatePhoto(photo *Photo) error {
	photo.ID = uuid.New().String()
	photo.CreatedAt = time.Now()
	photo.UpdatedAt = time.Now()
	return DB.Create(photo).Error
}

func GetPhotosByEventID(eventID string) ([]Photo, error) {
	var photos []Photo
	err := DB.Where("event_id = ?", eventID).Find(&photos).Error
	return photos, err
}

func GetPhotosByUserID(userID string) ([]Photo, error) {
	var photos []Photo
	err := DB.Where("uploaded_by = ?", userID).Find(&photos).Error
	return photos, err
}

func GetPhotosWithUser(userID string) ([]Photo, error) {
	var photos []Photo
	err := DB.Joins("JOIN photo_people ON photos.id = photo_people.photo_id").
		Where("photo_people.user_id = ?", userID).
		Find(&photos).Error
	return photos, err
}

// Get all photos in an event, separated by uploader
func GetEventPhotosByUploader(eventID, currentUserID string) (map[string][]Photo, error) {
	var photos []Photo
	err := DB.Where("event_id = ?", eventID).Find(&photos).Error
	if err != nil {
		return nil, err
	}

	photosByUploader := make(map[string][]Photo)
	for _, photo := range photos {
		if photo.UploadedBy == currentUserID {
			photosByUploader["uploaded"] = append(photosByUploader["uploaded"], photo)
		} else {
			photosByUploader["others"] = append(photosByUploader["others"], photo)
		}
	}
	return photosByUploader, nil
}

// Mark a photo as having faces detected
func UpdatePhotoFacesStatus(photoID string, hasFaces bool) error {
	return DB.Model(&Photo{}).
		Where("id = ?", photoID).
		Updates(map[string]interface{}{
			"has_faces":  hasFaces,
			"processed":  true,
			"updated_at": time.Now(),
		}).Error
}

// Get all photos where a user appears and is tagged
func GetPhotosPersonAppearsIn(userID string) ([]Photo, error) {
	var photos []Photo
	err := DB.Joins("JOIN photo_people ON photos.id = photo_people.photo_id").
		Where("photo_people.user_id = ?", userID).
		Find(&photos).Error
	return photos, err
}

// GetPhotoByID gets a photo by its ID
func GetPhotoByID(id string) (*Photo, error) {
	var photo Photo
	err := DB.First(&photo, "id = ?", id).Error
	return &photo, err
}

// PhotoPersonRepository
func AddPhotoPerson(photoPerson *PhotoPerson) error {
	photoPerson.ID = uuid.New().String()
	photoPerson.CreatedAt = time.Now()
	return DB.Create(photoPerson).Error
}

func GetPhotoPeople(photoID string) ([]string, error) {
	var photoPeople []PhotoPerson
	err := DB.Where("photo_id = ?", photoID).Find(&photoPeople).Error
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, len(photoPeople))
	for i, pp := range photoPeople {
		userIDs[i] = pp.UserID
	}
	return userIDs, nil
}

func GetPhotosWithPerson(userID string) ([]Photo, error) {
	var photos []Photo
	err := DB.Joins("JOIN photo_people ON photos.id = photo_people.photo_id").
		Where("photo_people.user_id = ?", userID).
		Find(&photos).Error
	return photos, err
}

// AddPhotoPersonMatch adds a face match record with confidence score
func AddPhotoPersonMatch(match *PhotoPersonMatch) error {
	match.ID = uuid.New().String()
	match.CreatedAt = time.Now()
	return DB.Create(match).Error
}

// GetPhotoMatches gets all face matches with confidence scores for a photo
func GetPhotoMatches(photoID string) ([]PhotoPersonMatch, error) {
	var matches []PhotoPersonMatch
	err := DB.Where("photo_id = ?", photoID).Find(&matches).Error
	return matches, err
}

// GetUsersFromMatches gets all user details for face matches
func GetUsersFromMatches(matches []PhotoPersonMatch) (map[string]User, error) {
	if len(matches) == 0 {
		return make(map[string]User), nil
	}

	userIDs := make([]string, len(matches))
	for i, match := range matches {
		userIDs[i] = match.UserID
	}

	var users []User
	err := DB.Where("id IN ?", userIDs).Find(&users).Error
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]User)
	for _, user := range users {
		userMap[user.ID] = user
	}

	return userMap, nil
}

// ReferencePhotoRepository
func CreateReferencePhoto(photo *ReferencePhoto) error {
	photo.ID = uuid.New().String()
	photo.CreatedAt = time.Now()
	photo.UpdatedAt = time.Now()
	photo.UploadedAt = time.Now()
	return DB.Create(photo).Error
}

func GetReferencePhotosByUserID(userID string) ([]ReferencePhoto, error) {
	var photos []ReferencePhoto
	err := DB.Where("user_id = ?", userID).Find(&photos).Error
	return photos, err
}

func GetReferencePhotoByID(id string) (*ReferencePhoto, error) {
	var photo ReferencePhoto
	err := DB.First(&photo, "id = ?", id).Error
	return &photo, err
}

func DeleteReferencePhoto(id string) error {
	return DB.Delete(&ReferencePhoto{}, "id = ?", id).Error
}

// SaveFaceLocation stores the location of a detected face in a photo
func SaveFaceLocation(face FaceLocation) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	_, err = sqlDB.Exec(`
		INSERT INTO face_locations (photo_id, user_id, x, y, width, height)
		VALUES (?, ?, ?, ?, ?, ?)`,
		face.PhotoID, face.UserID, face.X, face.Y, face.Width, face.Height)
	return err
}

// GetFaceLocation retrieves the location of a detected face in a photo
func GetFaceLocation(photoID, userID string) (FaceLocation, error) {
	var face FaceLocation
	sqlDB, err := DB.DB()
	if err != nil {
		return face, err
	}
	err = sqlDB.QueryRow(`
		SELECT photo_id, user_id, x, y, width, height 
		FROM face_locations 
		WHERE photo_id = ? AND user_id = ?`,
		photoID, userID).Scan(&face.PhotoID, &face.UserID, &face.X, &face.Y, &face.Width, &face.Height)
	return face, err
}

// SaveFaceEmbedding stores a face embedding in the database
func SaveFaceEmbedding(referencePhotoID string, userID string, embeddingVector []byte) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	embedding := FaceEmbedding{
		ID:               uuid.New().String(),
		ReferencePhotoID: referencePhotoID,
		UserID:           userID,
		EmbeddingVector:  embeddingVector,
		CreatedAt:        time.Now(),
	}

	_, err = db.NamedExec(`
		INSERT INTO face_embeddings (id, reference_photo_id, user_id, embedding_vector, created_at)
		VALUES (:id, :reference_photo_id, :user_id, :embedding_vector, :created_at)
	`, embedding)

	return err
}

// GetFaceEmbeddingsByUserID returns all face embeddings for a given user
func GetFaceEmbeddingsByUserID(userID string) ([]FaceEmbedding, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	var embeddings []FaceEmbedding
	err = db.Select(&embeddings, `
		SELECT id, reference_photo_id, user_id, embedding_vector, created_at 
		FROM face_embeddings 
		WHERE user_id = ?
	`, userID)

	return embeddings, err
}

// GetFaceEmbeddingsByUserIDs returns all face embeddings for a given set of user IDs
func GetFaceEmbeddingsByUserIDs(userIDs []string) ([]FaceEmbedding, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, reference_photo_id, user_id, embedding_vector, created_at 
		FROM face_embeddings 
		WHERE user_id IN (?)
	`
	query, args, err := sqlx.In(query, userIDs)
	if err != nil {
		return nil, err
	}

	var embeddings []FaceEmbedding
	err = db.Select(&embeddings, query, args...)
	return embeddings, err
}

// GetAllFaceEmbeddings returns all face embeddings in the database
func GetAllFaceEmbeddings() ([]FaceEmbedding, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	var embeddings []FaceEmbedding
	err = db.Select(&embeddings, `
		SELECT id, reference_photo_id, user_id, embedding_vector, created_at 
		FROM face_embeddings
	`)

	return embeddings, err
}

// DeleteFaceEmbeddingsByReferencePhotoID deletes embeddings associated with a reference photo
func DeleteFaceEmbeddingsByReferencePhotoID(referencePhotoID string) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(`DELETE FROM face_embeddings WHERE reference_photo_id = ?`, referencePhotoID)
	return err
}

// Helper function to generate a random event code
func generateEventCode() string {
	return strings.ToUpper(uuid.New().String()[:8])
}

// Admin functions
func CountUsers() (int64, error) {
	var count int64
	err := DB.Model(&User{}).Count(&count).Error
	return count, err
}

func CountEvents() (int64, error) {
	var count int64
	err := DB.Model(&Event{}).Count(&count).Error
	return count, err
}

func CountPhotos() (int64, error) {
	var count int64
	err := DB.Model(&Photo{}).Count(&count).Error
	return count, err
}

func CountActiveEvents() (int64, error) {
	var count int64
	err := DB.Model(&Event{}).Where("date > ?", time.Now()).Count(&count).Error
	return count, err
}

func ListUsers() ([]User, error) {
	var users []User
	err := DB.Find(&users).Error
	return users, err
}

func DeleteUser(id string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Get reference photos first so we can delete the files
		var referencePhotos []ReferencePhoto
		if err := tx.Where("user_id = ?", id).Find(&referencePhotos).Error; err != nil {
			return err
		}

		// Delete reference photo files
		for _, photo := range referencePhotos {
			if err := storage.DeleteFile(photo.URL); err != nil {
				log.Printf("Warning: Failed to delete reference photo file %s: %v", photo.URL, err)
				// Continue deleting other files even if one fails
			}
		}

		// Delete reference photos from database
		if err := tx.Where("user_id = ?", id).Delete(&ReferencePhoto{}).Error; err != nil {
			return err
		}

		// Get user's uploaded photos so we can delete the files
		var photos []Photo
		if err := tx.Where("uploaded_by = ?", id).Find(&photos).Error; err != nil {
			return err
		}

		// Delete photo files
		for _, photo := range photos {
			if err := storage.DeleteFile(photo.URL); err != nil {
				log.Printf("Warning: Failed to delete photo file %s: %v", photo.URL, err)
				// Continue deleting other files even if one fails
			}
		}

		// Delete user's photos from database
		if err := tx.Where("uploaded_by = ?", id).Delete(&Photo{}).Error; err != nil {
			return err
		}

		// Delete user's event participations
		if err := tx.Where("user_id = ?", id).Delete(&EventParticipant{}).Error; err != nil {
			return err
		}

		// Delete user's photo tags
		if err := tx.Where("user_id = ?", id).Delete(&PhotoPerson{}).Error; err != nil {
			return err
		}

		// Delete the user
		return tx.Delete(&User{}, "id = ?", id).Error
	})
}

// Helper function to get the underlying *sql.DB from GORM
func getDB() (*sqlx.DB, error) {
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}
	return sqlx.NewDb(sqlDB, "sqlite"), nil
}
