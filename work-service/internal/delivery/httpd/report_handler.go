package httpd

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetWorkReport(w http.ResponseWriter, r *http.Request) {
	workID := chi.URLParam(r, "id")
	if workID == "" {
		writeError(w, http.StatusBadRequest, "Work ID is required")
		return
	}

	ctx := r.Context()
	report, err := h.reportService.GetWorkReport(ctx, workID)
	if err != nil {
		h.handleReportError(w, err)
		return
	}

	writeSuccess(w, report)
}

func (h *Handler) handleReportError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case errMsg == "work not found":
		writeError(w, http.StatusNotFound, errMsg)
	default:
		h.logger.Error().Err(err).Msg("Report service error")
		writeError(w, http.StatusInternalServerError, "Internal server error")
	}
}
