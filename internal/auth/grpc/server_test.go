package grpc

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/mail"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

type testEnv struct {
	client      pb.AuthServiceClient
	userRepo    *mocks.MockUserRepository
	sessionRepo *mocks.MockSessionRepository
	profileUC   *mockProfileUsecase
	conn        *grpc.ClientConn
}

type mockProfileUsecase struct {
	uploadAvatarFn   func(ctx context.Context, userID int64, file io.ReadSeeker, fileSize int64, contentType string) (*domain.User, error)
	updateProfileFn  func(ctx context.Context, userID int64, username, status, description string) (*domain.User, error)
	changePasswordFn func(ctx context.Context, userID int64, currentPassword, newPassword string) error
	changeEmailFn    func(ctx context.Context, userID int64, newEmail, password string) (*domain.User, error)
}

func (m *mockProfileUsecase) UploadAvatar(ctx context.Context, userID int64, file io.ReadSeeker, fileSize int64, contentType string) (*domain.User, error) {
	if m.uploadAvatarFn != nil {
		return m.uploadAvatarFn(ctx, userID, file, fileSize, contentType)
	}
	return nil, nil
}

func (m *mockProfileUsecase) UpdateProfile(ctx context.Context, userID int64, username, st, description string) (*domain.User, error) {
	if m.updateProfileFn != nil {
		return m.updateProfileFn(ctx, userID, username, st, description)
	}
	return nil, nil
}

func (m *mockProfileUsecase) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	if m.changePasswordFn != nil {
		return m.changePasswordFn(ctx, userID, currentPassword, newPassword)
	}
	return nil
}

func (m *mockProfileUsecase) ChangeEmail(ctx context.Context, userID int64, newEmail, password string) (*domain.User, error) {
	if m.changeEmailFn != nil {
		return m.changeEmailFn(ctx, userID, newEmail, password)
	}
	return nil, nil
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	ctrl := gomock.NewController(t)

	userRepo := mocks.NewMockUserRepository(ctrl)
	sessionRepo := mocks.NewMockSessionRepository(ctrl)
	verificationRepo := mocks.NewMockVerificationRepository(ctrl)
	profileUC := &mockProfileUsecase{}

	mailSvc := mail.New("", "", "", "")

	authUC := usecase.New(userRepo, sessionRepo, verificationRepo, mailSvc, 24*time.Hour)
	eventRepo := mocks.NewMockEventRepository(ctrl)
	eventUC := usecase.NewEventUsecase(eventRepo)
	adminUC := usecase.NewAdminUsecase(userRepo, eventRepo)
	srv := NewServer(authUC, profileUC, eventUC, adminUC)

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcServer, srv)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("grpc serve error: %v", err)
		}
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		lis.Close()
	})

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return &testEnv{
		client:      pb.NewAuthServiceClient(conn),
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		profileUC:   profileUC,
		conn:        conn,
	}
}

func TestRegister_Success(t *testing.T) {
	env := setup(t)

	env.userRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, user *domain.User) (int64, error) {
			user.ID = 1
			user.CreatedAt = time.Now()
			user.UpdatedAt = time.Now()
			return 1, nil
		},
	)

	resp, err := env.client.Register(context.Background(), &pb.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if resp.GetUser().GetId() != 1 {
		t.Errorf("want user id 1, got %d", resp.GetUser().GetId())
	}
	if resp.GetUser().GetUsername() != "testuser" {
		t.Errorf("want username testuser, got %s", resp.GetUser().GetUsername())
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	env := setup(t)

	_, err := env.client.Register(context.Background(), &pb.RegisterRequest{
		Username: "testuser",
		Email:    "bad-email",
		Password: "Password123!",
	})
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", st.Code())
	}
}

