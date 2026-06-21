package account

type Account struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}

type CreateRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type UpdateRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"isActive"`
}
