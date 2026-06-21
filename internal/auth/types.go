package auth

import "github.com/golang-jwt/jwt/v5"

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string   `json:"accessToken"`
	User        UserInfo `json:"user"`
}

type UserInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	OrgID       string   `json:"orgId"`
	Permissions []string `json:"permissions"`
}

type Claims struct {
	UserID      string   `json:"userId"`
	OrgID       string   `json:"orgId"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

type contextKey string

const claimsKey contextKey = "claims"
