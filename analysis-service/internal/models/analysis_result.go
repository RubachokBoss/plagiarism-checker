package models

import (
	"time"
)

type AnalysisResult struct {
	WorkID            string        `json:"work_id"`
	Status            string        `json:"status"`
	PlagiarismFlag    bool          `json:"plagiarism_flag"`
	OriginalWorkID    *string       `json:"original_work_id,omitempty"`
	MatchPercentage   int           `json:"match_percentage"`
	ComparedWithCount int           `json:"compared_with_count"`
	SimilarWorks      []SimilarWork `json:"similar_works,omitempty"`
	FileHash          string        `json:"file_hash"`
	ProcessingTimeMs  int           `json:"processing_time_ms"`
	AnalyzedAt        time.Time     `json:"analyzed_at"`
	Details           []byte        `json:"details,omitempty"`
}

type SimilarWork struct {
	WorkID          string    `json:"work_id"`
	StudentID       string    `json:"student_id"`
	StudentName     string    `json:"student_name,omitempty"`
	MatchPercentage int       `json:"match_percentage"`
	FileHash        string    `json:"file_hash"`
	SubmittedAt     time.Time `json:"submitted_at"`
}

type PlagiarismCheckRequest struct {
	WorkID       string `json:"work_id"`
	FileID       string `json:"file_id"`
	AssignmentID string `json:"assignment_id"`
	StudentID    string `json:"student_id"`
}

type PlagiarismCheckResponse struct {
	ReportID        string    `json:"report_id"`
	WorkID          string    `json:"work_id"`
	Status          string    `json:"status"`
	PlagiarismFlag  bool      `json:"plagiarism_flag"`
	MatchPercentage int       `json:"match_percentage"`
	OriginalWorkID  *string   `json:"original_work_id,omitempty"`
	AnalyzedAt      time.Time `json:"analyzed_at"`
}

type BatchAnalysisRequest struct {
	WorkIDs []string `json:"work_ids"`
}

type BatchAnalysisResponse struct {
	Total       int                       `json:"total"`
	Processed   int                       `json:"processed"`
	Failed      int                       `json:"failed"`
	Results     []PlagiarismCheckResponse `json:"results"`
	CompletedAt time.Time                 `json:"completed_at"`
}

type AnalysisStats struct {
	TotalReports      int64             `json:"total_reports"`
	CompletedReports  int64             `json:"completed_reports"`
	PendingReports    int64             `json:"pending_reports"`
	PlagiarizedWorks  int64             `json:"plagiarized_works"`
	AvgProcessingTime float64           `json:"avg_processing_time"`
	TopAssignments    []AssignmentStats `json:"top_assignments"`
	TopStudents       []StudentStats    `json:"top_students"`
	RecentActivity    []Report          `json:"recent_activity"`
}
