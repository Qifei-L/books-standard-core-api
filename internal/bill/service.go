package bill

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("bill not found")
	ErrInvalidState = errors.New("invalid state for this operation")
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) List(ctx context.Context, orgID string, p ListParams) ([]Bill, error) {
	q := `SELECT b.id, b.contact_id, c.name, COALESCE(b.number,''), COALESCE(b.reference,''),
	             b.issue_date, COALESCE(b.due_date::text,''), b.status,
	             b.subtotal, b.tax_amount, b.total, b.amount_due, b.currency,
	             COALESCE(b.notes,''), b.created_at, b.updated_at
	      FROM bills b JOIN contacts c ON c.id = b.contact_id
	      WHERE b.org_id = $1`
	args := []any{orgID}
	if p.Status != "" {
		args = append(args, p.Status)
		q += fmt.Sprintf(" AND b.status = $%d", len(args))
	}
	if p.ContactID != "" {
		args = append(args, p.ContactID)
		q += fmt.Sprintf(" AND b.contact_id = $%d", len(args))
	}
	if p.DateFrom != "" {
		args = append(args, p.DateFrom)
		q += fmt.Sprintf(" AND b.issue_date >= $%d", len(args))
	}
	if p.DateTo != "" {
		args = append(args, p.DateTo)
		q += fmt.Sprintf(" AND b.issue_date <= $%d", len(args))
	}
	q += " ORDER BY b.issue_date DESC"

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Bill
	for rows.Next() {
		var b Bill
		if err := rows.Scan(
			&b.ID, &b.ContactID, &b.ContactName, &b.Number, &b.Reference,
			&b.IssueDate, &b.DueDate, &b.Status, &b.Subtotal, &b.TaxAmount,
			&b.Total, &b.AmountDue, &b.Currency, &b.Notes, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, orgID, id string) (Bill, error) {
	var b Bill
	err := s.pool.QueryRow(ctx,
		`SELECT b.id, b.contact_id, c.name, COALESCE(b.number,''), COALESCE(b.reference,''),
		        b.issue_date, COALESCE(b.due_date::text,''), b.status,
		        b.subtotal, b.tax_amount, b.total, b.amount_due, b.currency,
		        COALESCE(b.notes,''), b.created_at, b.updated_at
		 FROM bills b JOIN contacts c ON c.id = b.contact_id
		 WHERE b.id=$1 AND b.org_id=$2`,
		id, orgID,
	).Scan(
		&b.ID, &b.ContactID, &b.ContactName, &b.Number, &b.Reference,
		&b.IssueDate, &b.DueDate, &b.Status, &b.Subtotal, &b.TaxAmount,
		&b.Total, &b.AmountDue, &b.Currency, &b.Notes, &b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bill{}, ErrNotFound
	}
	if err != nil {
		return Bill{}, err
	}
	b.Lines, err = s.getLines(ctx, id)
	return b, err
}

func (s *Service) Create(ctx context.Context, orgID string, req CreateRequest) (Bill, error) {
	if len(req.Lines) == 0 {
		return Bill{}, fmt.Errorf("at least one line item required")
	}
	subtotal, taxAmount, total := calcTotals(req.Lines)
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Bill{}, err
	}
	defer tx.Rollback(ctx)

	var billID string
	err = tx.QueryRow(ctx,
		`INSERT INTO bills (org_id, contact_id, number, reference, issue_date, due_date, currency, notes,
		                    subtotal, tax_amount, total, amount_due)
		 VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,NULLIF($6,'')::date,$7,NULLIF($8,''),$9,$10,$11,$11)
		 RETURNING id`,
		orgID, req.ContactID, req.Number, req.Reference, req.IssueDate, req.DueDate,
		currency, req.Notes, subtotal, taxAmount, total,
	).Scan(&billID)
	if err != nil {
		return Bill{}, err
	}
	if err := insertLines(ctx, tx, billID, req.Lines); err != nil {
		return Bill{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Bill{}, err
	}
	return s.Get(ctx, orgID, billID)
}

func (s *Service) Approve(ctx context.Context, orgID, id string) (Bill, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE bills SET status='approved', updated_at=now() WHERE id=$1 AND org_id=$2 AND status='draft'`,
		id, orgID,
	)
	if err != nil {
		return Bill{}, err
	}
	if tag.RowsAffected() == 0 {
		return Bill{}, fmt.Errorf("%w: bill must be in draft status", ErrInvalidState)
	}
	return s.Get(ctx, orgID, id)
}

func (s *Service) Void(ctx context.Context, orgID, id string) (Bill, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE bills SET status='voided', updated_at=now()
		 WHERE id=$1 AND org_id=$2 AND status IN ('draft','approved')`,
		id, orgID,
	)
	if err != nil {
		return Bill{}, err
	}
	if tag.RowsAffected() == 0 {
		return Bill{}, fmt.Errorf("%w: cannot void a paid bill", ErrInvalidState)
	}
	return s.Get(ctx, orgID, id)
}

func (s *Service) RecordPayment(ctx context.Context, orgID, billID string, req RecordPaymentRequest) (Bill, error) {
	if req.Amount <= 0 {
		return Bill{}, fmt.Errorf("amount must be positive")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Bill{}, err
	}
	defer tx.Rollback(ctx)

	var amountDue float64
	err = tx.QueryRow(ctx,
		`SELECT amount_due FROM bills WHERE id=$1 AND org_id=$2 AND status='approved' FOR UPDATE`,
		billID, orgID,
	).Scan(&amountDue)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bill{}, fmt.Errorf("%w: bill not found or not approved", ErrInvalidState)
	}
	if err != nil {
		return Bill{}, err
	}
	if req.Amount > amountDue+0.001 {
		return Bill{}, fmt.Errorf("payment %.2f exceeds amount due %.2f", req.Amount, amountDue)
	}

	newDue := amountDue - req.Amount
	newStatus := "approved"
	if newDue < 0.001 {
		newStatus = "paid"
		newDue = 0
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO payments (org_id, type, reference_id, date, amount, account_code, reference)
		 VALUES ($1,'ap',$2,$3,$4,$5,NULLIF($6,''))`,
		orgID, billID, req.Date, req.Amount, req.AccountCode, req.Reference,
	)
	if err != nil {
		return Bill{}, err
	}
	_, err = tx.Exec(ctx,
		`UPDATE bills SET amount_due=$1, status=$2, updated_at=now() WHERE id=$3`,
		newDue, newStatus, billID,
	)
	if err != nil {
		return Bill{}, err
	}
	if err := postPaymentJournal(ctx, tx, orgID, billID, req); err != nil {
		return Bill{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Bill{}, err
	}
	return s.Get(ctx, orgID, billID)
}

func postPaymentJournal(ctx context.Context, tx pgx.Tx, orgID, billID string, req RecordPaymentRequest) error {
	var entryID string
	err := tx.QueryRow(ctx,
		`INSERT INTO journal_entries (org_id, date, description, source_type, source_id)
		 VALUES ($1,$2,'AP Payment','payment',$3) RETURNING id`,
		orgID, req.Date, billID,
	).Scan(&entryID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO journal_lines (entry_id, account_code, description, debit, credit, line_no) VALUES
		 ($1,'2000','AP cleared',$2,0,1),
		 ($1,$3,'Cash payment',0,$2,2)`,
		entryID, req.Amount, req.AccountCode,
	)
	return err
}

func (s *Service) getLines(ctx context.Context, billID string) ([]LineItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, description, quantity, unit_price, tax_rate, amount, account_code, line_no
		 FROM bill_lines WHERE bill_id=$1 ORDER BY line_no`,
		billID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lines []LineItem
	for rows.Next() {
		var l LineItem
		if err := rows.Scan(&l.ID, &l.Description, &l.Quantity, &l.UnitPrice, &l.TaxRate, &l.Amount, &l.AccountCode, &l.LineNo); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func insertLines(ctx context.Context, tx pgx.Tx, billID string, lines []LineItem) error {
	for i, l := range lines {
		_, err := tx.Exec(ctx,
			`INSERT INTO bill_lines (bill_id, description, quantity, unit_price, tax_rate, amount, account_code, line_no)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			billID, l.Description, l.Quantity, l.UnitPrice, l.TaxRate, l.Amount, l.AccountCode, i+1,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func calcTotals(lines []LineItem) (subtotal, taxAmount, total float64) {
	for _, l := range lines {
		amt := l.Quantity * l.UnitPrice
		subtotal += amt
		taxAmount += amt * l.TaxRate
	}
	total = subtotal + taxAmount
	return
}
