package models

type WorkCreatedEvent struct {
	WorkID       string `json:"work_id"`
	FileID       string `json:"file_id"`
	StudentID    string `json:"student_id"`
	AssignmentID string `json:"assignment_id"`
	Timestamp    int64  `json:"timestamp"`
}

type AnalysisCompletedEvent struct {
	WorkID          string  `json:"work_id"`
	Status          string  `json:"status"`
	PlagiarismFlag  bool    `json:"plagiarism_flag"`
	OriginalWorkID  *string `json:"original_work_id,omitempty"`
	MatchPercentage int     `json:"match_percentage"`
}
