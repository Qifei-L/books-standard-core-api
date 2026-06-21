package invoice

type LineItem struct {
	ID          string  `json:"id,omitempty"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	TaxRate     float64 `json:"taxRate"`
	Amount      float64 `json:"amount"`
	AccountCode string  `json:"accountCode"`
	LineNo      int     `json:"lineNo"`
}

type Invoice struct {
	ID          string     `json:"id"`
	ContactID   string     `json:"contactId"`
	ContactName string     `json:"contactName,omitempty"`
	Number      string     `json:"number"`
	IssueDate   string     `json:"issueDate"`
	DueDate     string     `json:"dueDate,omitempty"`
	Status      string     `json:"status"`
	Subtotal    string     `json:"subtotal"`
	TaxAmount   string     `json:"taxAmount"`
	Total       string     `json:"total"`
	AmountDue   string     `json:"amountDue"`
	Currency    string     `json:"currency"`
	Notes       string     `json:"notes,omitempty"`
	Lines       []LineItem `json:"lines,omitempty"`
	CreatedAt   string     `json:"createdAt"`
	UpdatedAt   string     `json:"updatedAt"`
}

type CreateRequest struct {
	ContactID string     `json:"contactId"`
	Number    string     `json:"number"`
	IssueDate string     `json:"issueDate"`
	DueDate   string     `json:"dueDate"`
	Currency  string     `json:"currency"`
	Notes     string     `json:"notes"`
	Lines     []LineItem `json:"lines"`
}

type UpdateRequest struct {
	ContactID *string    `json:"contactId"`
	IssueDate *string    `json:"issueDate"`
	DueDate   *string    `json:"dueDate"`
	Notes     *string    `json:"notes"`
	Lines     []LineItem `json:"lines"`
}

type RecordPaymentRequest struct {
	Date        string  `json:"date"`
	Amount      float64 `json:"amount"`
	AccountCode string  `json:"accountCode"`
	Reference   string  `json:"reference"`
}

type ListParams struct {
	Status    string
	ContactID string
	DateFrom  string
	DateTo    string
}
