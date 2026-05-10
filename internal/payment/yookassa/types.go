package yookassa

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Confirmation struct {
	Type              string `json:"type"`
	ConfirmationToken string `json:"confirmation_token,omitempty"`
	ConfirmationURL   string `json:"confirmation_url,omitempty"`
	ReturnURL         string `json:"return_url,omitempty"`
}

type CreatePaymentRequest struct {
	Amount       Amount            `json:"amount"`
	Capture      bool              `json:"capture"`
	Confirmation Confirmation      `json:"confirmation"`
	Description  string            `json:"description,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type Payment struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"`
	Amount       Amount            `json:"amount"`
	Description  string            `json:"description"`
	Paid         bool              `json:"paid"`
	Confirmation *Confirmation     `json:"confirmation,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type WebhookEvent struct {
	Type   string  `json:"type"`
	Event  string  `json:"event"`
	Object Payment `json:"object"`
}
