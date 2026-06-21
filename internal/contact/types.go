package contact

type Contact struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Type      string `json:"type"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}

type CreateRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
	Type  string `json:"type"`
}

type UpdateRequest struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Type     *string `json:"type"`
	IsActive *bool   `json:"isActive"`
}
