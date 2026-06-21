package invoice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("invoice not found")
	ErrInvalidState = errors.New("invalid state for this operation")
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) List(ctx context.Context, orgID string, p ListParams) ([]Invoice, error) {
	q := `SELECT i.id, i.contact_id, c.name, i.number, i.issue_date, COALESCE(i.due_date::text,''),
	             i.status, i.subtotal, i.tax_amount, i.total, i.amount_due, i.currency,
	             COALESCE(i.notes,''), i.created_at, i.updated_at
	      FROM invoices i
	      JOIN contacts c ON c.id = i.contact_id
	      WHERE i.org_id = $1`
	args := []any{orgID}
	if p.Status != "" {
		args = append(args, p.Status)
		q += fmt.Sprintf(" AND i.status = $%d", len(args))
	}
	if p.ContactID != "" {
		args = append(args, p.ContactID)
		q += fmt.Sprintf(" AND i.contact_id = $%d", len(args))
	}
	if p.DateFrom != "" {
		args = append(args, p.DateFrom)
		q += fmt.Sprintf(" AND i.issue_date >= $%d", len(args))
	}
	if p.DateTo != "" {
		args = append(args, p.DateTo)
		q += fmt.Sprintf(" AND i.issue_date <= $%d", len(args))
	}
	q += " ORDER BY i.issue_date DESC, i.number DESC"

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(
			&inv.ID, &inv.ContactID, &inv.ContactName, &inv.Number, &inv.IssueDate,
			&inv.DueDate, &inv.Status, &inv.Subtotal, &inv.TaxAmount, &inv.Total,
			&inv.AmountDue, &inv.Currency, &inv.Notes, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, orgID, id string) (Invoice, error) {
	var inv Invoice
	err := s.pool.QueryRow(ctx,
		`SELECT i.id, i.contact_id, c.name, i.number, i.issue_date, COALESCE(i.due_date::text,''),
		        i.status, i.subtotal, i.tax_amount, i.total, i.amount_due, i.currency,
		        COALESCE(i.notes,''), i.created_at, i.updated_at
		 FROM invoices i JOIN contacts c ON c.id = i.contact_id
		 WHERE i.id = $1 AND i.org_id = $2`,
		id, orgID,
	).Scan(
		&inv.ID, &inv.ContactID, &inv.ContactName, &inv.Number, &inv.IssueDate,
		&inv.DueDate, &inv.Status, &inv.Subtotal, &inv.TaxAmount, &inv.Total,
		&inv.AmountDue, &inv.Currency, &inv.Notes, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, err
	}
	inv.Lines, err = s.getLines(ctx, id)
	return inv, err
}

func (s *Service) Create(ctx context.Context, orgID string, req CreateRequest) (Invoice, error) {
	if err := validateLines(req.Lines); err != nil {
		return Invoice{}, err
	}
	subtotal, taxAmount, total := calcTotals(req.Lines)
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Invoice{}, err
	}
	defer tx.Rollback(ctx)

	var invID string
	err = tx.QueryRow(ctx,
		`INSERT INTO invoices (org_id, contact_id, number, issue_date, due_date, currency, notes,
		                       subtotal, tax_amount, total, amount_due)
		 VALUES ($1,$2,$3,$4,NULLIF($5,'')::date,$6,NULLIF($7,''),$8,$9,$10,$10)
		 RETURNING id`,
		orgID, req.ContactID, req.Number, req.IssueDate, req.DueDate,
		currency, req.Notes, subtotal, taxAmount, total,
	).Scan(&invID)
	if err != nil {
		return Invoice{}, err
	}
	if err := insertLines(ctx, tx, invID, req.Lines); err != nil {
		return Invoice{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, err
	}
	return s.Get(ctx, orgID, invID)
}

func (s *Service) Approve(ctx context.Context, orgID, id string) (Invoice, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE invoices SET status='approved', updated_at=now()
		 WHERE id=$1 AND org_id=$2 AND status='draft'`,
		id, orgID,
	)
	if err != nil {
		return Invoice{}, err
	}
	if tag.RowsAffected() == 0 {
		return Invoice{}, fmt.Errorf("%w: invoice must be in draft status", ErrInvalidState)
	}
	return s.Get(ctx, orgID, id)
}

func (s *Service) Void(ctx context.Context, orgID, id string) (Invoice, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE invoices SET status='voided', updated_at=now()
		 WHERE id=$1 AND org_id=$2 AND status IN ('draft','approved')`,
		id, orgID,
	)
	if err != nil {
		return Invoice{}, err
	}
	if tag.RowsAffected() == 0 {
		return Invoice{}, fmt.Errorf("%w: cannot void a paid invoice", ErrInvalidState)
	}
	return s.Get(ctx, orgID, id)
}

