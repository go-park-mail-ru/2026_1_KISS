package yookassa

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestCreatePayment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v3/payments" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Idempotence-Key") == "" {
			t.Fatal("missing Idempotence-Key header")
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Fatalf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		body, _ := io.ReadAll(r.Body)
		var req CreatePaymentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if req.Amount.Value != "999.00" || req.Amount.Currency != "RUB" {
			t.Fatalf("unexpected amount: %+v", req.Amount)
		}
		if req.Confirmation.Type != "embedded" {
			t.Fatalf("unexpected confirmation type: %s", req.Confirmation.Type)
		}

		resp := Payment{
			ID:     "27f9e7e0-000f-5000-9000-1de4d4eaf6b1",
			Status: "pending",
			Amount: req.Amount,
			Confirmation: &Confirmation{
				Type:              "embedded",
				ConfirmationToken: "ct-token-xyz",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test_shop", "test_secret_key", server.URL)

	payment, err := client.CreatePayment(context.Background(), "00000000-0000-4000-8000-000000000001", CreatePaymentRequest{
		Amount:       Amount{Value: "999.00", Currency: "RUB"},
		Capture:      true,
		Confirmation: Confirmation{Type: "embedded"},
		Description:  "Pro subscription",
		Metadata:     map[string]string{"user_id": "42", "plan": "pro"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.ID != "27f9e7e0-000f-5000-9000-1de4d4eaf6b1" {
		t.Fatalf("unexpected payment id: %s", payment.ID)
	}
	if payment.Confirmation == nil || payment.Confirmation.ConfirmationToken != "ct-token-xyz" {
		t.Fatalf("missing confirmation token in response: %+v", payment.Confirmation)
	}
}

func TestCreatePayment_YookassaError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","code":"invalid_request"}`))
	}))
	defer server.Close()

	client := NewClient("test_shop", "test_secret_key", server.URL)

	_, err := client.CreatePayment(context.Background(), "k", CreatePaymentRequest{Amount: Amount{Value: "0.00", Currency: "RUB"}, Confirmation: Confirmation{Type: "embedded"}})
	if !errors.Is(err, domain.ErrPaymentFailed) {
		t.Fatalf("expected ErrPaymentFailed, got %v", err)
	}
}

func TestGetPayment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient("test_shop", "test_secret_key", server.URL)

	_, err := client.GetPayment(context.Background(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetPayment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/payments/abc" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Payment{ID: "abc", Status: "succeeded", Paid: true, Amount: Amount{Value: "999.00", Currency: "RUB"}})
	}))
	defer server.Close()

	client := NewClient("test_shop", "test_secret_key", server.URL)

	p, err := client.GetPayment(context.Background(), "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != "succeeded" || !p.Paid {
		t.Fatalf("unexpected payment: %+v", p)
	}
}

func TestKopeksToString(t *testing.T) {
	cases := map[int64]string{
		0:      "0.00",
		1:      "0.01",
		99:     "0.99",
		100:    "1.00",
		99900:  "999.00",
		199999: "1999.99",
	}
	for in, want := range cases {
		got := KopeksToString(in)
		if got != want {
			t.Errorf("KopeksToString(%d) = %s, want %s", in, got, want)
		}
	}
}
