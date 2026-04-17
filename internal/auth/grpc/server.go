package grpc

import (
	"bytes"
	"context"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

type Server struct {
	pb.UnimplementedAuthServiceServer
	authUC    *usecase.AuthUsecase
	profileUC ProfileUsecase
}

type ProfileUsecase interface {
	UploadAvatar(ctx context.Context, userID int64, file io.ReadSeeker, fileSize int64, contentType string) (*domain.User, error)
	UpdateProfile(ctx context.Context, userID int64, username, status, description string) (*domain.User, error)
	ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error
	ChangeEmail(ctx context.Context, userID int64, newEmail, password string) (*domain.User, error)
}

func NewServer(authUC *usecase.AuthUsecase, profileUC ProfileUsecase) *Server {
	return &Server{authUC: authUC, profileUC: profileUC}
}

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	user, err := s.authUC.Register(ctx, req.GetUsername(), req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.RegisterResponse{User: userToProto(user)}, nil
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	session, user, err := s.authUC.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.LoginResponse{
		SessionId: session.ID,
		ExpiresAt: session.ExpiresAt.Unix(),
		User:      userToProto(user),
	}, nil
}

func (s *Server) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if err := s.authUC.Logout(ctx, req.GetSessionId()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.LogoutResponse{}, nil
}

func (s *Server) ValidateSession(ctx context.Context, req *pb.ValidateSessionRequest) (*pb.ValidateSessionResponse, error) {
	user, err := s.authUC.ValidateSession(ctx, req.GetSessionId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.ValidateSessionResponse{User: userToProto(user)}, nil
}

func (s *Server) GetUserByID(_ context.Context, _ *pb.GetUserByIDRequest) (*pb.UserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "use ValidateSession instead")
}

func (s *Server) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UserResponse, error) {
	user, err := s.profileUC.UpdateProfile(ctx, req.GetUserId(), req.GetUsername(), req.GetStatus(), req.GetDescription())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.UserResponse{User: userToProto(user)}, nil
}

func (s *Server) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	if err := s.profileUC.ChangePassword(ctx, req.GetUserId(), req.GetCurrentPassword(), req.GetNewPassword()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.ChangePasswordResponse{}, nil
}

func (s *Server) ChangeEmail(ctx context.Context, req *pb.ChangeEmailRequest) (*pb.UserResponse, error) {
	user, err := s.profileUC.ChangeEmail(ctx, req.GetUserId(), req.GetNewEmail(), req.GetPassword())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.UserResponse{User: userToProto(user)}, nil
}

func (s *Server) UploadAvatar(stream pb.AuthService_UploadAvatarServer) error {
	var buf bytes.Buffer
	var userID int64
	var filename string
	var fileSize int64

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive chunk: %v", err)
		}
		if userID == 0 {
			userID = chunk.GetUserId()
			filename = chunk.GetFilename()
			fileSize = chunk.GetFileSize()
		}
		buf.Write(chunk.GetData())
	}

	if userID == 0 {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	reader := bytes.NewReader(buf.Bytes())
	user, err := s.profileUC.UploadAvatar(stream.Context(), userID, reader, fileSize, filename)
	if err != nil {
		return grpcutil.DomainToGRPCError(err)
	}

	return stream.SendAndClose(&pb.UserResponse{User: userToProto(user)})
}

func (s *Server) GetOAuthURL(ctx context.Context, req *pb.GetOAuthURLRequest) (*pb.GetOAuthURLResponse, error) {
	return nil, status.Error(codes.Unimplemented, "OAuth not implemented yet")
}

func (s *Server) OAuthLogin(ctx context.Context, req *pb.OAuthLoginRequest) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "OAuth not implemented yet")
}

func userToProto(u *domain.User) *pb.UserInfo {
	if u == nil {
		return nil
	}
	return &pb.UserInfo{
		Id:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		AvatarUrl:   u.AvatarURL,
		Status:      u.Status,
		Description: u.Description,
		CreatedAt:   u.CreatedAt.Unix(),
		UpdatedAt:   u.UpdatedAt.Unix(),
	}
}
