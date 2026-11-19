package models

import (
	"time"
)

// Media represents an uploaded media asset (image/document).
type Media struct {
	ID          string    `json:"id"`
	UploadedBy  string    `json:"uploaded_by"`
	Filename    string    `json:"filename"`
	StorageDisk string    `json:"storage_disk"`
	StorageKey  string    `json:"storage_key"`
	URL         string    `json:"url"`
	MimeType    string    `json:"mime_type"`
	Size        int64     `json:"size"`
	Width       *int32    `json:"width,omitempty"`
	Height      *int32    `json:"height,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