func (s *Service) RecordPayment(ctx context.Context, orgID, invID string, req RecordPaymentRequest) (Invoice, error) {
	if req.Amount <= 0 {
		return Invoice{}, fmt.Errorf("amount must be positive")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Invoice{}, err
	}
	defer tx.Rollback(ctx)

	var amountDue float64
	err = tx.QueryRow(ctx,
		`SELECT amount_due FROM invoices WHERE id=$1 AND org_id=$2 AND status='approved' FOR UPDATE`,
		invID, orgID,
	).Scan(&amountDue)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, fmt.Errorf("%w: invoice not found or not approved", ErrInvalidState)
	}
	if err != nil {
		return Invoice{}, err
	}
	if req.Amount > amountDue+0.001 {
		return Invoice{}, fmt.Errorf("payment amount %.2f exceeds amount due %.2f", req.Amount, amountDue)
	}

	newDue := amountDue - req.Amount
	newStatus := "approved"
	if newDue < 0.001 {
		newStatus = "paid"
		newDue = 0
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO payments (org_id, type, reference_id, date, amount, account_code, reference)
		 VALUES ($1,'ar',$2,$3,$4,$5,NULLIF($6,''))`,
		orgID, invID, req.Date, req.Amount, req.AccountCode, req.Reference,
	)
	if err != nil {
		return Invoice{}, err
	}
	_, err = tx.Exec(ctx,
		`UPDATE invoices SET amount_due=$1, status=$2, updated_at=now() WHERE id=$3`,
		newDue, newStatus, invID,
	)
	if err != nil {
		return Invoice{}, err
	}
	if err := s.postPaymentJournal(ctx, tx, orgID, invID, req); err != nil {
		return Invoice{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Invoice{}, err
	}
	return s.Get(ctx, orgID, invID)
}

func (s *Service) postPaymentJournal(ctx context.Context, tx pgx.Tx, orgID, invID string, req RecordPaymentRequest) error {
	var entryID string
	err := tx.QueryRow(ctx,
		`INSERT INTO journal_entries (org_id, date, description, source_type, source_id)
		 VALUES ($1,$2,'AR Payment',$3,$4) RETURNING id`,
		orgID, req.Date, "payment", invID,
	).Scan(&entryID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO journal_lines (entry_id, account_code, description, debit, credit, line_no) VALUES
		 ($1,$2,'Cash receipt',$3,0,1),
		 ($1,'1100','AR cleared',0,$3,2)`,
		entryID, req.AccountCode, req.Amount,
	)
	return err
}

func (s *Service) getLines(ctx context.Context, invID string) ([]LineItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, description, quantity, unit_price, tax_rate, amount, account_code, line_no
		 FROM invoice_lines WHERE invoice_id=$1 ORDER BY line_no`,
		invID,
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

func insertLines(ctx context.Context, tx pgx.Tx, invID string, lines []LineItem) error {
	for i, l := range lines {
		_, err := tx.Exec(ctx,
			`INSERT INTO invoice_lines (invoice_id, description, quantity, unit_price, tax_rate, amount, account_code, line_no)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			invID, l.Description, l.Quantity, l.UnitPrice, l.TaxRate, l.Amount, l.AccountCode, i+1,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateLines(lines []LineItem) error {
	if len(lines) == 0 {
		return fmt.Errorf("at least one line item required")
	}
	for i, l := range lines {
		if l.Description == "" {
			return fmt.Errorf("line %d: description required", i+1)
		}
		if l.AccountCode == "" {
			return fmt.Errorf("line %d: accountCode required", i+1)
		}
	}
	return nil
}

func calcTotals(lines []LineItem) (subtotal, taxAmount, total float64) {
	for _, l := range lines {
		amt := l.Quantity * l.UnitPrice
		tax := amt * l.TaxRate
		subtotal += amt
		taxAmount += tax
	}
	total = subtotal + taxAmount
	return
}

func insertJournalForInvoice(ctx context.Context, tx pgx.Tx, orgID, invID, issueDate string, lines []LineItem, total float64) error {
	var entryID string
	err := tx.QueryRow(ctx,
		`INSERT INTO journal_entries (org_id, date, description, source_type, source_id)
		 VALUES ($1,$2,'Sales Invoice','invoice',$3) RETURNING id`,
		orgID, issueDate, invID,
	).Scan(&entryID)
	if err != nil {
		return err
	}
	lineNo := 1
	_, err = tx.Exec(ctx,
		`INSERT INTO journal_lines (entry_id, account_code, description, debit, credit, line_no)
		 VALUES ($1,'1100','AR',$2,0,$3)`,
		entryID, total, lineNo,
	)
	if err != nil {
		return err
	}
	lineNo++
	for _, l := range lines {
		amt := l.Quantity * l.UnitPrice
		tax := amt * l.TaxRate
		if amt != 0 {
			_, err = tx.Exec(ctx,
				`INSERT INTO journal_lines (entry_id, account_code, description, debit, credit, line_no)
				 VALUES ($1,$2,$3,0,$4,$5)`,
				entryID, l.AccountCode, l.Description, amt, lineNo,
			)
			if err != nil {
				return err
			}
			lineNo++
		}
		if tax != 0 {
			_, err = tx.Exec(ctx,
				`INSERT INTO journal_lines (entry_id, account_code, description, debit, credit, line_no)
				 VALUES ($1,'2100','VAT',0,$2,$3)`,
				entryID, tax, lineNo,
			)
			if err != nil {
				return err
			}
			lineNo++
		}
	}
	return nil
}

func defaultNumber() string {
	return fmt.Sprintf("INV-%s", time.Now().Format("20060102-150405"))
}
