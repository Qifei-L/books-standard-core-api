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
	row := s.pool.QueryRow(ctx,
		`SELECT id, org_id, email, password_hash, name, role FROM users WHERE email = $1 AND is_active = true`,
		email,
	)
	var u struct {
		ID           string
		OrgID        string
		Email        string
		PasswordHash string
		Name         string
		Role         string
	}
	if err := row.Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Name, &u.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResponse{}, "", ErrInvalidCredentials
		}
		return LoginResponse{}, "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return LoginResponse{}, "", ErrInvalidCredentials
	}

	accessToken, err := s.signAccess(u.ID, u.OrgID, u.Role)
	if err != nil {
		return LoginResponse{}, "", err
	}
	refreshToken, err := s.createRefresh(ctx, u.ID)
	if err != nil {
		return LoginResponse{}, "", err
	}

	return LoginResponse{
		AccessToken: accessToken,
		User:        UserInfo{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, OrgID: u.OrgID},
	}, refreshToken, nil
}

func (s *Service) Refresh(ctx context.Context, rawToken string) (string, error) {
	h := hashToken(rawToken)
	row := s.pool.QueryRow(ctx,
		`SELECT u.id, u.org_id, u.role FROM refresh_tokens rt
		 JOIN users u ON u.id = rt.user_id
		 WHERE rt.token_hash = $1 AND rt.expires_at > now() AND u.is_active = true`,
		h,
	)
	var userID, orgID, role string
	if err := row.Scan(&userID, &orgID, &role); err != nil {
		return "", ErrInvalidToken
	}
	return s.signAccess(userID, orgID, role)
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

func (s *Service) signAccess(userID, orgID, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func (s *Service) createRefresh(ctx context.Context, userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(b)
	h := hashToken(raw)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, h, time.Now().Add(30*24*time.Hour),
	)
	return raw, err
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
