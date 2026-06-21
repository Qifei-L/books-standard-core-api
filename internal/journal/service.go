package journal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("journal entry not found")
	ErrInvalidState = errors.New("invalid state for this operation")
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) List(ctx context.Context, orgID string, p ListParams) ([]JournalEntry, error) {
	q := `SELECT id, date, COALESCE(reference,''), description, status, COALESCE(source_type,''), created_at
	      FROM journal_entries WHERE org_id = $1`
	args := []any{orgID}
	if p.DateFrom != "" {
		args = append(args, p.DateFrom)
		q += fmt.Sprintf(" AND date >= $%d", len(args))
	}
	if p.DateTo != "" {
		args = append(args, p.DateTo)
		q += fmt.Sprintf(" AND date <= $%d", len(args))
	}
	q += " ORDER BY date DESC, created_at DESC"

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JournalEntry
	for rows.Next() {
		var e JournalEntry
		if err := rows.Scan(&e.ID, &e.Date, &e.Reference, &e.Description, &e.Status, &e.SourceType, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, orgID, id string) (JournalEntry, error) {
	var e JournalEntry
	err := s.pool.QueryRow(ctx,
		`SELECT id, date, COALESCE(reference,''), description, status, COALESCE(source_type,''), created_at
		 FROM journal_entries WHERE id=$1 AND org_id=$2`,
		id, orgID,
	).Scan(&e.ID, &e.Date, &e.Reference, &e.Description, &e.Status, &e.SourceType, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return JournalEntry{}, ErrNotFound
	}
	if err != nil {
		return JournalEntry{}, err
	}
	e.Lines, err = s.getLines(ctx, id)
	return e, err
}

func (s *Service) Create(ctx context.Context, orgID string, req CreateRequest) (JournalEntry, error) {
	if req.Description == "" {
		return JournalEntry{}, fmt.Errorf("description required")
	}
	if err := validateLines(req.Lines); err != nil {
		return JournalEntry{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return JournalEntry{}, err
	}
	defer tx.Rollback(ctx)

	var entryID string
	err = tx.QueryRow(ctx,
		`INSERT INTO journal_entries (org_id, date, reference, description, source_type)
		 VALUES ($1,$2,NULLIF($3,''),$4,'manual') RETURNING id`,
		orgID, req.Date, req.Reference, req.Description,
	).Scan(&entryID)
	if err != nil {
		return JournalEntry{}, err
	}
	for i, l := range req.Lines {
		_, err = tx.Exec(ctx,
			`INSERT INTO journal_lines (entry_id, account_code, description, debit, credit, line_no)
			 VALUES ($1,$2,NULLIF($3,''),$4,$5,$6)`,
			entryID, l.AccountCode, l.Description, l.Debit, l.Credit, i+1,
		)
		if err != nil {
			return JournalEntry{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return JournalEntry{}, err
	}
	return s.Get(ctx, orgID, entryID)
}

func (s *Service) Void(ctx context.Context, orgID, id string) (JournalEntry, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE journal_entries SET status='voided' WHERE id=$1 AND org_id=$2 AND status='posted' AND source_type='manual'`,
		id, orgID,
	)
	if err != nil {
		return JournalEntry{}, err
	}
	if tag.RowsAffected() == 0 {
		return JournalEntry{}, fmt.Errorf("%w: only posted manual entries can be voided", ErrInvalidState)
	}
	return s.Get(ctx, orgID, id)
}

func (s *Service) getLines(ctx context.Context, entryID string) ([]JournalLine, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, account_code, COALESCE(description,''), debit, credit, line_no
		 FROM journal_lines WHERE entry_id=$1 ORDER BY line_no`,
		entryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lines []JournalLine
	for rows.Next() {
		var l JournalLine
		if err := rows.Scan(&l.ID, &l.AccountCode, &l.Description, &l.Debit, &l.Credit, &l.LineNo); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func validateLines(lines []JournalLine) error {
	if len(lines) < 2 {
		return fmt.Errorf("at least two lines required")
	}
	var totalDebit, totalCredit float64
	for i, l := range lines {
		if l.AccountCode == "" {
			return fmt.Errorf("line %d: accountCode required", i+1)
		}
		if l.Debit < 0 || l.Credit < 0 {
			return fmt.Errorf("line %d: amounts must be non-negative", i+1)
		}
		totalDebit += l.Debit
		totalCredit += l.Credit
	}
	if abs(totalDebit-totalCredit) > 0.001 {
		return fmt.Errorf("journal entry must balance: debit %.2f ≠ credit %.2f", totalDebit, totalCredit)
	}
	return nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
