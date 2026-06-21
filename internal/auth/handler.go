package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/Qifei-L/books-standard-core-api/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
)

const refreshCookie = "refresh_token"

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/auth/login", h.login)
	r.Post("/api/auth/refresh", h.refresh)
	r.Post("/api/auth/logout", h.logout)
}

func (h *Handler) RegisterProtected(r chi.Router) {
	r.Get("/api/auth/me", h.me)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	resp, refreshToken, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "login failed")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookie,
		Value:    refreshToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/auth",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookie)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing refresh token")
		return
	}
	token, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"accessToken": token})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookie)
	if err == nil {
		_ = h.svc.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:    refreshCookie,
		Value:   "",
		MaxAge:  -1,
		Path:    "/api/auth",
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFrom(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{
		"userId": claims.UserID,
		"orgId":  claims.OrgID,
		"role":   claims.Role,
	})
}
