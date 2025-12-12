package httpd

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) CreateWork(w http.ResponseWriter, r *http.Request) {
	// Проверяем Content-Type
	if r.Header.Get("Content-Type") == "multipart/form-data" {
		h.UploadWork(w, r)
		return
	}

	// Читаем JSON запрос
	var req models.CreateWorkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Валидация
	if req.StudentID == "" || req.AssignmentID == "" {
		writeError(w, http.StatusBadRequest, "student_id and assignment_id are required")
		return
	}

	// Проверяем UUID
	if _, err := uuid.Parse(req.StudentID); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid student_id format")
		return
	}

	if _, err := uuid.Parse(req.AssignmentID); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assignment_id format")
		return
	}

	// Создаем работу
	ctx := r.Context()
	response, err := h.workService.CreateWork(ctx, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeSuccess(w, response)
}

func (h *Handler) UploadWork(w http.ResponseWriter, r *http.Request) {
	// Парсим multipart форму
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
		writeError(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	// Получаем файл
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "File is required")
		return
	}
	defer file.Close()

	// Читаем содержимое файла
	fileContent, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Получаем остальные поля формы
	studentID := r.FormValue("student_id")
	assignmentID := r.FormValue("assignment_id")

	// Валидация
	if studentID == "" || assignmentID == "" {
		writeError(w, http.StatusBadRequest, "student_id and assignment_id are required")
		return
	}

	// Проверяем UUID
	if _, err := uuid.Parse(studentID); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid student_id format")
		return
	}

	if _, err := uuid.Parse(assignmentID); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid assignment_id format")
		return
	}

	// Создаем запрос на загрузку
	req := &models.UploadWorkRequest{
		StudentID:    studentID,
		AssignmentID: assignmentID,
		FileContent:  fileContent,
		FileName:     header.Filename,
	}

	// Загружаем работу
	ctx := r.Context()
	response, err := h.workService.UploadWork(ctx, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeSuccess(w, response)
}

func (h *Handler) GetWorkByID(w http.ResponseWriter, r *http.Request) {
	workID := chi.URLParam(r, "id")
	if workID == "" {
		writeError(w, http.StatusBadRequest, "Work ID is required")
		return
	}

	ctx := r.Context()
	work, err := h.workService.GetWorkByID(ctx, workID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	if work == nil {
		writeError(w, http.StatusNotFound, "Work not found")
		return
	}

	writeSuccess(w, work)
}

func (h *Handler) GetAllWorks(w http.ResponseWriter, r *http.Request) {
	page := getIntQueryParam(r, "page", 1)
	limit := getIntQueryParam(r, "limit", 20)

	ctx := r.Context()
	response, err := h.workService.GetAllWorks(ctx, page, limit)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeSuccess(w, response)
}

func (h *Handler) DeleteWork(w http.ResponseWriter, r *http.Request) {
	workID := chi.URLParam(r, "id")
	if workID == "" {
		writeError(w, http.StatusBadRequest, "Work ID is required")
		return
	}

	ctx := r.Context()
	if err := h.workService.DeleteWork(ctx, workID); err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeSuccess(w, map[string]interface{}{
		"message": "Work deleted successfully",
	})
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case errMsg == "student not found" || errMsg == "assignment not found":
		writeError(w, http.StatusNotFound, errMsg)
	case errMsg == "work already submitted for this assignment":
		writeError(w, http.StatusConflict, errMsg)
	case errMsg == "work not found":
		writeError(w, http.StatusNotFound, errMsg)
	case errMsg == "invalid work status":
		writeError(w, http.StatusBadRequest, errMsg)
	default:
		h.logger.Error().Err(err).Msg("Service error")
		writeError(w, http.StatusInternalServerError, "Internal server error")
	}
}
