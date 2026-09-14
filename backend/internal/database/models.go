package database

import (
	"time"
)

type User struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Avatar    string    `json:"avatar"`
	Token     string    `json:"token" gorm:"uniqueIndex"`
	TokenExp  time.Time `json:"tokenExp"`
	IsAdmin   bool      `json:"isAdmin" gorm:"default:false"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Event struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Location    string    `json:"location,omitempty"`
	Code        string    `json:"code" gorm:"uniqueIndex"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type EventParticipant struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	EventID   string    `json:"eventId" gorm:"index"`
	UserID    string    `json:"userId" gorm:"index"`
	Status    string    `json:"status"` // 'uploaded' | 'not_uploading' | 'will_upload'
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Photo struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	URL        string    `json:"url"`
	EventID    string    `json:"eventId" gorm:"index"`
	UploadedBy string    `json:"uploadedBy" gorm:"index"`
	UploadedAt time.Time `json:"uploadedAt"`
	HasFaces   bool      `json:"hasFaces" gorm:"default:false"`
	Processed  bool      `json:"processed" gorm:"default:false"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type PhotoPerson struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	PhotoID   string    `json:"photoId" gorm:"index"`
	UserID    string    `json:"userId" gorm:"index"`
	CreatedAt time.Time `json:"createdAt"`
}

type PhotoPersonMatch struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	PhotoID    string    `json:"photoId" gorm:"index"`
	UserID     string    `json:"userId" gorm:"index"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ReferencePhoto struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	UserID     string    `json:"userId" gorm:"index"`
	URL        string    `json:"url"`
	UploadedAt time.Time `json:"uploadedAt"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// FaceLocation stores the coordinates of a detected face in a photo
type FaceLocation struct {
	PhotoID string `db:"photo_id"`
	UserID  string `db:"user_id"`
	X       int    `db:"x"`
	Y       int    `db:"y"`
	Width   int    `db:"width"`
	Height  int    `db:"height"`
}

// FaceEmbedding stores the pre-computed face embedding vector for a reference photo
type FaceEmbedding struct {
	ID               string    `json:"id" db:"id" gorm:"primaryKey"`
	ReferencePhotoID string    `json:"referencePhotoId" db:"reference_photo_id" gorm:"index"`
	UserID           string    `json:"userId" db:"user_id" gorm:"index"`
	EmbeddingVector  []byte    `json:"embeddingVector" db:"embedding_vector"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
}
