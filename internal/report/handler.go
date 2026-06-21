package report

import (
	"net/http"
	"time"

	"github.com/Qifei-L/books-standard-core-api/internal/auth"
	"github.com/Qifei-L/books-standard-core-api/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/reports/trial-balance", h.trialBalance)
	r.Get("/api/reports/profit-and-loss", h.profitAndLoss)
	r.Get("/api/reports/balance-sheet", h.balanceSheet)
	r.Get("/api/reports/aged-receivables", h.agedReceivables)
	r.Get("/api/reports/aged-payables", h.agedPayables)
}

func (h *Handler) trialBalance(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	dateFrom, dateTo := dateRange(r)
	report, err := h.svc.TrialBalance(r.Context(), orgID, dateFrom, dateTo)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, report)
}

func (h *Handler) profitAndLoss(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	dateFrom, dateTo := dateRange(r)
	report, err := h.svc.ProfitAndLoss(r.Context(), orgID, dateFrom, dateTo)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, report)
}

func (h *Handler) balanceSheet(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	asAt := r.URL.Query().Get("asAt")
	if asAt == "" {
		asAt = time.Now().Format("2006-01-02")
	}
	report, err := h.svc.BalanceSheet(r.Context(), orgID, asAt)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, report)
}

func (h *Handler) agedReceivables(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	report, err := h.svc.AgedReceivables(r.Context(), orgID, r.URL.Query().Get("asAt"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, report)
}

func (h *Handler) agedPayables(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	report, err := h.svc.AgedPayables(r.Context(), orgID, r.URL.Query().Get("asAt"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, report)
}

func dateRange(r *http.Request) (string, string) {
	now := time.Now()
	dateFrom := r.URL.Query().Get("dateFrom")
	dateTo := r.URL.Query().Get("dateTo")
	if dateFrom == "" {
		dateFrom = now.Format("2006-01") + "-01"
	}
	if dateTo == "" {
		dateTo = now.Format("2006-01-02")
	}
	return dateFrom, dateTo
}
