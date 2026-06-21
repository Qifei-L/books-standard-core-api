package bill

import (
	"errors"
	"net/http"

	"github.com/Qifei-L/books-standard-core-api/internal/auth"
	"github.com/Qifei-L/books-standard-core-api/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/bills", h.list)
	r.Post("/api/bills", h.create)
	r.Get("/api/bills/{id}", h.get)
	r.Post("/api/bills/{id}/approve", h.approve)
	r.Post("/api/bills/{id}/void", h.void)
	r.Post("/api/bills/{id}/payments", h.recordPayment)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	p := ListParams{
		Status:    r.URL.Query().Get("status"),
		ContactID: r.URL.Query().Get("contactId"),
		DateFrom:  r.URL.Query().Get("dateFrom"),
		DateTo:    r.URL.Query().Get("dateTo"),
	}
	items, err := h.svc.List(r.Context(), orgID, p)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list bills")
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	b, err := h.svc.Get(r.Context(), orgID, chi.URLParam(r, "id"))
	if writeBillErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	var req CreateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	b, err := h.svc.Create(r.Context(), orgID, req)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, b)
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	b, err := h.svc.Approve(r.Context(), orgID, chi.URLParam(r, "id"))
	if writeBillErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (h *Handler) void(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	b, err := h.svc.Void(r.Context(), orgID, chi.URLParam(r, "id"))
	if writeBillErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (h *Handler) recordPayment(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	var req RecordPaymentRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	b, err := h.svc.RecordPayment(r.Context(), orgID, chi.URLParam(r, "id"), req)
	if writeBillErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func writeBillErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	} else if errors.Is(err, ErrInvalidState) {
		httpx.Error(w, http.StatusConflict, "INVALID_STATE", err.Error())
	} else {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	return true
}
