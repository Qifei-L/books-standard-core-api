package report

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) TrialBalance(ctx context.Context, orgID, dateFrom, dateTo string) (TrialBalance, error) {
	if dateFrom == "" || dateTo == "" {
		return TrialBalance{}, fmt.Errorf("dateFrom and dateTo required")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT a.code, a.name, a.type,
		       COALESCE(SUM(jl.debit),0)::float8,
		       COALESCE(SUM(jl.credit),0)::float8
		FROM journal_entries je
		JOIN journal_lines jl ON jl.entry_id = je.id
		JOIN accounts a ON a.code = jl.account_code AND a.org_id = je.org_id
		WHERE je.org_id = $1
		  AND je.status = 'posted'
		  AND je.date BETWEEN $2 AND $3
		GROUP BY a.code, a.name, a.type
		ORDER BY a.code`,
		orgID, dateFrom, dateTo,
	)
	if err != nil {
		return TrialBalance{}, err
	}
	defer rows.Close()

	var lines []TrialBalanceLine
	var totalDebit, totalCredit float64
	for rows.Next() {
		var l TrialBalanceLine
		if err := rows.Scan(&l.AccountCode, &l.AccountName, &l.AccountType, &l.Debit, &l.Credit); err != nil {
			return TrialBalance{}, err
		}
		l.Balance = l.Debit - l.Credit
		totalDebit += l.Debit
		totalCredit += l.Credit
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		return TrialBalance{}, err
	}
	return TrialBalance{
		DateFrom:    dateFrom,
		DateTo:      dateTo,
		Lines:       lines,
		TotalDebit:  totalDebit,
		TotalCredit: totalCredit,
		Balanced:    abs(totalDebit-totalCredit) < 0.01,
	}, nil
}

func (s *Service) ProfitAndLoss(ctx context.Context, orgID, dateFrom, dateTo string) (ProfitAndLoss, error) {
	if dateFrom == "" || dateTo == "" {
		return ProfitAndLoss{}, fmt.Errorf("dateFrom and dateTo required")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT a.code, a.name, a.type,
		       COALESCE(SUM(jl.debit),0)::float8,
		       COALESCE(SUM(jl.credit),0)::float8
		FROM journal_entries je
		JOIN journal_lines jl ON jl.entry_id = je.id
		JOIN accounts a ON a.code = jl.account_code AND a.org_id = je.org_id
		WHERE je.org_id = $1
		  AND je.status = 'posted'
		  AND je.date BETWEEN $2 AND $3
		  AND a.type IN ('income','expense')
		GROUP BY a.code, a.name, a.type
		ORDER BY a.type, a.code`,
		orgID, dateFrom, dateTo,
	)
	if err != nil {
		return ProfitAndLoss{}, err
	}
	defer rows.Close()

	var revenue, expenses []PLLine
	var totalRevenue, totalExpenses float64
	for rows.Next() {
		var code, name, accType string
		var debit, credit float64
		if err := rows.Scan(&code, &name, &accType, &debit, &credit); err != nil {
			return ProfitAndLoss{}, err
		}
		switch accType {
		case "income":
			amt := credit - debit
			revenue = append(revenue, PLLine{AccountCode: code, AccountName: name, Amount: amt})
			totalRevenue += amt
		case "expense":
			amt := debit - credit
			expenses = append(expenses, PLLine{AccountCode: code, AccountName: name, Amount: amt})
			totalExpenses += amt
		}
	}
	if err := rows.Err(); err != nil {
		return ProfitAndLoss{}, err
	}
	return ProfitAndLoss{
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		Revenue:       revenue,
		TotalRevenue:  totalRevenue,
		Expenses:      expenses,
		TotalExpenses: totalExpenses,
		NetIncome:     totalRevenue - totalExpenses,
	}, nil
}

