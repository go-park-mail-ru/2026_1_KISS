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

const defaultAdminPageSize = 20

type Server struct {
	pb.UnimplementedAuthServiceServer
	authUC    *usecase.AuthUsecase
	profileUC ProfileUsecase
	eventUC   *usecase.EventUsecase
	adminUC   *usecase.AdminUsecase
}

type ProfileUsecase interface {
	UploadAvatar(ctx context.Context, userID int64, file io.ReadSeeker, fileSize int64, contentType string) (*domain.User, error)
	UpdateProfile(ctx context.Context, userID int64, username, status, description string) (*domain.User, error)
	ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error
	ChangeEmail(ctx context.Context, userID int64, newEmail, password string) (*domain.User, error)
}

func NewServer(authUC *usecase.AuthUsecase, profileUC ProfileUsecase, eventUC *usecase.EventUsecase, adminUC *usecase.AdminUsecase) *Server {
	return &Server{authUC: authUC, profileUC: profileUC, eventUC: eventUC, adminUC: adminUC}
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

func (s *Server) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.UserResponse, error) {
	if req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	user, err := s.authUC.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.UserResponse{User: userToProto(user)}, nil
}

func (s *Server) GetUserByIdentifier(ctx context.Context, req *pb.GetUserByIdentifierRequest) (*pb.UserResponse, error) {
	if req.GetIdentifier() == "" {
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}
	user, err := s.authUC.GetUserByIdentifier(ctx, req.GetIdentifier())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.UserResponse{User: userToProto(user)}, nil
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

func (s *Server) TrackEvent(ctx context.Context, req *pb.TrackEventRequest) (*pb.TrackEventResponse, error) {
	if err := s.eventUC.Track(ctx, req.GetUserId(), req.GetEventType(), req.GetMetadataJson()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.TrackEventResponse{}, nil
}

func (s *Server) AdminListUsers(ctx context.Context, req *pb.AdminListUsersRequest) (*pb.AdminListUsersResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultAdminPageSize
	}
	users, total, err := s.adminUC.ListUsers(ctx, req.GetAdminUserId(), limit, int(req.GetOffset()), req.GetSearch())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	pbUsers := make([]*pb.UserInfo, len(users))
	for i := range users {
		pbUsers[i] = userToProto(&users[i])
	}
	totalInt64 := int64(total)
	if totalInt64 > 2147483647 {
		totalInt64 = 2147483647
	}
	//nolint:gosec
	return &pb.AdminListUsersResponse{Users: pbUsers, Total: int32(uint32(totalInt64))}, nil
}

func (s *Server) AdminSetBan(ctx context.Context, req *pb.AdminSetBanRequest) (*pb.AdminSetBanResponse, error) {
	if err := s.adminUC.SetBan(ctx, req.GetAdminUserId(), req.GetTargetUserId(), req.GetBan()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.AdminSetBanResponse{}, nil
}

func (s *Server) AdminGetStats(ctx context.Context, req *pb.AdminGetStatsRequest) (*pb.AdminGetStatsResponse, error) {
	stats, err := s.adminUC.GetStats(ctx, req.GetAdminUserId())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.AdminGetStatsResponse{
		TotalUsers:    stats.TotalUsers,
		TotalSessions: stats.TotalSessions,
		Dau:           stats.DAU,
		Mau:           stats.MAU,
	}, nil
}

func (s *Server) AdminUpdateUser(ctx context.Context, req *pb.AdminUpdateUserRequest) (*pb.UserResponse, error) {
	user, err := s.adminUC.AdminUpdateUser(ctx, req.GetAdminUserId(), req.GetTargetUserId(), req.GetUsername(), req.GetEmail())
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.UserResponse{User: userToProto(user)}, nil
}

func (s *Server) AdminResetPassword(ctx context.Context, req *pb.AdminResetPasswordRequest) (*pb.AdminResetPasswordResponse, error) {
	if err := s.adminUC.AdminResetPassword(ctx, req.GetAdminUserId(), req.GetTargetUserId(), req.GetNewPassword()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.AdminResetPasswordResponse{}, nil
}

func (s *Server) AdminSetPlan(ctx context.Context, req *pb.AdminSetPlanRequest) (*pb.AdminSetPlanResponse, error) {
	if err := s.adminUC.AdminSetPlan(ctx, req.GetAdminUserId(), req.GetTargetUserId(), req.GetPlan()); err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	return &pb.AdminSetPlanResponse{}, nil
}

func (s *Server) AdminGetActivityStats(ctx context.Context, req *pb.AdminGetActivityStatsRequest) (*pb.AdminGetActivityStatsResponse, error) {
	dauDays := int(req.GetDauDays())
	if dauDays <= 0 {
		dauDays = 30
	}
	mauMonths := int(req.GetMauMonths())
	if mauMonths <= 0 {
		mauMonths = 12
	}
	dau, mau, err := s.adminUC.GetActivityStats(ctx, req.GetAdminUserId(), dauDays, mauMonths)
	if err != nil {
		return nil, grpcutil.DomainToGRPCError(err)
	}
	pbDau := make([]*pb.DauEntry, len(dau))
	for i, d := range dau {
		pbDau[i] = &pb.DauEntry{Date: d.Date.Format("2006-01-02"), Count: d.Count}
	}
	pbMau := make([]*pb.MauEntry, len(mau))
	for i, m := range mau {
		pbMau[i] = &pb.MauEntry{Month: m.Month.Format("2006-01"), Count: m.Count}
	}
	return &pb.AdminGetActivityStatsResponse{Dau: pbDau, Mau: pbMau}, nil
}

func userToProto(u *domain.User) *pb.UserInfo {
	if u == nil {
		return nil
	}
	info := &pb.UserInfo{
		Id:               u.ID,
		Username:         u.Username,
		Email:            u.Email,
		AvatarUrl:        u.AvatarURL,
		Status:           u.Status,
		Description:      u.Description,
		CreatedAt:        u.CreatedAt.Unix(),
		UpdatedAt:        u.UpdatedAt.Unix(),
		IsAdmin:          u.IsAdmin,
		Plan:             u.Plan,
		TotalTimeSeconds: u.TotalTimeSeconds,
	}
	if u.LastActiveAt != nil {
		info.LastActiveAt = u.LastActiveAt.Unix()
	}
	return info
}
