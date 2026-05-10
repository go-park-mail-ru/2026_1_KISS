package usecase

import (
	"context"

	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

type AuthClient interface {
	SetUserPlan(ctx context.Context, userID int64, plan string, expiresAt int64) error
}

type GRPCAuthAdapter struct {
	client pbauth.AuthServiceClient
}

func NewGRPCAuthAdapter(client pbauth.AuthServiceClient) *GRPCAuthAdapter {
	return &GRPCAuthAdapter{client: client}
}

func (a *GRPCAuthAdapter) SetUserPlan(ctx context.Context, userID int64, plan string, expiresAt int64) error {
	_, err := a.client.SetUserPlanInternal(ctx, &pbauth.SetUserPlanInternalRequest{
		UserId:    userID,
		Plan:      plan,
		ExpiresAt: expiresAt,
	})
	return err
}
