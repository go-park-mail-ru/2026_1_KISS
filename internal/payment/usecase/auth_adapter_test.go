package usecase

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

func TestGRPCAuthAdapter_SetUserPlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	adapter := NewGRPCAuthAdapter(client)

	client.EXPECT().
		SetUserPlanInternal(gomock.Any(), &pbauth.SetUserPlanInternalRequest{
			UserId: 42, Plan: "pro", ExpiresAt: 12345,
		}).
		Return(&pbauth.SetUserPlanInternalResponse{}, nil)

	if err := adapter.SetUserPlan(context.Background(), 42, "pro", 12345); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGRPCAuthAdapter_PropagatesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockAuthServiceClient(ctrl)
	adapter := NewGRPCAuthAdapter(client)

	want := errors.New("boom")
	client.EXPECT().SetUserPlanInternal(gomock.Any(), gomock.Any()).Return(nil, want)

	if err := adapter.SetUserPlan(context.Background(), 1, "max", 0); !errors.Is(err, want) {
		t.Errorf("expected boom, got %v", err)
	}
}
