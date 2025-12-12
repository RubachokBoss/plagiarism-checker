package httpd

import (
	"encoding/json"
	"net/http"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	ctx := r.Context()
	assignment, err := h.assignmentService.CreateAssignment(ctx, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create assignment")
		writeError(w, http.StatusInternalServerError, "Failed to create assignment")
		return
	}

	writeSuccess(w, assignment)
}

func (h *Handler) GetAssignmentByID(w http.ResponseWriter, r *http.Request) {
	assignmentID := chi.URLParam(r, "id")
	if assignmentID == "" {
		writeError(w, http.StatusBadRequest, "Assignment ID is required")
		return
	}

	ctx := r.Context()
	assignment, err := h.assignmentService.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		h.handleAssignmentError(w, err)
		return
	}

	if assignment == nil {
		writeError(w, http.StatusNotFound, "Assignment not found")
		return
	}

	writeSuccess(w, assignment)
}

func (h *Handler) GetAllAssignments(w http.ResponseWriter, r *http.Request) {
	page := getIntQueryParam(r, "page", 1)
	limit := getIntQueryParam(r, "limit", 20)

	ctx := r.Context()
	assignments, total, err := h.assignmentService.GetAllAssignments(ctx, page, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get assignments")
		writeError(w, http.StatusInternalServerError, "Failed to get assignments")
		return
	}

	response := map[string]interface{}{
		"assignments": assignments,
		"total":       total,
		"page":        page,
		"limit":       limit,
	}

	writeSuccess(w, response)
}

func (h *Handler) UpdateAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID := chi.URLParam(r, "id")
	if assignmentID == "" {
		writeError(w, http.StatusBadRequest, "Assignment ID is required")
		return
	}

	var req models.CreateAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	ctx := r.Context()
	if err := h.assignmentService.UpdateAssignment(ctx, assignmentID, &req); err != nil {
		h.handleAssignmentError(w, err)
		return
	}

	writeSuccess(w, map[string]interface{}{
		"message": "Assignment updated successfully",
	})
}

func (h *Handler) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID := chi.URLParam(r, "id")
	if assignmentID == "" {
		writeError(w, http.StatusBadRequest, "Assignment ID is required")
		return
	}

	ctx := r.Context()
	if err := h.assignmentService.DeleteAssignment(ctx, assignmentID); err != nil {
		h.handleAssignmentError(w, err)
		return
	}

	writeSuccess(w, map[string]interface{}{
		"message": "Assignment deleted successfully",
	})
}

func (h *Handler) GetWorksByAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID := chi.URLParam(r, "id")
	if assignmentID == "" {
		writeError(w, http.StatusBadRequest, "Assignment ID is required")
		return
	}

	page := getIntQueryParam(r, "page", 1)
	limit := getIntQueryParam(r, "limit", 20)

	ctx := r.Context()
	response, err := h.workService.GetWorksByAssignment(ctx, assignmentID, page, limit)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeSuccess(w, response)
}

func (h *Handler) handleAssignmentError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case errMsg == "assignment not found":
		writeError(w, http.StatusNotFound, errMsg)
	case errMsg == "cannot delete assignment with existing works":
		writeError(w, http.StatusConflict, errMsg)
	default:
		h.logger.Error().Err(err).Msg("Assignment service error")
		writeError(w, http.StatusInternalServerError, "Internal server error")
	}
}
