package invoice

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
	r.Get("/api/invoices", h.list)
	r.Post("/api/invoices", h.create)
	r.Get("/api/invoices/{id}", h.get)
	r.Post("/api/invoices/{id}/approve", h.approve)
	r.Post("/api/invoices/{id}/void", h.void)
	r.Post("/api/invoices/{id}/payments", h.recordPayment)
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
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list invoices")
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	inv, err := h.svc.Get(r.Context(), orgID, chi.URLParam(r, "id"))
	if writeInvoiceErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	var req CreateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	if req.Number == "" {
		req.Number = defaultNumber()
	}
	inv, err := h.svc.Create(r.Context(), orgID, req)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, inv)
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	inv, err := h.svc.Approve(r.Context(), orgID, chi.URLParam(r, "id"))
	if writeInvoiceErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func (h *Handler) void(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	inv, err := h.svc.Void(r.Context(), orgID, chi.URLParam(r, "id"))
	if writeInvoiceErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func (h *Handler) recordPayment(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	var req RecordPaymentRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	inv, err := h.svc.RecordPayment(r.Context(), orgID, chi.URLParam(r, "id"), req)
	if writeInvoiceErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func writeInvoiceErr(w http.ResponseWriter, err error) bool {
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
