package journal

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
	r.Get("/api/journal-entries", h.list)
	r.Post("/api/journal-entries", h.create)
	r.Get("/api/journal-entries/{id}", h.get)
	r.Post("/api/journal-entries/{id}/void", h.void)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	items, err := h.svc.List(r.Context(), orgID, ListParams{
		DateFrom: r.URL.Query().Get("dateFrom"),
		DateTo:   r.URL.Query().Get("dateTo"),
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list journal entries")
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	e, err := h.svc.Get(r.Context(), orgID, chi.URLParam(r, "id"))
	if writeJournalErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, e)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	var req CreateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	e, err := h.svc.Create(r.Context(), orgID, req)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, e)
}

func (h *Handler) void(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	e, err := h.svc.Void(r.Context(), orgID, chi.URLParam(r, "id"))
	if writeJournalErr(w, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, e)
}

func writeJournalErr(w http.ResponseWriter, err error) bool {
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
