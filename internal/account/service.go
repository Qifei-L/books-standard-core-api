package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("account not found")
	ErrConflict = errors.New("account code already exists")
)

var validTypes = map[string]bool{
	"asset": true, "liability": true, "equity": true,
	"income": true, "expense": true,
}

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) List(ctx context.Context, orgID, accountType string) ([]Account, error) {
	q := `SELECT id, code, name, type, is_active, created_at FROM accounts WHERE org_id = $1`
	args := []any{orgID}
	if accountType != "" {
		q += fmt.Sprintf(" AND type = $%d", len(args)+1)
		args = append(args, accountType)
	}
	q += " ORDER BY code"
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.Type, &a.IsActive, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, orgID, code string) (Account, error) {
	var a Account
	err := s.pool.QueryRow(ctx,
		`SELECT id, code, name, type, is_active, created_at FROM accounts WHERE org_id=$1 AND code=$2`,
		orgID, code,
	).Scan(&a.ID, &a.Code, &a.Name, &a.Type, &a.IsActive, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	return a, err
}

func (s *Service) Create(ctx context.Context, orgID string, req CreateRequest) (Account, error) {
	if req.Code == "" || req.Name == "" {
		return Account{}, fmt.Errorf("code and name required")
	}
	if !validTypes[req.Type] {
		return Account{}, fmt.Errorf("type must be asset, liability, equity, income, or expense")
	}
	var a Account
	err := s.pool.QueryRow(ctx,
		`INSERT INTO accounts (org_id, code, name, type)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, code, name, type, is_active, created_at`,
		orgID, req.Code, req.Name, req.Type,
	).Scan(&a.ID, &a.Code, &a.Name, &a.Type, &a.IsActive, &a.CreatedAt)
	if err != nil && isDuplicateKey(err) {
		return Account{}, ErrConflict
	}
	return a, err
}

func (s *Service) Update(ctx context.Context, orgID, code string, req UpdateRequest) (Account, error) {
	existing, err := s.Get(ctx, orgID, code)
	if err != nil {
		return Account{}, err
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	var a Account
	err = s.pool.QueryRow(ctx,
		`UPDATE accounts SET name=$1, is_active=$2 WHERE org_id=$3 AND code=$4
		 RETURNING id, code, name, type, is_active, created_at`,
		existing.Name, existing.IsActive, orgID, code,
	).Scan(&a.ID, &a.Code, &a.Name, &a.Type, &a.IsActive, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	return a, err
}

func isDuplicateKey(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
