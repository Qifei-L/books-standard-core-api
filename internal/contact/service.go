package contact

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("contact not found")

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) List(ctx context.Context, orgID, contactType string) ([]Contact, error) {
	q := `SELECT id, name, COALESCE(email,''), COALESCE(phone,''), type, is_active, created_at
	      FROM contacts WHERE org_id = $1`
	args := []any{orgID}
	if contactType != "" {
		q += fmt.Sprintf(" AND (type = $%d OR type = 'both')", len(args)+1)
		args = append(args, contactType)
	}
	q += " ORDER BY name"
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Contact
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Type, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, orgID, id string) (Contact, error) {
	var c Contact
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, COALESCE(email,''), COALESCE(phone,''), type, is_active, created_at
		 FROM contacts WHERE id = $1 AND org_id = $2`,
		id, orgID,
	).Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Type, &c.IsActive, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Contact{}, ErrNotFound
	}
	return c, err
}

func (s *Service) Create(ctx context.Context, orgID string, req CreateRequest) (Contact, error) {
	if req.Name == "" {
		return Contact{}, fmt.Errorf("name required")
	}
	if req.Type != "customer" && req.Type != "supplier" && req.Type != "both" {
		return Contact{}, fmt.Errorf("type must be customer, supplier, or both")
	}
	var c Contact
	err := s.pool.QueryRow(ctx,
		`INSERT INTO contacts (org_id, name, email, phone, type)
		 VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), $5)
		 RETURNING id, name, COALESCE(email,''), COALESCE(phone,''), type, is_active, created_at`,
		orgID, req.Name, req.Email, req.Phone, req.Type,
	).Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Type, &c.IsActive, &c.CreatedAt)
	return c, err
}

func (s *Service) Update(ctx context.Context, orgID, id string, req UpdateRequest) (Contact, error) {
	existing, err := s.Get(ctx, orgID, id)
	if err != nil {
		return Contact{}, err
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Email != nil {
		existing.Email = *req.Email
	}
	if req.Phone != nil {
		existing.Phone = *req.Phone
	}
	if req.Type != nil {
		existing.Type = *req.Type
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	var c Contact
	err = s.pool.QueryRow(ctx,
		`UPDATE contacts SET name=$1, email=NULLIF($2,''), phone=NULLIF($3,''), type=$4, is_active=$5
		 WHERE id=$6 AND org_id=$7
		 RETURNING id, name, COALESCE(email,''), COALESCE(phone,''), type, is_active, created_at`,
		existing.Name, existing.Email, existing.Phone, existing.Type, existing.IsActive, id, orgID,
	).Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Type, &c.IsActive, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Contact{}, ErrNotFound
	}
	return c, err
}
