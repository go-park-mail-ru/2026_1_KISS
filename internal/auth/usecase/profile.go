package usecase

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/muesli/smartcrop"
	"github.com/muesli/smartcrop/nfnt"
	"golang.org/x/crypto/bcrypt"
	_ "golang.org/x/image/bmp"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/httputil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/sanitize"
)

//go:generate mockgen -destination=../../../internal/mocks/file_uploader_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase FileUploader

type FileUploader interface {
	Upload(ctx context.Context, ownerID int64, category, filename string, data io.Reader, size int64) (string, error)
	Delete(ctx context.Context, url string) error
}

type ProfileUsecase struct {
	userRepo    repository.UserRepository
	uploader    FileUploader
	maxFileSize int64
}

func NewProfileUsecase(userRepo repository.UserRepository, uploader FileUploader, maxFileSize int64) *ProfileUsecase {
	return &ProfileUsecase{
		userRepo:    userRepo,
		uploader:    uploader,
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

	var saveReader io.Reader = file

	img, _, decErr := image.Decode(file)
	if decErr != nil {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", decErr)
		return nil, fmt.Errorf("decode image: %w", decErr)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek file after decode: %w", err)
	}

	bounds := img.Bounds()
	dx, dy := bounds.Dx(), bounds.Dy()

	if dx != dy {
		side := dx
		if dy < side {
			side = dy
		}

		analyzer := smartcrop.NewAnalyzer(nfnt.NewDefaultResizer())
		bestCrop, cropErr := analyzer.FindBestCrop(img, side, side)
		if cropErr != nil {
			logger.Error(ctx, "usecase.profile.UploadAvatar", "error", cropErr)
			return nil, fmt.Errorf("smartcrop: %w", cropErr)
		}

		cropped := image.NewRGBA(image.Rect(0, 0, side, side))
		draw.Draw(cropped, cropped.Bounds(), img, image.Pt(bestCrop.Min.X, bestCrop.Min.Y), draw.Src)

		var buf bytes.Buffer
		if mime == "image/png" {
			if err := png.Encode(&buf, cropped); err != nil {
				return nil, fmt.Errorf("encode png: %w", err)
			}
		} else {
			if err := jpeg.Encode(&buf, cropped, &jpeg.Options{Quality: 90}); err != nil {
				return nil, fmt.Errorf("encode jpeg: %w", err)
			}
			ext = ".jpg"
		}

		saveReader = bytes.NewReader(buf.Bytes())
		logger.Info(ctx, "usecase.profile.UploadAvatar", "cropped", true, "side", side)
	}

	filename := uuid.New().String() + ext

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", err)
		return nil, err
	}
	oldAvatar := user.AvatarURL

	var buf bytes.Buffer
	size, err := io.Copy(&buf, saveReader)
	if err != nil {
		return nil, fmt.Errorf("buffer avatar: %w", err)
	}

	url, err := uc.uploader.Upload(ctx, userID, "avatars", filename, bytes.NewReader(buf.Bytes()), size)
	if err != nil {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", err)
		return nil, fmt.Errorf("save avatar: %w", err)
	}

	if err := uc.userRepo.UpdateAvatarURL(ctx, userID, url); err != nil {
		logger.Error(ctx, "usecase.profile.UploadAvatar", "error", err)
		_ = uc.uploader.Delete(ctx, url)
		return nil, err
	}

	if oldAvatar != "" {
		_ = uc.uploader.Delete(ctx, oldAvatar)
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
	if utf8.RuneCountInString(status) > 100 {
		logger.Error(ctx, "usecase.profile.UpdateProfile", "error", "status too long")
		return nil, fmt.Errorf("%w: status must not exceed 100 characters", domain.ErrInvalidInput)
	}
	if utf8.RuneCountInString(description) > 500 {
		logger.Error(ctx, "usecase.profile.UpdateProfile", "error", "description too long")
		return nil, fmt.Errorf("%w: description must not exceed 500 characters", domain.ErrInvalidInput)
	}

	user := &domain.User{
		ID:          userID,
		Username:    sanitize.EscapeHTML(username),
		Status:      sanitize.EscapeHTML(status),
		Description: sanitize.EscapeHTML(description),
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
