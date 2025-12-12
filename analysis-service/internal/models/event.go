package models

import (
	"time"
)

type WorkCreatedEvent struct {
	WorkID       string `json:"work_id"`
	FileID       string `json:"file_id"`
	StudentID    string `json:"student_id"`
	AssignmentID string `json:"assignment_id"`
	Timestamp    int64  `json:"timestamp"`
}

type AnalysisStartedEvent struct {
	WorkID    string    `json:"work_id"`
	StartedAt time.Time `json:"started_at"`
}

type AnalysisCompletedEvent struct {
	WorkID          string    `json:"work_id"`
	ReportID        string    `json:"report_id"`
	Status          string    `json:"status"`
	PlagiarismFlag  bool      `json:"plagiarism_flag"`
	OriginalWorkID  *string   `json:"original_work_id,omitempty"`
	MatchPercentage int       `json:"match_percentage"`
	ProcessingTime  int       `json:"processing_time_ms"`
	CompletedAt     time.Time `json:"completed_at"`
}

type AnalysisFailedEvent struct {
	WorkID   string    `json:"work_id"`
	Error    string    `json:"error"`
	Attempts int       `json:"attempts"`
	FailedAt time.Time `json:"failed_at"`
}

type QueueStatsEvent struct {
	QueueLength    int       `json:"queue_length"`
	ActiveWorkers  int       `json:"active_workers"`
	ProcessedToday int       `json:"processed_today"`
	Timestamp      time.Time `json:"timestamp"`
}
