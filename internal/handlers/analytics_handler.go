package handlers

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type AnalyticsHandler struct {
	analyticsSvc service.AnalyticsServiceInterface
}

func NewAnalyticsHandler(analyticsSvc service.AnalyticsServiceInterface) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsSvc: analyticsSvc}
}

// RecordEvent handles POST /api/v1/articles/{id}/analytics
func (h *AnalyticsHandler) RecordEvent(w http.ResponseWriter, r *http.Request) {
	articleID := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(articleID) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid article ID")
		return
	}

	var input models.AnalyticsEventInput
	if err := utils.DecodeJSON(w, r, &input); err != nil {
		utils.HandleDecodeError(w, err)
		return
	}

	// Idempotency fallback from header if missing in JSON body
	if input.EventID == "" {
		input.EventID = r.Header.Get("X-Event-ID")
	}
	if input.SessionID == "" {
		input.SessionID = r.Header.Get("X-Session-ID")
	}

	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = r.RemoteAddr
		}
	} else if strings.Contains(ip, ",") {
		// Take the first client IP in proxy chain
		parts := strings.Split(ip, ",")
		ip = strings.TrimSpace(parts[0])
	}

	userAgent := r.UserAgent()

	if err := h.analyticsSvc.RecordEvent(r.Context(), articleID, input, ip, userAgent); err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "article not found")
			return
		}
		utils.ValidationError(w, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// GetArticleAnalytics handles GET /api/v1/admin/articles/{id}/analytics
func (h *AnalyticsHandler) GetArticleAnalytics(w http.ResponseWriter, r *http.Request) {
	articleID := strings.TrimSpace(chi.URLParam(r, "id"))
	if !utils.IsValidULID(articleID) {
		utils.ErrorCode(w, http.StatusBadRequest, "INVALID_ID", "invalid article ID")
		return
	}

	now := time.Now().UTC()
	startDate := now.AddDate(0, 0, -30)
	endDate := now

	if s := r.URL.Query().Get("start_date"); s != "" {
		if parsed, err := time.Parse("2006-01-02", s); err == nil {
			startDate = parsed
		}
	}
	if e := r.URL.Query().Get("end_date"); e != "" {
		if parsed, err := time.Parse("2006-01-02", e); err == nil {
			endDate = parsed
		}
	}

	report, err := h.analyticsSvc.GetArticleAnalytics(r.Context(), articleID, startDate, endDate)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "NOT_FOUND", "article not found")
			return
		}
		utils.ErrorCode(w, http.StatusInternalServerError, "FETCH_ERROR", "failed to fetch article analytics")
		return
	}

	utils.JSON(w, http.StatusOK, report)
}

// GetOverview handles GET /api/v1/admin/analytics/overview
func (h *AnalyticsHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.analyticsSvc.GetOverview(r.Context())
	if err != nil {
		utils.ErrorCode(w, http.StatusInternalServerError, "FETCH_ERROR", "failed to fetch analytics overview")
		return
	}

	utils.JSON(w, http.StatusOK, overview)
}
