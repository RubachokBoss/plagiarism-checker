package models

import (
	"encoding/json"
	"time"
)

type FileMetadata struct {
	ID              string          `json:"id" db:"id"`
	OriginalName    string          `json:"original_name" db:"original_name"`
	FileName        string          `json:"file_name" db:"file_name"`
	FileExtension   string          `json:"file_extension" db:"file_extension"`
	FileSize        int64           `json:"file_size" db:"file_size"`
	MimeType        string          `json:"mime_type" db:"mime_type"`
	Hash            string          `json:"hash" db:"hash"`
	StorageProvider string          `json:"storage_provider" db:"storage_provider"`
	StorageBucket   string          `json:"storage_bucket" db:"storage_bucket"`
	StoragePath     string          `json:"storage_path" db:"storage_path"`
	StorageURL      string          `json:"storage_url,omitempty" db:"storage_url"`
	UploadStatus    string          `json:"upload_status" db:"upload_status"`
	UploadedBy      string          `json:"uploaded_by,omitempty" db:"uploaded_by"`
	UploadedAt      time.Time       `json:"uploaded_at" db:"uploaded_at"`
	AccessCount     int             `json:"access_count" db:"access_count"`
	LastAccessedAt  *time.Time      `json:"last_accessed_at,omitempty" db:"last_accessed_at"`
	Metadata        json.RawMessage `json:"metadata,omitempty" db:"metadata"`
}

type FileUploadStatus string

const (
	FileStatusUploaded   FileUploadStatus = "uploaded"
	FileStatusProcessing FileUploadStatus = "processing"
	FileStatusFailed     FileUploadStatus = "failed"
	FileStatusDeleted    FileUploadStatus = "deleted"
)

func (fs FileUploadStatus) String() string {
	return string(fs)
}

type FileAssociation struct {
	ID              string    `json:"id" db:"id"`
	FileID          string    `json:"file_id" db:"file_id"`
	EntityType      string    `json:"entity_type" db:"entity_type"`
	EntityID        string    `json:"entity_id" db:"entity_id"`
	AssociationType string    `json:"association_type" db:"association_type"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type FileStats struct {
	TotalFiles      int64               `json:"total_files"`
	TotalSize       int64               `json:"total_size"`
	UploadedToday   int64               `json:"uploaded_today"`
	AverageFileSize int64               `json:"average_file_size"`
	TopExtensions   []FileExtensionStat `json:"top_extensions"`
}

type FileExtensionStat struct {
	Extension string `json:"extension"`
	Count     int64  `json:"count"`
	TotalSize int64  `json:"total_size"`
}
