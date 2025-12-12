package httpd

import (
	_ "encoding/json"
	"net/http"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "report_id")
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "Report ID is required")
		return
	}

	ctx := r.Context()
	report, err := h.reportService.GetReport(ctx, reportID)
	if err != nil {
		h.handleReportError(w, err)
		return
	}

	writeSuccess(w, report)
}

func (h *Handler) GetReportByWorkID(w http.ResponseWriter, r *http.Request) {
	workID := chi.URLParam(r, "work_id")
	if workID == "" {
		writeError(w, http.StatusBadRequest, "Work ID is required")
		return
	}

	ctx := r.Context()
	report, err := h.reportService.GetReportByWorkID(ctx, workID)
	if err != nil {
		h.handleReportError(w, err)
		return
	}

	writeSuccess(w, report)
}

func (h *Handler) SearchReports(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	workID := r.URL.Query().Get("work_id")
	assignmentID := r.URL.Query().Get("assignment_id")
	studentID := r.URL.Query().Get("student_id")
	status := r.URL.Query().Get("status")
	plagiarismFlag := getBoolQueryParam(r, "plagiarism_flag")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	page := getIntQueryParam(r, "page", 1)
	limit := getIntQueryParam(r, "limit", 20)

	// Build request
	req := models.SearchReportsRequest{
		WorkID:         stringOrNil(workID),
		AssignmentID:   stringOrNil(assignmentID),
		StudentID:      stringOrNil(studentID),
		Status:         stringOrNil(status),
		PlagiarismFlag: plagiarismFlag,
		DateFrom:       stringOrNil(dateFrom),
		DateTo:         stringOrNil(dateTo),
		Page:           page,
		Limit:          limit,
	}

	ctx := r.Context()
	response, err := h.reportService.SearchReports(ctx, req)
	if err != nil {
		h.handleReportError(w, err)
		return
	}

	writeSuccess(w, response)
}

func (h *Handler) GetAssignmentStats(w http.ResponseWriter, r *http.Request) {
	assignmentID := chi.URLParam(r, "assignment_id")
	if assignmentID == "" {
		writeError(w, http.StatusBadRequest, "Assignment ID is required")
		return
	}

	ctx := r.Context()
	stats, err := h.reportService.GetAssignmentStats(ctx, assignmentID)
	if err != nil {
		h.handleReportError(w, err)
		return
	}

	writeSuccess(w, stats)
}

func (h *Handler) GetStudentStats(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "student_id")
	if studentID == "" {
		writeError(w, http.StatusBadRequest, "Student ID is required")
		return
	}

	ctx := r.Context()
	stats, err := h.reportService.GetStudentStats(ctx, studentID)
	if err != nil {
		h.handleReportError(w, err)
		return
	}

	writeSuccess(w, stats)
}

func (h *Handler) GetAllStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, err := h.reportService.GetAllStats(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get all stats")
		writeError(w, http.StatusInternalServerError, "Failed to get statistics")
		return
	}

	writeSuccess(w, stats)
}

func (h *Handler) ExportReports(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	if format != "json" && format != "csv" {
		writeError(w, http.StatusBadRequest, "Unsupported format. Use 'json' or 'csv'")
		return
	}

	// Build filters
	filters := make(map[string]interface{})

	if workID := r.URL.Query().Get("work_id"); workID != "" {
		filters["work_id"] = workID
	}

	if assignmentID := r.URL.Query().Get("assignment_id"); assignmentID != "" {
		filters["assignment_id"] = assignmentID
	}

	if studentID := r.URL.Query().Get("student_id"); studentID != "" {
		filters["student_id"] = studentID
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}

	if plagiarismFlag := getBoolQueryParam(r, "plagiarism_flag"); plagiarismFlag != nil {
		filters["plagiarism_flag"] = *plagiarismFlag
	}

	ctx := r.Context()
	data, err := h.reportService.ExportReports(ctx, filters, format)
	if err != nil {
		h.handleReportError(w, err)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", getContentType(format))
	w.Header().Set("Content-Disposition", "attachment; filename=\"reports."+format+"\"")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) GetServiceStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status, err := h.analysisService.GetServiceStatus(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get service status")
		writeError(w, http.StatusInternalServerError, "Failed to get service status")
		return
	}

	writeSuccess(w, status)
}

func (h *Handler) handleReportError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case errMsg == "report not found":
		writeError(w, http.StatusNotFound, errMsg)
	case errMsg == "assignment not found or no reports available":
		writeError(w, http.StatusNotFound, errMsg)
	case errMsg == "student not found or no reports available":
		writeError(w, http.StatusNotFound, errMsg)
	case contains(errMsg, "failed to search reports"):
		h.logger.Error().Err(err).Msg("Database error")
		writeError(w, http.StatusInternalServerError, "Failed to search reports")
	default:
		h.logger.Error().Err(err).Msg("Report service error")
		writeError(w, http.StatusInternalServerError, "Internal server error")
	}
}

func stringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func getContentType(format string) string {
	switch format {
	case "json":
		return "application/json"
	case "csv":
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}
