package httpd

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type Handler struct {
	analysisService service.AnalysisService
	reportService   service.ReportService
	logger          zerolog.Logger
}

func NewHandler(
	analysisService service.AnalysisService,
	reportService service.ReportService,
	logger zerolog.Logger,
) *Handler {
	return &Handler{
		analysisService: analysisService,
		reportService:   reportService,
		logger:          logger,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	// Health check
	router.Get("/health", h.HealthCheck)
	router.Get("/status", h.GetServiceStatus)
	router.Get("/stats", h.GetAllStats)

	// Analysis endpoints
	router.Route("/analysis", func(r chi.Router) {
		r.Post("/", h.AnalyzeWork)
		r.Post("/batch", h.BatchAnalyze)
		r.Post("/async", h.AnalyzeWorkAsync)
		r.Get("/{work_id}", h.GetAnalysisResult)
		r.Post("/retry", h.RetryFailedAnalyses)
	})

	// Report endpoints
	router.Route("/reports", func(r chi.Router) {
		r.Get("/", h.SearchReports)
		r.Get("/{report_id}", h.GetReport)
		r.Get("/work/{work_id}", h.GetReportByWorkID)
		r.Get("/assignment/{assignment_id}", h.GetAssignmentStats)
		r.Get("/student/{student_id}", h.GetStudentStats)
		r.Get("/export", h.ExportReports)
	})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "analysis-service",
		"timestamp": time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, response)
}

// Вспомогательные функции
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

func getInt64QueryParam(r *http.Request, key string, defaultValue int64) int64 {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func getBoolQueryParam(r *http.Request, key string) *bool {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return nil
	}

	return &boolValue
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
