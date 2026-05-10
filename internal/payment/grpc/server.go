package grpc

import (
	"context"

	paymentdomain "github.com/go-park-mail-ru/2026_1_KISS/internal/payment"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/payment/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/payment"
)

type Server struct {
	pb.UnimplementedPaymentServiceServer
	uc *usecase.PaymentUsecase
}

func NewServer(uc *usecase.PaymentUsecase) *Server {
	return &Server{uc: uc}
}

func (s *Server) CreateSubscriptionPayment(ctx context.Context, req *pb.CreateSubscriptionPaymentRequest) (*pb.CreateSubscriptionPaymentResponse, error) {
	out, err := s.uc.CreateSubscriptionPayment(ctx, usecase.CreateInput{
		UserID:    req.GetUserId(),
		UserEmail: req.GetUserEmail(),
		PlanName:  req.GetPlanName(),
		ReturnURL: req.GetReturnUrl(),
	})
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.CreateSubscriptionPaymentResponse{
		Id:                out.PaymentID,
		ConfirmationToken: out.ConfirmationToken,
		AmountKopeks:      out.AmountKopeks,
		PlanName:          out.PlanName,
	}, nil
}

func (s *Server) GetPaymentStatus(ctx context.Context, req *pb.GetPaymentStatusRequest) (*pb.GetPaymentStatusResponse, error) {
	p, err := s.uc.GetStatus(ctx, req.GetPaymentId(), req.GetUserId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.GetPaymentStatusResponse{Payment: paymentToProto(p)}, nil
}

func (s *Server) ListPlans(ctx context.Context, _ *pb.ListPlansRequest) (*pb.ListPlansResponse, error) {
	plans, err := s.uc.ListPlans(ctx)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	out := make([]*pb.Plan, len(plans))
	for i := range plans {
		out[i] = planToProto(&plans[i])
	}
	return &pb.ListPlansResponse{Plans: out}, nil
}

func (s *Server) GetMySubscription(ctx context.Context, req *pb.GetMySubscriptionRequest) (*pb.GetMySubscriptionResponse, error) {
	sub, err := s.uc.GetMySubscription(ctx, req.GetUserId())
	if err != nil {
		return &pb.GetMySubscriptionResponse{HasActive: false}, nil
	}
	return &pb.GetMySubscriptionResponse{HasActive: true, Subscription: subscriptionToProto(sub)}, nil
}

func (s *Server) HandleWebhook(ctx context.Context, req *pb.WebhookEvent) (*pb.WebhookAck, error) {
	if err := s.uc.HandleWebhook(ctx, usecase.WebhookInput{
		YooKassaPaymentID: req.GetYookassaPaymentId(),
		Status:            req.GetStatus(),
	}); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.WebhookAck{Processed: true}, nil
}

func paymentToProto(p *paymentdomain.Payment) *pb.Payment {
	if p == nil {
		return nil
	}
	pb := &pb.Payment{
		Id:                p.ID,
		UserId:            p.UserID,
		YookassaPaymentId: p.YooKassaPaymentID,
		Status:            p.Status,
		AmountKopeks:      p.AmountKopeks,
		ConfirmationToken: p.ConfirmationToken,
		CreatedAt:         p.CreatedAt.Unix(),
	}
	if p.PaidAt != nil {
		pb.PaidAt = p.PaidAt.Unix()
	}
	return pb
}

func planToProto(p *paymentdomain.Plan) *pb.Plan {
	return &pb.Plan{
		Id:             p.ID,
		Name:           p.Name,
		PriceKopeks:    p.PriceKopeks,
		ExecutionQuota: p.ExecutionQuota,
		DurationDays:   p.DurationDays,
	}
}

func subscriptionToProto(s *paymentdomain.Subscription) *pb.Subscription {
	return &pb.Subscription{
		Id:        s.ID,
		UserId:    s.UserID,
		PlanName:  s.PlanName,
		StartedAt: s.StartedAt.Unix(),
		ExpiresAt: s.ExpiresAt.Unix(),
		Active:    true,
	}
}
