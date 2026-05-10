package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractClientIP(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(*http.Request)
		remoteAddr string
		want       string
	}{
		{
			name:       "X-Real-IP wins",
			setup:      func(r *http.Request) { r.Header.Set("X-Real-IP", "1.2.3.4") },
			remoteAddr: "10.0.0.1:12345",
			want:       "1.2.3.4",
		},
		{
			name:       "X-Forwarded-For first",
			setup:      func(r *http.Request) { r.Header.Set("X-Forwarded-For", "5.6.7.8, 9.9.9.9") },
			remoteAddr: "10.0.0.1:12345",
			want:       "5.6.7.8",
		},
		{
			name:       "fallback to RemoteAddr host",
			setup:      func(_ *http.Request) {},
			remoteAddr: "10.0.0.1:12345",
			want:       "10.0.0.1",
		},
		{
			name:       "fallback raw on parse error",
			setup:      func(_ *http.Request) {},
			remoteAddr: "raw-address",
			want:       "raw-address",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/x", nil)
			r.RemoteAddr = tc.remoteAddr
			tc.setup(r)
			got := extractClientIP(r)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAllowedWebhookIP_NoCIDRsAllowsAll(t *testing.T) {
	h := NewPaymentHandler(nil, nil)
	if !h.allowedWebhookIP("1.2.3.4") {
		t.Error("empty whitelist should allow any IP")
	}
}

func TestAllowedWebhookIP_Whitelist(t *testing.T) {
	h := NewPaymentHandler(nil, []string{"185.71.76.0/27", "77.75.156.11/32"})

	if !h.allowedWebhookIP("185.71.76.10") {
		t.Error("185.71.76.10 should be in 185.71.76.0/27")
	}
	if !h.allowedWebhookIP("77.75.156.11") {
		t.Error("77.75.156.11 should match /32")
	}
	if h.allowedWebhookIP("8.8.8.8") {
		t.Error("8.8.8.8 should be rejected")
	}
	if h.allowedWebhookIP("not-an-ip") {
		t.Error("invalid IP should be rejected")
	}
}

func TestAllowedWebhookIP_BadCIDRSkipped(t *testing.T) {
	h := NewPaymentHandler(nil, []string{"garbage", "185.71.76.0/27"})
	if !h.allowedWebhookIP("185.71.76.5") {
		t.Error("valid CIDR after garbage should still work")
	}
}

func TestWebhook_RejectsForeignIP(t *testing.T) {
	h := NewPaymentHandler(nil, []string{"185.71.76.0/27"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook", nil)
	r.RemoteAddr = "8.8.8.8:1"
	w := httptest.NewRecorder()
	h.Webhook(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestWebhook_BadJSON(t *testing.T) {
	h := NewPaymentHandler(nil, nil)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook",
		newReader(`{not json`))
	w := httptest.NewRecorder()
	h.Webhook(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWebhook_MissingFields(t *testing.T) {
	h := NewPaymentHandler(nil, nil)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook",
		newReader(`{"event":"payment.succeeded","object":{"id":"","status":""}}`))
	w := httptest.NewRecorder()
	h.Webhook(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func newReader(s string) *stringReader { return &stringReader{s: s} }

type stringReader struct {
	s string
	i int
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.i >= len(r.s) {
		return 0, http.ErrBodyReadAfterClose
	}
	n := copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

func (r *stringReader) Close() error { return nil }