func TestLogin_Success(t *testing.T) {
	env := setup(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	env.userRepo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(&domain.User{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hash),
		IsVerified:   true,
	}, nil)
	env.sessionRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	resp, err := env.client.Login(context.Background(), &pb.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if resp.GetSessionId() == "" {
		t.Error("expected non-empty session id")
	}
	if resp.GetUser().GetId() != 1 {
		t.Errorf("want user id 1, got %d", resp.GetUser().GetId())
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	env := setup(t)

	env.userRepo.EXPECT().GetByEmail(gomock.Any(), "test@example.com").Return(nil, domain.ErrNotFound)

	_, err := env.client.Login(context.Background(), &pb.LoginRequest{
		Email:    "test@example.com",
		Password: "wrong",
	})
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", st.Code())
	}
}

func TestLogout_Success(t *testing.T) {
	env := setup(t)

	env.sessionRepo.EXPECT().DeleteByID(gomock.Any(), "session-123").Return(nil)

	_, err := env.client.Logout(context.Background(), &pb.LogoutRequest{
		SessionId: "session-123",
	})
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
}

func TestValidateSession_Success(t *testing.T) {
	env := setup(t)

	env.sessionRepo.EXPECT().GetByID(gomock.Any(), "session-123").Return(&domain.Session{
		ID:        "session-123",
		UserID:    1,
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	env.userRepo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(&domain.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
	}, nil)

	resp, err := env.client.ValidateSession(context.Background(), &pb.ValidateSessionRequest{
		SessionId: "session-123",
	})
	if err != nil {
		t.Fatalf("validate session: %v", err)
	}
	if resp.GetUser().GetId() != 1 {
		t.Errorf("want user id 1, got %d", resp.GetUser().GetId())
	}
}

func TestValidateSession_Expired(t *testing.T) {
	env := setup(t)

	env.sessionRepo.EXPECT().GetByID(gomock.Any(), "expired").Return(nil, domain.ErrSessionExpired)

	_, err := env.client.ValidateSession(context.Background(), &pb.ValidateSessionRequest{
		SessionId: "expired",
	})
	if err == nil {
		t.Fatal("expected error for expired session")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", st.Code())
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	env := setup(t)

	env.profileUC.updateProfileFn = func(_ context.Context, userID int64, username, st, desc string) (*domain.User, error) {
		return &domain.User{
			ID:          userID,
			Username:    username,
			Status:      st,
			Description: desc,
		}, nil
	}

	resp, err := env.client.UpdateProfile(context.Background(), &pb.UpdateProfileRequest{
		UserId:      1,
		Username:    "newname",
		Status:      "active",
		Description: "hello",
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if resp.GetUser().GetUsername() != "newname" {
		t.Errorf("want newname, got %s", resp.GetUser().GetUsername())
	}
}

func TestChangePassword_Success(t *testing.T) {
	env := setup(t)

	env.profileUC.changePasswordFn = func(_ context.Context, _ int64, _, _ string) error {
		return nil
	}

	_, err := env.client.ChangePassword(context.Background(), &pb.ChangePasswordRequest{
		UserId:          1,
		CurrentPassword: "old",
		NewPassword:     "new",
	})
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
}

func TestChangeEmail_Success(t *testing.T) {
	env := setup(t)

	env.profileUC.changeEmailFn = func(_ context.Context, userID int64, email, _ string) (*domain.User, error) {
		return &domain.User{ID: userID, Email: email}, nil
	}

	resp, err := env.client.ChangeEmail(context.Background(), &pb.ChangeEmailRequest{
		UserId:   1,
		NewEmail: "new@example.com",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("change email: %v", err)
	}
	if resp.GetUser().GetEmail() != "new@example.com" {
		t.Errorf("want new@example.com, got %s", resp.GetUser().GetEmail())
	}
}

func TestGetOAuthURL_Unimplemented(t *testing.T) {
	env := setup(t)

	_, err := env.client.GetOAuthURL(context.Background(), &pb.GetOAuthURLRequest{
		Provider: "yandex",
	})
	if err == nil {
		t.Fatal("expected error for unimplemented")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Unimplemented {
		t.Errorf("want Unimplemented, got %v", st.Code())
	}
}
