package models

import (
	"encoding/json"
	"time"
)

type Report struct {
	ID                 string          `json:"id" db:"id"`
	WorkID             string          `json:"work_id" db:"work_id"`
	FileID             string          `json:"file_id" db:"file_id"`
	AssignmentID       string          `json:"assignment_id" db:"assignment_id"`
	StudentID          string          `json:"student_id" db:"student_id"`
	Status             string          `json:"status" db:"status"`
	PlagiarismFlag     bool            `json:"plagiarism_flag" db:"plagiarism_flag"`
	OriginalWorkID     *string         `json:"original_work_id,omitempty" db:"original_work_id"`
	MatchPercentage    int             `json:"match_percentage" db:"match_percentage"`
	FileHash           string          `json:"file_hash,omitempty" db:"file_hash"`
	ComparedHashes     []string        `json:"compared_hashes,omitempty" db:"compared_hashes"`
	Details            json.RawMessage `json:"details,omitempty" db:"details"`
	ProcessingTimeMs   *int            `json:"processing_time_ms,omitempty" db:"processing_time_ms"`
	ComparedFilesCount int             `json:"compared_files_count" db:"compared_files_count"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	StartedAt          *time.Time      `json:"started_at,omitempty" db:"started_at"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

type ReportStatus string

const (
	ReportStatusPending    ReportStatus = "pending"
	ReportStatusProcessing ReportStatus = "processing"
	ReportStatusCompleted  ReportStatus = "completed"
	ReportStatusFailed     ReportStatus = "failed"
)

func (rs ReportStatus) String() string {
	return string(rs)
}

type ReportDetails struct {
	ComparisonResults []ComparisonResult `json:"comparison_results,omitempty"`
	FileInfo          FileInfo           `json:"file_info,omitempty"`
	AnalysisMetadata  AnalysisMetadata   `json:"analysis_metadata,omitempty"`
}

type ComparisonResult struct {
	ComparedWorkID  string `json:"compared_work_id"`
	StudentID       string `json:"student_id"`
	MatchPercentage int    `json:"match_percentage"`
	FileHash        string `json:"file_hash"`
	FileName        string `json:"file_name"`
	ComparedAt      string `json:"compared_at"`
}

type FileInfo struct {
	FileName     string `json:"file_name"`
	FileSize     int64  `json:"file_size"`
	MimeType     string `json:"mime_type"`
	OriginalName string `json:"original_name"`
}

type AnalysisMetadata struct {
	AlgorithmUsed    string    `json:"algorithm_used"`
	SimilarityMethod string    `json:"similarity_method"`
	AnalysisVersion  string    `json:"analysis_version"`
	Threshold        int       `json:"threshold"`
	StartedAt        time.Time `json:"started_at"`
	CompletedAt      time.Time `json:"completed_at"`
}

type AssignmentStats struct {
	AssignmentID       string     `json:"assignment_id" db:"assignment_id"`
	TotalWorks         int        `json:"total_works" db:"total_works"`
	AnalyzedWorks      int        `json:"analyzed_works" db:"analyzed_works"`
	PlagiarizedWorks   int        `json:"plagiarized_works" db:"plagiarized_works"`
	AvgMatchPercentage float64    `json:"avg_match_percentage" db:"avg_match_percentage"`
	LastAnalyzedAt     *time.Time `json:"last_analyzed_at,omitempty" db:"last_analyzed_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

type StudentStats struct {
	StudentID          string     `json:"student_id" db:"student_id"`
	TotalWorks         int        `json:"total_works" db:"total_works"`
	AnalyzedWorks      int        `json:"analyzed_works" db:"analyzed_works"`
	PlagiarizedWorks   int        `json:"plagiarized_works" db:"plagiarized_works"`
	AvgMatchPercentage float64    `json:"avg_match_percentage" db:"avg_match_percentage"`
	LastAnalyzedAt     *time.Time `json:"last_analyzed_at,omitempty" db:"last_analyzed_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

type AnalysisQueueItem struct {
	ID           string     `json:"id" db:"id"`
	WorkID       string     `json:"work_id" db:"work_id"`
	FileID       string     `json:"file_id" db:"file_id"`
	AssignmentID string     `json:"assignment_id" db:"assignment_id"`
	StudentID    string     `json:"student_id" db:"student_id"`
	Status       string     `json:"status" db:"status"`
	Priority     int        `json:"priority" db:"priority"`
	Attempts     int        `json:"attempts" db:"attempts"`
	MaxAttempts  int        `json:"max_attempts" db:"max_attempts"`
	ErrorMessage string     `json:"error_message,omitempty" db:"error_message"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	ScheduledAt  *time.Time `json:"scheduled_at,omitempty" db:"scheduled_at"`
	StartedAt    *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}
