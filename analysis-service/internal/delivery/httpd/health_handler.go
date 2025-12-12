package httpd

import (
	"net/http"
	"time"
)

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "analysis-service",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	writeJSON(w, http.StatusOK, response)
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
