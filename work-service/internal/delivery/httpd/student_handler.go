package httpd

import (
	"encoding/json"
	"net/http"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	var req models.CreateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	ctx := r.Context()
	student, err := h.studentService.CreateStudent(ctx, &req)
	if err != nil {
		h.handleStudentError(w, err)
		return
	}

	writeSuccess(w, student)
}

func (h *Handler) GetStudentByID(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "id")
	if studentID == "" {
		writeError(w, http.StatusBadRequest, "Student ID is required")
		return
	}

	ctx := r.Context()
	student, err := h.studentService.GetStudentByID(ctx, studentID)
	if err != nil {
		h.handleStudentError(w, err)
		return
	}

	if student == nil {
		writeError(w, http.StatusNotFound, "Student not found")
		return
	}

	writeSuccess(w, student)
}

func (h *Handler) GetStudentByEmail(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")
	if email == "" {
		writeError(w, http.StatusBadRequest, "Email is required")
		return
	}

	ctx := r.Context()
	student, err := h.studentService.GetStudentByEmail(ctx, email)
	if err != nil {
		h.handleStudentError(w, err)
		return
	}

	if student == nil {
		writeError(w, http.StatusNotFound, "Student not found")
		return
	}

	writeSuccess(w, student)
}

func (h *Handler) GetAllStudents(w http.ResponseWriter, r *http.Request) {
	page := getIntQueryParam(r, "page", 1)
	limit := getIntQueryParam(r, "limit", 20)

	ctx := r.Context()
	students, total, err := h.studentService.GetAllStudents(ctx, page, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get students")
		writeError(w, http.StatusInternalServerError, "Failed to get students")
		return
	}

	response := map[string]interface{}{
		"students": students,
		"total":    total,
		"page":     page,
		"limit":    limit,
	}

	writeSuccess(w, response)
}

func (h *Handler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "id")
	if studentID == "" {
		writeError(w, http.StatusBadRequest, "Student ID is required")
		return
	}

	var req models.CreateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	ctx := r.Context()
	if err := h.studentService.UpdateStudent(ctx, studentID, &req); err != nil {
		h.handleStudentError(w, err)
		return
	}

	writeSuccess(w, map[string]interface{}{
		"message": "Student updated successfully",
	})
}

func (h *Handler) DeleteStudent(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "id")
	if studentID == "" {
		writeError(w, http.StatusBadRequest, "Student ID is required")
		return
	}

	ctx := r.Context()
	if err := h.studentService.DeleteStudent(ctx, studentID); err != nil {
		h.handleStudentError(w, err)
		return
	}

	writeSuccess(w, map[string]interface{}{
		"message": "Student deleted successfully",
	})
}

func (h *Handler) GetWorksByStudent(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "id")
	if studentID == "" {
		writeError(w, http.StatusBadRequest, "Student ID is required")
		return
	}

	page := getIntQueryParam(r, "page", 1)
	limit := getIntQueryParam(r, "limit", 20)

	ctx := r.Context()
	response, err := h.workService.GetWorksByStudent(ctx, studentID, page, limit)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeSuccess(w, response)
}

func (h *Handler) handleStudentError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case errMsg == "student not found":
		writeError(w, http.StatusNotFound, errMsg)
	case errMsg == "student with this email already exists":
		writeError(w, http.StatusConflict, errMsg)
	case errMsg == "email already in use by another student":
		writeError(w, http.StatusConflict, errMsg)
	case errMsg == "cannot delete student with existing works":
		writeError(w, http.StatusConflict, errMsg)
	default:
		h.logger.Error().Err(err).Msg("Student service error")
		writeError(w, http.StatusInternalServerError, "Internal server error")
	}
}
