package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type Service struct {
	pool      *pgxpool.Pool
	jwtSecret []byte
}

func NewService(pool *pgxpool.Pool) *Service {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "change-me-in-production"
	}
	return &Service{pool: pool, jwtSecret: []byte(secret)}
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResponse, string, error) {
	// Load the user and their single active membership (v1: one org per user).
	// Returns the first active membership if the user somehow has multiple.
	row := s.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.password_hash, u.name,
		       om.org_id, r.name, r.permissions
		FROM users u
		JOIN org_members om ON om.user_id = u.id AND om.is_active = true
		JOIN roles r        ON r.id = om.role_id
		JOIN organizations o ON o.id = om.org_id AND o.is_active = true
		WHERE u.email = $1 AND u.is_active = true
		LIMIT 1`,
		email,
	)

	var u struct {
		ID           string
		Email        string
		PasswordHash string
		Name         string
		OrgID        string
		Role         string
		Permissions  []string
	}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name,
		&u.OrgID, &u.Role, &u.Permissions); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResponse{}, "", ErrInvalidCredentials
		}
		return LoginResponse{}, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return LoginResponse{}, "", ErrInvalidCredentials
	}

	accessToken, err := s.signAccess(u.ID, u.OrgID, u.Role, u.Permissions)
	if err != nil {
		return LoginResponse{}, "", err
	}
	refreshToken, err := s.createRefresh(ctx, u.ID, u.OrgID)
	if err != nil {
		return LoginResponse{}, "", err
	}

	return LoginResponse{
		AccessToken: accessToken,
		User: UserInfo{
			ID:          u.ID,
			Name:        u.Name,
			Email:       u.Email,
			Role:        u.Role,
			OrgID:       u.OrgID,
			Permissions: u.Permissions,
		},
	}, refreshToken, nil
}

func (s *Service) Refresh(ctx context.Context, rawToken string) (string, error) {
	h := hashToken(rawToken)
	row := s.pool.QueryRow(ctx, `
		SELECT u.id, rt.org_id, r.name, r.permissions
		FROM refresh_tokens rt
		JOIN users u        ON u.id = rt.user_id AND u.is_active = true
		JOIN org_members om ON om.user_id = u.id AND om.org_id = rt.org_id AND om.is_active = true
		JOIN roles r        ON r.id = om.role_id
		WHERE rt.token_hash = $1 AND rt.expires_at > now()`,
		h,
	)
	var userID, orgID, role string
	var permissions []string
	if err := row.Scan(&userID, &orgID, &role, &permissions); err != nil {
		return "", ErrInvalidToken
	}
	return s.signAccess(userID, orgID, role, permissions)
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	h := hashToken(rawToken)
	_, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, h)
	return err
}

func (s *Service) ValidateAccess(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	c, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return c, nil
}

func (s *Service) signAccess(userID, orgID, role string, permissions []string) (string, error) {
	claims := Claims{
		UserID:      userID,
		OrgID:       orgID,
		Role:        role,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *Service) createRefresh(ctx context.Context, userID, orgID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(b)
	h := hashToken(raw)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, org_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		userID, orgID, h, time.Now().Add(30*24*time.Hour),
	)
	return raw, err
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
