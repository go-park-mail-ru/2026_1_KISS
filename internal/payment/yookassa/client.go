package yookassa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	shopID    string
	secretKey string
	baseURL   string
	http      Doer
}

func NewClient(shopID, secretKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.yookassa.ru"
	}
	return &Client{
		shopID:    shopID,
		secretKey: secretKey,
		baseURL:   baseURL,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

func NewClientWithDoer(shopID, secretKey, baseURL string, doer Doer) *Client {
	if baseURL == "" {
		baseURL = "https://api.yookassa.ru"
	}
	return &Client{
		shopID:    shopID,
		secretKey: secretKey,
		baseURL:   baseURL,
		http:      doer,
	}
}

func (c *Client) authHeader() string {
	cred := c.shopID + ":" + c.secretKey
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(cred))
}

func (c *Client) CreatePayment(ctx context.Context, idempotenceKey string, body CreatePaymentRequest) (*Payment, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v3/payments", bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", idempotenceKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrYooKassaUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: yookassa returned %d: %s", domain.ErrPaymentFailed, resp.StatusCode, string(respBody))
	}

	var payment Payment
	if err := json.Unmarshal(respBody, &payment); err != nil {
		return nil, fmt.Errorf("unmarshal payment: %w", err)
	}
	return &payment, nil
}

func (c *Client) GetPayment(ctx context.Context, paymentID string) (*Payment, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v3/payments/"+paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrYooKassaUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: yookassa returned %d: %s", domain.ErrPaymentFailed, resp.StatusCode, string(respBody))
	}

	var payment Payment
	if err := json.Unmarshal(respBody, &payment); err != nil {
		return nil, fmt.Errorf("unmarshal payment: %w", err)
	}
	return &payment, nil
}

func KopeksToString(kopeks int64) string {
	return fmt.Sprintf("%d.%02d", kopeks/100, kopeks%100)
}