func (s *Service) BalanceSheet(ctx context.Context, orgID, asAt string) (BalanceSheet, error) {
	if asAt == "" {
		asAt = time.Now().Format("2006-01-02")
	}

	// Cumulative BS accounts (asset/liability/equity) up to asAt
	rows, err := s.pool.Query(ctx, `
		SELECT a.code, a.name, a.type,
		       COALESCE(SUM(jl.debit),0)::float8,
		       COALESCE(SUM(jl.credit),0)::float8
		FROM journal_entries je
		JOIN journal_lines jl ON jl.entry_id = je.id
		JOIN accounts a ON a.code = jl.account_code AND a.org_id = je.org_id
		WHERE je.org_id = $1
		  AND je.status = 'posted'
		  AND je.date <= $2
		  AND a.type IN ('asset','liability','equity')
		GROUP BY a.code, a.name, a.type
		ORDER BY a.type, a.code`,
		orgID, asAt,
	)
	if err != nil {
		return BalanceSheet{}, err
	}
	defer rows.Close()

	var assets, liabilities, equity []BSLine
	var totalAssets, totalLiabilities, totalEquity float64
	for rows.Next() {
		var code, name, accType string
		var debit, credit float64
		if err := rows.Scan(&code, &name, &accType, &debit, &credit); err != nil {
			return BalanceSheet{}, err
		}
		switch accType {
		case "asset":
			bal := debit - credit
			assets = append(assets, BSLine{AccountCode: code, AccountName: name, Balance: bal})
			totalAssets += bal
		case "liability":
			bal := credit - debit
			liabilities = append(liabilities, BSLine{AccountCode: code, AccountName: name, Balance: bal})
			totalLiabilities += bal
		case "equity":
			bal := credit - debit
			equity = append(equity, BSLine{AccountCode: code, AccountName: name, Balance: bal})
			totalEquity += bal
		}
	}
	if err := rows.Err(); err != nil {
		return BalanceSheet{}, err
	}

	// Retained earnings = cumulative net income up to asAt
	var retainedEarnings float64
	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(
		  CASE a.type
		    WHEN 'income'  THEN jl.credit - jl.debit
		    WHEN 'expense' THEN jl.debit  - jl.credit
		    ELSE 0
		  END
		), 0)::float8
		FROM journal_entries je
		JOIN journal_lines jl ON jl.entry_id = je.id
		JOIN accounts a ON a.code = jl.account_code AND a.org_id = je.org_id
		WHERE je.org_id = $1
		  AND je.status = 'posted'
		  AND je.date <= $2`,
		orgID, asAt,
	).Scan(&retainedEarnings)
	if err != nil {
		return BalanceSheet{}, err
	}

	totalEquity += retainedEarnings
	return BalanceSheet{
		AsAt:             asAt,
		Assets:           assets,
		TotalAssets:      totalAssets,
		Liabilities:      liabilities,
		TotalLiabilities: totalLiabilities,
		Equity:           equity,
		RetainedEarnings: retainedEarnings,
		TotalEquity:      totalEquity,
		TotalLiabEquity:  totalLiabilities + totalEquity,
	}, nil
}

func (s *Service) AgedReceivables(ctx context.Context, orgID, asAt string) (AgingReport, error) {
	return s.aging(ctx, orgID, asAt, "ar", "invoices", "contact_id")
}

func (s *Service) AgedPayables(ctx context.Context, orgID, asAt string) (AgingReport, error) {
	return s.aging(ctx, orgID, asAt, "ap", "bills", "contact_id")
}

func (s *Service) aging(ctx context.Context, orgID, asAt, paymentType, table, contactCol string) (AgingReport, error) {
	if asAt == "" {
		asAt = time.Now().Format("2006-01-02")
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT c.id, c.name, t.due_date::text, t.amount_due::float8
		FROM %s t
		JOIN contacts c ON c.id = t.%s
		WHERE t.org_id = $1
		  AND t.status IN ('approved')
		  AND t.amount_due > 0
		  AND (t.due_date IS NULL OR t.due_date <= $2::date)
		ORDER BY c.name, t.due_date`, table, contactCol),
		orgID, asAt,
	)
	if err != nil {
		return AgingReport{}, err
	}
	defer rows.Close()

	asAtDate, _ := time.Parse("2006-01-02", asAt)
	byContact := map[string]*AgingLine{}
	var order []string

	for rows.Next() {
		var contactID, contactName, dueDateStr string
		var amountDue float64
		if err := rows.Scan(&contactID, &contactName, &dueDateStr, &amountDue); err != nil {
			return AgingReport{}, err
		}
		if _, ok := byContact[contactID]; !ok {
			byContact[contactID] = &AgingLine{ContactID: contactID, ContactName: contactName}
			order = append(order, contactID)
		}
		line := byContact[contactID]
		dueDate, _ := time.Parse("2006-01-02", dueDateStr)
		days := int(asAtDate.Sub(dueDate).Hours() / 24)
		switch {
		case days <= 0:
			line.Current += amountDue
		case days <= 30:
			line.Days1to30 += amountDue
		case days <= 60:
			line.Days31to60 += amountDue
		case days <= 90:
			line.Days61to90 += amountDue
		default:
			line.Days90Plus += amountDue
		}
		line.Total += amountDue
	}
	if err := rows.Err(); err != nil {
		return AgingReport{}, err
	}

	report := AgingReport{AsAt: asAt}
	for _, id := range order {
		l := *byContact[id]
		report.Lines = append(report.Lines, l)
		report.Current += l.Current
		report.Days1to30 += l.Days1to30
		report.Days31to60 += l.Days31to60
		report.Days61to90 += l.Days61to90
		report.Days90Plus += l.Days90Plus
		report.Total += l.Total
	}
	return report, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
