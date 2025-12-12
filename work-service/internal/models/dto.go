package models

import "time"

// Data Transfer Objects

type CreateWorkRequest struct {
	StudentID    string `json:"student_id" validate:"required,uuid"`
	AssignmentID string `json:"assignment_id" validate:"required,uuid"`
}

type CreateWorkResponse struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	FileID    string    `json:"file_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type UploadWorkRequest struct {
	StudentID    string `json:"student_id" validate:"required,uuid"`
	AssignmentID string `json:"assignment_id" validate:"required,uuid"`
	FileContent  []byte `json:"-"` // Для внутреннего использования
	FileName     string `json:"file_name"`
}

type UpdateWorkStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=uploaded analyzing analyzed failed"`
}

type CreateAssignmentRequest struct {
	Title       string `json:"title" validate:"required,min=3,max=255"`
	Description string `json:"description" validate:"max=1000"`
}

type CreateStudentRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=255"`
	Email string `json:"email" validate:"required,email,max=255"`
}

type ReportResponse struct {
	WorkID          string     `json:"work_id"`
	StudentID       string     `json:"student_id"`
	AssignmentID    string     `json:"assignment_id"`
	Status          string     `json:"status"`
	PlagiarismFlag  bool       `json:"plagiarism_flag"`
	OriginalWorkID  *string    `json:"original_work_id,omitempty"`
	MatchPercentage int        `json:"match_percentage"`
	AnalyzedAt      *time.Time `json:"analyzed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type WorksResponse struct {
	Works []WorkWithDetails `json:"works"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}
