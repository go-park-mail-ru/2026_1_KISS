package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"testing"

	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
)

func testUser() *domain.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	return &domain.User{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hash),
		AvatarURL:    "/uploads/old.jpg",
	}
}

func makeTestImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 128, A: 255})
		}
	}
	return img
}

func encodeJPEG(img image.Image) []byte {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestUploadAvatar(t *testing.T) {
	jpegData := encodeJPEG(makeTestImage(4, 2))
	pngData := encodePNG(makeTestImage(4, 2))
	pngSquareData := encodePNG(makeTestImage(4, 4))

	tests := []struct {
		name      string
		fileData  []byte
		fileSize  int64
		wantErr   bool
		errTarget error
	}{
		{
			name:     "success jpeg non-square",
			fileData: jpegData,
			fileSize: int64(len(jpegData)),
		},
		{
			name:     "success png non-square",
			fileData: pngData,
			fileSize: int64(len(pngData)),
		},
		{
			name:     "success png square passthrough",
			fileData: pngSquareData,
			fileSize: int64(len(pngSquareData)),
		},
		{
			name:      "file too large",
			fileData:  jpegData,
			fileSize:  3 << 20,
			wantErr:   true,
			errTarget: domain.ErrInvalidInput,
		},
		{
			name:      "invalid mime type",
			fileData:  []byte("this is plain text content that is not an image"),
			fileSize:  47,
			wantErr:   true,
			errTarget: domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := testUser()
			repo := mocks.NewMockUserRepository(ctrl)
			fs := mocks.NewMockFileUploader(ctrl)

			needsRepo := !tc.wantErr || (!errors.Is(tc.errTarget, domain.ErrInvalidInput))
			if needsRepo {
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
				fs.EXPECT().Upload(gomock.Any(), int64(1), "avatars", gomock.Any(), gomock.Any(), gomock.Any()).Return("/uploads/avatars/new.jpg", nil)
				repo.EXPECT().UpdateAvatarURL(gomock.Any(), int64(1), "/uploads/avatars/new.jpg").Return(nil)
				fs.EXPECT().Delete(gomock.Any(), user.AvatarURL).Return(nil)
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			}

			uc := usecase.NewProfileUsecase(repo, fs, 2<<20)

			_, err := uc.UploadAvatar(t.Context(), 1, bytes.NewReader(tc.fileData), tc.fileSize, "")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errTarget != nil && !errors.Is(err, tc.errTarget) {
					t.Errorf("expected %v, got %v", tc.errTarget, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUploadAvatar_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := testUser()
	repo := mocks.NewMockUserRepository(ctrl)
	fs := mocks.NewMockFileUploader(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	fs.EXPECT().Upload(gomock.Any(), int64(1), "avatars", gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New("disk full"))

	uc := usecase.NewProfileUsecase(repo, fs, 2<<20)

	data := encodePNG(makeTestImage(4, 4))
	_, err := uc.UploadAvatar(t.Context(), 1, bytes.NewReader(data), int64(len(data)), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUploadAvatar_DeletesOldAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := testUser()
	user.AvatarURL = "/uploads/old-avatar.jpg"

	repo := mocks.NewMockUserRepository(ctrl)
	fs := mocks.NewMockFileUploader(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	fs.EXPECT().Upload(gomock.Any(), int64(1), "avatars", gomock.Any(), gomock.Any(), gomock.Any()).Return("/uploads/avatars/new.jpg", nil)
	repo.EXPECT().UpdateAvatarURL(gomock.Any(), int64(1), "/uploads/avatars/new.jpg").Return(nil)
	fs.EXPECT().Delete(gomock.Any(), "/uploads/old-avatar.jpg").Return(nil)
	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)

	uc := usecase.NewProfileUsecase(repo, fs, 2<<20)

	data := encodeJPEG(makeTestImage(4, 4))
	_, err := uc.UploadAvatar(t.Context(), 1, bytes.NewReader(data), int64(len(data)), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadAvatar_UpdateAvatarURLError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := testUser()
	repo := mocks.NewMockUserRepository(ctrl)
	fs := mocks.NewMockFileUploader(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	fs.EXPECT().Upload(gomock.Any(), int64(1), "avatars", gomock.Any(), gomock.Any(), gomock.Any()).Return("/uploads/avatars/new.jpg", nil)
	repo.EXPECT().UpdateAvatarURL(gomock.Any(), int64(1), "/uploads/avatars/new.jpg").Return(errors.New("db error"))
	fs.EXPECT().Delete(gomock.Any(), "/uploads/avatars/new.jpg").Return(nil)

	uc := usecase.NewProfileUsecase(repo, fs, 2<<20)

	data := encodeJPEG(makeTestImage(4, 4))
	_, err := uc.UploadAvatar(t.Context(), 1, bytes.NewReader(data), int64(len(data)), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUploadAvatar_GetByIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockUserRepository(ctrl)
	fs := mocks.NewMockFileUploader(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nil, domain.ErrNotFound)

	uc := usecase.NewProfileUsecase(repo, fs, 2<<20)

	data := encodeJPEG(makeTestImage(4, 4))
	_, err := uc.UploadAvatar(t.Context(), 1, bytes.NewReader(data), int64(len(data)), "")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUploadAvatar_NoOldAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := testUser()
	user.AvatarURL = ""

	repo := mocks.NewMockUserRepository(ctrl)
	fs := mocks.NewMockFileUploader(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	fs.EXPECT().Upload(gomock.Any(), int64(1), "avatars", gomock.Any(), gomock.Any(), gomock.Any()).Return("/uploads/avatars/new.jpg", nil)
	repo.EXPECT().UpdateAvatarURL(gomock.Any(), int64(1), "/uploads/avatars/new.jpg").Return(nil)
	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)

	uc := usecase.NewProfileUsecase(repo, fs, 2<<20)

	data := encodeJPEG(makeTestImage(4, 4))
	_, err := uc.UploadAvatar(t.Context(), 1, bytes.NewReader(data), int64(len(data)), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadAvatar_SquareImageDataIntegrity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := testUser()
	repo := mocks.NewMockUserRepository(ctrl)
	fs := mocks.NewMockFileUploader(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	fs.EXPECT().Upload(gomock.Any(), int64(1), "avatars", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, _, _ string, data io.Reader, _ int64) (string, error) {
			content, err := io.ReadAll(data)
			if err != nil {
				t.Fatalf("failed to read data: %v", err)
			}
			if len(content) == 0 {
				t.Fatal("Upload received empty reader for square image")
			}
			if _, _, err := image.Decode(bytes.NewReader(content)); err != nil {
				t.Fatalf("Upload received invalid image data: %v", err)
			}
			return "/uploads/avatars/new.png", nil
		})
	repo.EXPECT().UpdateAvatarURL(gomock.Any(), int64(1), "/uploads/avatars/new.png").Return(nil)
	fs.EXPECT().Delete(gomock.Any(), user.AvatarURL).Return(nil)
	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)

	uc := usecase.NewProfileUsecase(repo, fs, 2<<20)

	data := encodePNG(makeTestImage(4, 4))
	_, err := uc.UploadAvatar(t.Context(), 1, bytes.NewReader(data), int64(len(data)), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateProfile(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		status      string
		description string
		wantErr     bool
		errTarget   error
	}{
		{
			name:        "success",
			username:    "newuser",
			status:      "Hello",
			description: "Bio",
		},
		{
			name:      "invalid username - too short",
			username:  "ab",
			wantErr:   true,
			errTarget: domain.ErrInvalidInput,
		},
		{
			name:      "status too long",
			username:  "validuser",
			status:    string(make([]byte, 101)),
			wantErr:   true,
			errTarget: domain.ErrInvalidInput,
		},
		{
			name:        "description too long",
			username:    "validuser",
			description: string(make([]byte, 501)),
			wantErr:     true,
			errTarget:   domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := testUser()
			repo := mocks.NewMockUserRepository(ctrl)
			fs := mocks.NewMockFileUploader(ctrl)

			if !tc.wantErr {
				repo.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(nil)
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			}

			uc := usecase.NewProfileUsecase(repo, fs, 5<<20)

			_, err := uc.UpdateProfile(t.Context(), 1, tc.username, tc.status, tc.description)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errTarget != nil && !errors.Is(err, tc.errTarget) {
					t.Errorf("expected %v, got %v", tc.errTarget, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateProfile_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockUserRepository(ctrl)
	fs := mocks.NewMockFileUploader(ctrl)

	repo.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(domain.ErrConflict)

	uc := usecase.NewProfileUsecase(repo, fs, 5<<20)

	_, err := uc.UpdateProfile(t.Context(), 1, "validuser", "", "")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	tests := []struct {
		name        string
		currentPass string
		newPass     string
		wantErr     bool
		errTarget   error
	}{
		{
			name:        "success",
			currentPass: "password123",
			newPass:     "newpassword123",
		},
		{
			name:        "wrong current password",
			currentPass: "wrongpassword",
			newPass:     "newpassword123",
			wantErr:     true,
			errTarget:   domain.ErrUnauthorized,
		},
		{
			name:        "new password too short",
			currentPass: "password123",
			newPass:     "short",
			wantErr:     true,
			errTarget:   domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := testUser()
			repo := mocks.NewMockUserRepository(ctrl)
			fs := mocks.NewMockFileUploader(ctrl)

			repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			if !tc.wantErr {
				repo.EXPECT().UpdatePassword(gomock.Any(), int64(1), gomock.Any()).Return(nil)
			}

			uc := usecase.NewProfileUsecase(repo, fs, 5<<20)

			err := uc.ChangePassword(t.Context(), 1, tc.currentPass, tc.newPass)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errTarget != nil && !errors.Is(err, tc.errTarget) {
					t.Errorf("expected %v, got %v", tc.errTarget, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestChangeEmail(t *testing.T) {
	tests := []struct {
		name      string
		newEmail  string
		password  string
		wantErr   bool
		errTarget error
	}{
		{
			name:     "success",
			newEmail: "new@example.com",
			password: "password123",
		},
		{
			name:      "wrong password",
			newEmail:  "new@example.com",
			password:  "wrongpassword",
			wantErr:   true,
			errTarget: domain.ErrUnauthorized,
		},
		{
			name:      "invalid email",
			newEmail:  "not-an-email",
			password:  "password123",
			wantErr:   true,
			errTarget: domain.ErrInvalidInput,
		},
		{
			name:      "same email",
			newEmail:  "test@example.com",
			password:  "password123",
			wantErr:   true,
			errTarget: domain.ErrInvalidInput,
		},
		{
			name:      "same email case insensitive",
			newEmail:  "TEST@EXAMPLE.COM",
			password:  "password123",
			wantErr:   true,
			errTarget: domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := testUser()
			repo := mocks.NewMockUserRepository(ctrl)
			fs := mocks.NewMockFileUploader(ctrl)

			repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			if !tc.wantErr {
				repo.EXPECT().UpdateEmail(gomock.Any(), int64(1), tc.newEmail).Return(nil)
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			}

			uc := usecase.NewProfileUsecase(repo, fs, 5<<20)

			_, err := uc.ChangeEmail(t.Context(), 1, tc.newEmail, tc.password)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errTarget != nil && !errors.Is(err, tc.errTarget) {
					t.Errorf("expected %v, got %v", tc.errTarget, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestChangeEmail_Conflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := testUser()
	repo := mocks.NewMockUserRepository(ctrl)
	fs := mocks.NewMockFileUploader(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	repo.EXPECT().UpdateEmail(gomock.Any(), int64(1), "new@example.com").Return(domain.ErrConflict)

	uc := usecase.NewProfileUsecase(repo, fs, 5<<20)

	_, err := uc.ChangeEmail(t.Context(), 1, "new@example.com", "password123")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}
