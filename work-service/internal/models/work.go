package models

import (
	"time"
)

type Work struct {
	ID           string    `json:"id" db:"id"`
	StudentID    string    `json:"student_id" db:"student_id"`
	AssignmentID string    `json:"assignment_id" db:"assignment_id"`
	FileID       string    `json:"file_id" db:"file_id"`
	Status       string    `json:"status" db:"status"` // uploaded, analyzing, analyzed, failed
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type WorkWithDetails struct {
	Work
	StudentName     string `json:"student_name" db:"student_name"`
	StudentEmail    string `json:"student_email" db:"student_email"`
	AssignmentTitle string `json:"assignment_title" db:"assignment_title"`
}

type WorkStatus string

const (
	WorkStatusUploaded  WorkStatus = "uploaded"
	WorkStatusAnalyzing WorkStatus = "analyzing"
	WorkStatusAnalyzed  WorkStatus = "analyzed"
	WorkStatusFailed    WorkStatus = "failed"
)

func (ws WorkStatus) String() string {
	return string(ws)
}

func IsValidWorkStatus(status string) bool {
	switch status {
	case "uploaded", "analyzing", "analyzed", "failed":
		return true
	default:
		return false
	}
}
