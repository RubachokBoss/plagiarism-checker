package models

import "time"

// Data Transfer Objects

type CreateReportRequest struct {
	WorkID       string `json:"work_id" validate:"required"`
	FileID       string `json:"file_id" validate:"required"`
	AssignmentID string `json:"assignment_id" validate:"required"`
	StudentID    string `json:"student_id" validate:"required"`
}

type UpdateReportRequest struct {
	Status          string                 `json:"status" validate:"required,oneof=pending processing completed failed"`
	PlagiarismFlag  bool                   `json:"plagiarism_flag"`
	OriginalWorkID  *string                `json:"original_work_id,omitempty"`
	MatchPercentage int                    `json:"match_percentage" validate:"min=0,max=100"`
	Details         map[string]interface{} `json:"details,omitempty"`
}

type GetReportResponse struct {
	ReportID           string                 `json:"report_id"`
	WorkID             string                 `json:"work_id"`
	FileID             string                 `json:"file_id"`
	AssignmentID       string                 `json:"assignment_id"`
	StudentID          string                 `json:"student_id"`
	Status             string                 `json:"status"`
	PlagiarismFlag     bool                   `json:"plagiarism_flag"`
	OriginalWorkID     *string                `json:"original_work_id,omitempty"`
	MatchPercentage    int                    `json:"match_percentage"`
	FileHash           string                 `json:"file_hash,omitempty"`
	Details            map[string]interface{} `json:"details,omitempty"`
	ProcessingTimeMs   *int                   `json:"processing_time_ms,omitempty"`
	ComparedFilesCount int                    `json:"compared_files_count"`
	CreatedAt          time.Time              `json:"created_at"`
	StartedAt          *time.Time             `json:"started_at,omitempty"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
}

type GetAssignmentStatsResponse struct {
	AssignmentID       string                 `json:"assignment_id"`
	TotalWorks         int                    `json:"total_works"`
	AnalyzedWorks      int                    `json:"analyzed_works"`
	PlagiarizedWorks   int                    `json:"plagiarized_works"`
	AvgMatchPercentage float64                `json:"avg_match_percentage"`
	Reports            []GetReportResponse    `json:"reports,omitempty"`
	Statistics         map[string]interface{} `json:"statistics,omitempty"`
	LastAnalyzedAt     *time.Time             `json:"last_analyzed_at,omitempty"`
}

type GetStudentStatsResponse struct {
	StudentID          string                 `json:"student_id"`
	TotalWorks         int                    `json:"total_works"`
	AnalyzedWorks      int                    `json:"analyzed_works"`
	PlagiarizedWorks   int                    `json:"plagiarized_works"`
	AvgMatchPercentage float64                `json:"avg_match_percentage"`
	Reports            []GetReportResponse    `json:"reports,omitempty"`
	Statistics         map[string]interface{} `json:"statistics,omitempty"`
	LastAnalyzedAt     *time.Time             `json:"last_analyzed_at,omitempty"`
}

type SearchReportsRequest struct {
	WorkID         *string `json:"work_id,omitempty"`
	AssignmentID   *string `json:"assignment_id,omitempty"`
	StudentID      *string `json:"student_id,omitempty"`
	Status         *string `json:"status,omitempty"`
	PlagiarismFlag *bool   `json:"plagiarism_flag,omitempty"`
	DateFrom       *string `json:"date_from,omitempty"`
	DateTo         *string `json:"date_to,omitempty"`
	Page           int     `json:"page" validate:"min=1"`
	Limit          int     `json:"limit" validate:"min=1,max=100"`
}

type SearchReportsResponse struct {
	Reports    []GetReportResponse `json:"reports"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

type HealthCheckResponse struct {
	Status        string    `json:"status"`
	Database      bool      `json:"database"`
	RabbitMQ      bool      `json:"rabbitmq"`
	WorkService   bool      `json:"work_service"`
	FileService   bool      `json:"file_service"`
	ActiveWorkers int       `json:"active_workers"`
	QueueLength   int       `json:"queue_length"`
	Uptime        string    `json:"uptime"`
	Timestamp     time.Time `json:"timestamp"`
}

type FileHashRequest struct {
	FileID string `json:"file_id" validate:"required"`
}

type FileHashResponse struct {
	FileID string `json:"file_id"`
	Hash   string `json:"hash"`
	Size   int64  `json:"size"`
}
