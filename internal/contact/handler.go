package contact

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
	r.Get("/api/contacts", h.list)
	r.Post("/api/contacts", h.create)
	r.Get("/api/contacts/{id}", h.get)
	r.Patch("/api/contacts/{id}", h.update)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	contacts, err := h.svc.List(r.Context(), orgID, r.URL.Query().Get("type"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list contacts")
		return
	}
	httpx.JSON(w, http.StatusOK, contacts)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	c, err := h.svc.Get(r.Context(), orgID, chi.URLParam(r, "id"))
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "contact not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get contact")
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	var req CreateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	c, err := h.svc.Create(r.Context(), orgID, req)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	var req UpdateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	c, err := h.svc.Update(r.Context(), orgID, chi.URLParam(r, "id"), req)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "contact not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}
