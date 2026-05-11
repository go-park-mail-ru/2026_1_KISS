package handler

import (
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/mailru/easyjson"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/payment"
)

type PaymentHandler struct {
	client       pb.PaymentServiceClient
	webhookCIDRs []*net.IPNet
}

func NewPaymentHandler(client pb.PaymentServiceClient, webhookCIDRs []string) *PaymentHandler {
	nets := make([]*net.IPNet, 0, len(webhookCIDRs))
	for _, cidr := range webhookCIDRs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, n, err := net.ParseCIDR(cidr)
		if err == nil {
			nets = append(nets, n)
		}
	}
	return &PaymentHandler{client: client, webhookCIDRs: nets}
}

func (h *PaymentHandler) RegisterRoutes(mux *http.ServeMux, authMw middleware.Middleware) {
	mux.Handle("POST /api/v1/payments/subscription", authMw(http.HandlerFunc(h.CreateSubscription)))
	mux.Handle("GET /api/v1/payments/{id}/status", authMw(http.HandlerFunc(h.GetStatus)))
	mux.Handle("GET /api/v1/subscription/plans", authMw(http.HandlerFunc(h.ListPlans)))
	mux.Handle("GET /api/v1/subscription/me", authMw(http.HandlerFunc(h.GetMine)))
	mux.HandleFunc("POST /api/v1/payments/webhook", h.Webhook)
}

type createSubscriptionRequest struct {
	Plan      string `json:"plan"`
	ReturnURL string `json:"return_url,omitempty"`
}

type createSubscriptionResponse struct {
	PaymentID         string `json:"payment_id"`
	ConfirmationToken string `json:"confirmation_token"`
	AmountKopeks      int64  `json:"amount_kopeks"`
	Plan              string `json:"plan"`
}

type paymentStatusResponse struct {
	PaymentID    string `json:"payment_id"`
	Status       string `json:"status"`
	AmountKopeks int64  `json:"amount_kopeks"`
	CreatedAt    int64  `json:"created_at"`
	PaidAt       int64  `json:"paid_at"`
}

type planResponse struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	PriceKopeks    int64  `json:"price_kopeks"`
	ExecutionQuota int32  `json:"execution_quota"`
	DurationDays   int32  `json:"duration_days"`
}

type listPlansResponse struct {
	Plans []planResponse `json:"plans"`
}

type subscriptionMeResponse struct {
	HasActive bool   `json:"has_active"`
	Plan      string `json:"plan"`
	StartedAt int64  `json:"started_at,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

type webhookObject struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (h *PaymentHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createSubscriptionRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Plan == "" {
		httputil.Error(w, http.StatusBadRequest, "plan is required")
		return
	}

	resp, err := h.client.CreateSubscriptionPayment(r.Context(), &pb.CreateSubscriptionPaymentRequest{
		UserId:    user.ID,
		UserEmail: user.Email,
		PlanName:  req.Plan,
		ReturnUrl: req.ReturnURL,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	httputil.JSON(w, http.StatusCreated, createSubscriptionResponse{
		PaymentID:         resp.GetId(),
		ConfirmationToken: resp.GetConfirmationToken(),
		AmountKopeks:      resp.GetAmountKopeks(),
		Plan:              resp.GetPlanName(),
	})
}

func (h *PaymentHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	paymentID := r.PathValue("id")
	resp, err := h.client.GetPaymentStatus(r.Context(), &pb.GetPaymentStatusRequest{
		PaymentId: paymentID,
		UserId:    user.ID,
	})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}

	p := resp.GetPayment()
	httputil.JSON(w, http.StatusOK, paymentStatusResponse{
		PaymentID:    p.GetId(),
		Status:       p.GetStatus(),
		AmountKopeks: p.GetAmountKopeks(),
		CreatedAt:    p.GetCreatedAt(),
		PaidAt:       p.GetPaidAt(),
	})
}

func (h *PaymentHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	resp, err := h.client.ListPlans(r.Context(), &pb.ListPlansRequest{})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	plans := make([]planResponse, 0, len(resp.GetPlans()))
	for _, p := range resp.GetPlans() {
		plans = append(plans, planResponse{
			ID:             p.GetId(),
			Name:           p.GetName(),
			PriceKopeks:    p.GetPriceKopeks(),
			ExecutionQuota: p.GetExecutionQuota(),
			DurationDays:   p.GetDurationDays(),
		})
	}
	httputil.JSON(w, http.StatusOK, listPlansResponse{Plans: plans})
}

func (h *PaymentHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	resp, err := h.client.GetMySubscription(r.Context(), &pb.GetMySubscriptionRequest{UserId: user.ID})
	if err != nil {
		httputil.MapDomainError(w, grpcutil.GRPCToDomainError(err))
		return
	}
	if !resp.GetHasActive() {
		httputil.JSON(w, http.StatusOK, subscriptionMeResponse{HasActive: false, Plan: user.Plan})
		return
	}
	sub := resp.GetSubscription()
	httputil.JSON(w, http.StatusOK, subscriptionMeResponse{
		HasActive: true,
		Plan:      sub.GetPlanName(),
		StartedAt: sub.GetStartedAt(),
		ExpiresAt: sub.GetExpiresAt(),
	})
}

type webhookPayload struct {
	Type   string        `json:"type"`
	Event  string        `json:"event"`
	Object webhookObject `json:"object"`
}

func (h *PaymentHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	clientIP := extractClientIP(r)
	if !h.allowedWebhookIP(clientIP) {
		logger.Warn(r.Context(), "payment.webhook.rejected", "remote_ip", clientIP)
		httputil.Error(w, http.StatusForbidden, "forbidden")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "read body")
		return
	}

	var payload webhookPayload
	if err := easyjson.Unmarshal(body, &payload); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if payload.Object.ID == "" || payload.Object.Status == "" {
		httputil.Error(w, http.StatusBadRequest, "missing fields")
		return
	}

	_, err = h.client.HandleWebhook(r.Context(), &pb.WebhookEvent{
		Event:             payload.Event,
		YookassaPaymentId: payload.Object.ID,
		Status:            payload.Object.Status,
		SourceIp:          clientIP,
		RawBody:           body,
	})
	if err != nil {
		logger.Error(r.Context(), "payment.webhook.error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *PaymentHandler) allowedWebhookIP(ip string) bool {
	if len(h.webhookCIDRs) == 0 {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range h.webhookCIDRs {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

func extractClientIP(r *http.Request) string {
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		parts := strings.Split(v, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
