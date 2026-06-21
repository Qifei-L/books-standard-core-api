package account

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
	r.Get("/api/accounts", h.list)
	r.Post("/api/accounts", h.create)
	r.Get("/api/accounts/{code}", h.get)
	r.Patch("/api/accounts/{code}", h.update)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	accounts, err := h.svc.List(r.Context(), orgID, r.URL.Query().Get("type"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list accounts")
		return
	}
	httpx.JSON(w, http.StatusOK, accounts)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	a, err := h.svc.Get(r.Context(), orgID, chi.URLParam(r, "code"))
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "account not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get account")
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	var req CreateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	a, err := h.svc.Create(r.Context(), orgID, req)
	if errors.Is(err, ErrConflict) {
		httpx.Error(w, http.StatusConflict, "CONFLICT", "account code already exists")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, a)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFrom(r.Context())
	var req UpdateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	a, err := h.svc.Update(r.Context(), orgID, chi.URLParam(r, "code"), req)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "NOT_FOUND", "account not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}
