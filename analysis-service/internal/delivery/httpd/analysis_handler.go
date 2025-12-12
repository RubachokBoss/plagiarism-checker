package httpd

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) AnalyzeWork(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkID       string `json:"work_id"`
		FileID       string `json:"file_id"`
		AssignmentID string `json:"assignment_id"`
		StudentID    string `json:"student_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.WorkID == "" || req.FileID == "" || req.AssignmentID == "" || req.StudentID == "" {
		writeError(w, http.StatusBadRequest, "All fields (work_id, file_id, assignment_id, student_id) are required")
		return
	}

	ctx := r.Context()
	result, err := h.analysisService.AnalyzeWork(ctx, req.WorkID, req.FileID, req.AssignmentID, req.StudentID)
	if err != nil {
		h.handleAnalysisError(w, err)
		return
	}

	writeSuccess(w, result)
}

func (h *Handler) AnalyzeWorkAsync(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkID       string `json:"work_id"`
		FileID       string `json:"file_id"`
		AssignmentID string `json:"assignment_id"`
		StudentID    string `json:"student_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.WorkID == "" || req.FileID == "" || req.AssignmentID == "" || req.StudentID == "" {
		writeError(w, http.StatusBadRequest, "All fields (work_id, file_id, assignment_id, student_id) are required")
		return
	}

	ctx := r.Context()
	reportID, err := h.analysisService.AnalyzeWorkAsync(ctx, req.WorkID, req.FileID, req.AssignmentID, req.StudentID)
	if err != nil {
		h.handleAnalysisError(w, err)
		return
	}

	response := map[string]interface{}{
		"report_id":  reportID,
		"message":    "Analysis started asynchronously",
		"status_url": "/api/v1/analysis/" + req.WorkID,
	}

	writeSuccess(w, response)
}

func (h *Handler) GetAnalysisResult(w http.ResponseWriter, r *http.Request) {
	workID := chi.URLParam(r, "work_id")
	if workID == "" {
		writeError(w, http.StatusBadRequest, "Work ID is required")
		return
	}

	ctx := r.Context()
	result, err := h.analysisService.GetAnalysisResult(ctx, workID)
	if err != nil {
		h.handleAnalysisError(w, err)
		return
	}

	writeSuccess(w, result)
}

func (h *Handler) BatchAnalyze(w http.ResponseWriter, r *http.Request) {
	var req models.BatchAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.WorkIDs) == 0 {
		writeError(w, http.StatusBadRequest, "At least one work ID is required")
		return
	}

	ctx := r.Context()
	response, err := h.analysisService.BatchAnalyze(ctx, req.WorkIDs)
	if err != nil {
		h.handleAnalysisError(w, err)
		return
	}

	writeSuccess(w, response)
}

func (h *Handler) RetryFailedAnalyses(w http.ResponseWriter, r *http.Request) {
	limit := getIntQueryParam(r, "limit", 10)

	ctx := r.Context()
	retryCount, err := h.analysisService.RetryFailedAnalyses(ctx, limit)
	if err != nil {
		h.handleAnalysisError(w, err)
		return
	}

	response := map[string]interface{}{
		"retried":   retryCount,
		"limit":     limit,
		"message":   "Failed analyses retry completed",
		"timestamp": time.Now().UTC(),
	}

	writeSuccess(w, response)
}

func (h *Handler) handleAnalysisError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case errMsg == "analysis not found for this work":
		writeError(w, http.StatusNotFound, errMsg)
	case errMsg == "report not found for this work":
		writeError(w, http.StatusNotFound, errMsg)
	case errMsg == "batch size exceeds limit":
		writeError(w, http.StatusBadRequest, errMsg)
	case contains(errMsg, "failed to get file hash"):
		h.logger.Error().Err(err).Msg("File service error")
		writeError(w, http.StatusBadGateway, "File service unavailable")
	case contains(errMsg, "failed to get previous works"):
		h.logger.Error().Err(err).Msg("Work service error")
		writeError(w, http.StatusBadGateway, "Work service unavailable")
	case contains(errMsg, "plagiarism check failed"):
		h.logger.Error().Err(err).Msg("Analysis processing error")
		writeError(w, http.StatusInternalServerError, "Analysis failed")
	default:
		h.logger.Error().Err(err).Msg("Analysis error")
		writeError(w, http.StatusInternalServerError, "Internal server error")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
