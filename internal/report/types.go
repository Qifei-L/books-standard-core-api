package report

type TrialBalanceLine struct {
	AccountCode string  `json:"accountCode"`
	AccountName string  `json:"accountName"`
	AccountType string  `json:"accountType"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
	Balance     float64 `json:"balance"`
}

type TrialBalance struct {
	DateFrom    string             `json:"dateFrom"`
	DateTo      string             `json:"dateTo"`
	Lines       []TrialBalanceLine `json:"lines"`
	TotalDebit  float64            `json:"totalDebit"`
	TotalCredit float64            `json:"totalCredit"`
	Balanced    bool               `json:"balanced"`
}

type PLLine struct {
	AccountCode string  `json:"accountCode"`
	AccountName string  `json:"accountName"`
	Amount      float64 `json:"amount"`
}

type ProfitAndLoss struct {
	DateFrom      string   `json:"dateFrom"`
	DateTo        string   `json:"dateTo"`
	Revenue       []PLLine `json:"revenue"`
	TotalRevenue  float64  `json:"totalRevenue"`
	Expenses      []PLLine `json:"expenses"`
	TotalExpenses float64  `json:"totalExpenses"`
	NetIncome     float64  `json:"netIncome"`
}

type BSLine struct {
	AccountCode string  `json:"accountCode"`
	AccountName string  `json:"accountName"`
	Balance     float64 `json:"balance"`
}

type BalanceSheet struct {
	AsAt               string   `json:"asAt"`
	Assets             []BSLine `json:"assets"`
	TotalAssets        float64  `json:"totalAssets"`
	Liabilities        []BSLine `json:"liabilities"`
	TotalLiabilities   float64  `json:"totalLiabilities"`
	Equity             []BSLine `json:"equity"`
	RetainedEarnings   float64  `json:"retainedEarnings"`
	TotalEquity        float64  `json:"totalEquity"`
	TotalLiabEquity    float64  `json:"totalLiabEquity"`
}

type AgingLine struct {
	ContactID   string  `json:"contactId"`
	ContactName string  `json:"contactName"`
	Current     float64 `json:"current"`
	Days1to30   float64 `json:"days1to30"`
	Days31to60  float64 `json:"days31to60"`
	Days61to90  float64 `json:"days61to90"`
	Days90Plus  float64 `json:"days90plus"`
	Total       float64 `json:"total"`
}

type AgingReport struct {
	AsAt        string      `json:"asAt"`
	Lines       []AgingLine `json:"lines"`
	Current     float64     `json:"current"`
	Days1to30   float64     `json:"days1to30"`
	Days31to60  float64     `json:"days31to60"`
	Days61to90  float64     `json:"days61to90"`
	Days90Plus  float64     `json:"days90plus"`
	Total       float64     `json:"total"`
}
