package httpd

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type Handler struct {
	workService       service.WorkService
	assignmentService service.AssignmentService
	studentService    service.StudentService
	reportService     service.ReportService
	logger            zerolog.Logger
}

func NewHandler(
	workService service.WorkService,
	assignmentService service.AssignmentService,
	studentService service.StudentService,
	reportService service.ReportService,
	logger zerolog.Logger,
) *Handler {
	return &Handler{
		workService:       workService,
		assignmentService: assignmentService,
		studentService:    studentService,
		reportService:     reportService,
		logger:            logger,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/health", h.HealthCheck)

	router.Route("/api/v1", func(api chi.Router) {
		api.Route("/works", func(r chi.Router) {
			r.Post("/", h.CreateWork)
			r.Get("/", h.GetAllWorks)
			r.Get("/{id}", h.GetWorkByID)
			r.Delete("/{id}", h.DeleteWork)
			r.Get("/{id}/reports", h.GetWorkReport)
			r.Put("/{id}/status", h.UpdateWorkStatus)
		})

		api.Route("/assignments", func(r chi.Router) {
			r.Post("/", h.CreateAssignment)
			r.Get("/", h.GetAllAssignments)
			r.Get("/{id}", h.GetAssignmentByID)
			r.Put("/{id}", h.UpdateAssignment)
			r.Delete("/{id}", h.DeleteAssignment)
			r.Get("/{id}/works", h.GetWorksByAssignment)
		})

		api.Route("/students", func(r chi.Router) {
			r.Post("/", h.CreateStudent)
			r.Get("/", h.GetAllStudents)
			r.Get("/{id}", h.GetStudentByID)
			r.Get("/email/{email}", h.GetStudentByEmail)
			r.Put("/{id}", h.UpdateStudent)
			r.Delete("/{id}", h.DeleteStudent)
			r.Get("/{id}/works", h.GetWorksByStudent)
		})
	})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "work-service",
		"timestamp": time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, response)
}

func getIntQueryParam(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error":   http.StatusText(status),
		"message": message,
	})
}

func writeSuccess(w http.ResponseWriter, data interface{}) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}
	writeJSON(w, http.StatusOK, response)
}
