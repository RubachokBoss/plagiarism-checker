package models

import (
	"encoding/json"
	"mime/multipart"
	"time"
)

// Data Transfer Objects

type UploadFileRequest struct {
	File       *multipart.FileHeader `json:"-" form:"file"`
	UploadedBy string                `json:"uploaded_by,omitempty" form:"uploaded_by"`
	Metadata   json.RawMessage       `json:"metadata,omitempty" form:"metadata"`
}

type UploadFileResponse struct {
	FileID     string          `json:"file_id"`
	FileName   string          `json:"file_name"`
	FileSize   int64           `json:"file_size"`
	Hash       string          `json:"hash"`
	MimeType   string          `json:"mime_type"`
	UploadedAt time.Time       `json:"uploaded_at"`
	StorageURL string          `json:"storage_url,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type FileInfoResponse struct {
	FileID         string          `json:"file_id"`
	OriginalName   string          `json:"original_name"`
	FileSize       int64           `json:"file_size"`
	MimeType       string          `json:"mime_type"`
	Hash           string          `json:"hash"`
	UploadStatus   string          `json:"upload_status"`
	UploadedAt     time.Time       `json:"uploaded_at"`
	AccessCount    int             `json:"access_count"`
	LastAccessedAt *time.Time      `json:"last_accessed_at,omitempty"`
	StorageURL     string          `json:"storage_url,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type DownloadFileResponse struct {
	Content     []byte `json:"-"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
}

type DeleteFileResponse struct {
	FileID  string `json:"file_id"`
	Deleted bool   `json:"deleted"`
	Message string `json:"message,omitempty"`
}

type AssociateFileRequest struct {
	FileID          string `json:"file_id" validate:"required,uuid"`
	EntityType      string `json:"entity_type" validate:"required"`
	EntityID        string `json:"entity_id" validate:"required"`
	AssociationType string `json:"association_type" validate:"required"`
}

type StorageInfo struct {
	Provider   string `json:"provider"`
	BucketName string `json:"bucket_name"`
	Region     string `json:"region"`
	UsedSpace  int64  `json:"used_space"`
	FileCount  int64  `json:"file_count"`
}
