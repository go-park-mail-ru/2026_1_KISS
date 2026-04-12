package usecase

//go:generate mockgen -source=profile.go -destination=../../mocks/profile_repo_mock.go -package=mocks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/filestorage"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

type userRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateAvatarURL(ctx context.Context, userID int64, avatarURL string) error
	UpdateProfile(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
	UpdateEmail(ctx context.Context, userID int64, email string) error
}

// ProfileUsecase handles profile-related business logic.
type ProfileUsecase struct {
	userRepo    userRepository
	fileStorage filestorage.FileStorage
	maxFileSize int64
}

// New creates a new ProfileUsecase.
func New(userRepo userRepository, fs filestorage.FileStorage, maxFileSize int64) *ProfileUsecase {
	return &ProfileUsecase{
		userRepo:    userRepo,
		fileStorage: fs,
		maxFileSize: maxFileSize,
	}
}

var allowedMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/bmp":  ".bmp",
}

func (uc *ProfileUsecase) UploadAvatar(ctx context.Context, userID int64, file io.ReadSeeker, fileSize int64, _ string) (*domain.User, error) {
	logger.Info(ctx, "usecase.profile.UploadAvatar", "user_id", userID, "file_size", fileSize)

	if fileSize > uc.maxFileSize {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", "file too large")
		return nil, fmt.Errorf("%w: file too large", domain.ErrInvalidInput)
	}

	sniffBuf := make([]byte, 512)
	n, err := io.ReadFull(file, sniffBuf)
	if err != nil && err != io.ErrUnexpectedEOF {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", err)
		return nil, fmt.Errorf("read file header: %w", err)
	}
	sniffBuf = sniffBuf[:n]

	detected := http.DetectContentType(sniffBuf)
	mime := strings.Split(detected, ";")[0]

	logger.Info(ctx, "usecase.profile.UploadAvatar", "content_type", mime)

	ext, ok := allowedMIME[mime]
	if !ok {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", "invalid file type", "content_type", mime)
		return nil, fmt.Errorf("%w: invalid file type", domain.ErrInvalidInput)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", err)
		return nil, fmt.Errorf("seek file: %w", err)
	}

	filename := uuid.New().String() + ext

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", err)
		return nil, err
	}
	oldAvatar := user.AvatarURL

	url, err := uc.fileStorage.Save(filename, file)
	if err != nil {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", err)
		return nil, fmt.Errorf("save avatar: %w", err)
	}

	if err := uc.userRepo.UpdateAvatarURL(ctx, userID, url); err != nil {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", err)
		_ = uc.fileStorage.Delete(url)
		return nil, err
	}

	if oldAvatar != "" {
		_ = uc.fileStorage.Delete(oldAvatar)
	}

	logger.Info(ctx, "usecase.profile.UploadAvatar", "user_id", userID, "status", "ok")
	return uc.userRepo.GetByID(ctx, userID)
}

func (uc *ProfileUsecase) UpdateProfile(ctx context.Context, userID int64, username, status, description string) (*domain.User, error) {
	logger.Info(ctx, "usecase.profile.UpdateProfile", "user_id", userID, "username", username)

	if err := httputil.ValidateUsername(username); err != nil {
		logger.Error(ctx, "usecase.profile.UpdateProfile", "error", err)
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error())
	}
	if len(status) > 100 {
		logger.Error(ctx, "usecase.profile.UpdateProfile", "error", "status too long")
		return nil, fmt.Errorf("%w: status must not exceed 100 characters", domain.ErrInvalidInput)
	}
	if len(description) > 500 {
		logger.Error(ctx, "usecase.profile.UpdateProfile", "error", "description too long")
		return nil, fmt.Errorf("%w: description must not exceed 500 characters", domain.ErrInvalidInput)
	}

	user := &domain.User{
		ID:          userID,
		Username:    username,
		Status:      status,
		Description: description,
	}
	if err := uc.userRepo.UpdateProfile(ctx, user); err != nil {
		logger.Error(ctx, "usecase.profile.UpdateProfile", "error", err)
		return nil, err
	}
	logger.Info(ctx, "usecase.profile.UpdateProfile", "user_id", userID, "status", "ok")
	return uc.userRepo.GetByID(ctx, userID)
}

func (uc *ProfileUsecase) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	logger.Info(ctx, "usecase.profile.ChangePassword", "user_id", userID)

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "usecase.profile.ChangePassword", "error", err)
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		logger.Error(ctx, "usecase.profile.ChangePassword", "error", domain.ErrUnauthorized)
		return domain.ErrUnauthorized
	}

	if err := httputil.ValidatePassword(newPassword); err != nil {
		logger.Error(ctx, "usecase.profile.ChangePassword", "error", err)
		return fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error())
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error(ctx, "usecase.profile.ChangePassword", "error", err)
		return fmt.Errorf("hash password: %w", err)
	}

	if err := uc.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		logger.Error(ctx, "usecase.profile.ChangePassword", "error", err)
		return err
	}
	logger.Info(ctx, "usecase.profile.ChangePassword", "user_id", userID, "status", "ok")
	return nil
}

func (uc *ProfileUsecase) ChangeEmail(ctx context.Context, userID int64, newEmail, password string) (*domain.User, error) {
	logger.Info(ctx, "usecase.profile.ChangeEmail", "user_id", userID, "new_email", newEmail)

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "usecase.profile.ChangeEmail", "error", err)
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		logger.Error(ctx, "usecase.profile.ChangeEmail", "error", domain.ErrUnauthorized)
		return nil, domain.ErrUnauthorized
	}

	if err := httputil.ValidateEmail(newEmail); err != nil {
		logger.Error(ctx, "usecase.profile.ChangeEmail", "error", err)
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error())
	}

	if err := uc.userRepo.UpdateEmail(ctx, userID, newEmail); err != nil {
		logger.Error(ctx, "usecase.profile.ChangeEmail", "error", err)
		return nil, err
	}

	logger.Info(ctx, "usecase.profile.ChangeEmail", "user_id", userID, "status", "ok")
	return uc.userRepo.GetByID(ctx, userID)
}
