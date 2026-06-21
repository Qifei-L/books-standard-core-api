package journal

type JournalLine struct {
	ID          string  `json:"id,omitempty"`
	AccountCode string  `json:"accountCode"`
	Description string  `json:"description,omitempty"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
	LineNo      int     `json:"lineNo"`
}

type JournalEntry struct {
	ID          string        `json:"id"`
	Date        string        `json:"date"`
	Reference   string        `json:"reference,omitempty"`
	Description string        `json:"description"`
	Status      string        `json:"status"`
	SourceType  string        `json:"sourceType,omitempty"`
	Lines       []JournalLine `json:"lines,omitempty"`
	CreatedAt   string        `json:"createdAt"`
}

type CreateRequest struct {
	Date        string        `json:"date"`
	Reference   string        `json:"reference"`
	Description string        `json:"description"`
	Lines       []JournalLine `json:"lines"`
}

type ListParams struct {
	DateFrom string
	DateTo   string
}
