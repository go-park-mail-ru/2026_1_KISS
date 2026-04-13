package usecase_test

import (
	"bytes"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/profile/usecase"
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

var jpegHeader = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
var bmpHeader = []byte{0x42, 0x4D, 0x36, 0x00, 0x0C, 0x00, 0x00, 0x00, 0x00, 0x00, 0x36, 0x00, 0x00, 0x00}
var pngHeader = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}

func TestUploadAvatar(t *testing.T) {
	tests := []struct {
		name      string
		fileData  []byte
		fileSize  int64
		wantErr   bool
		errTarget error
	}{
		{
			name:     "success jpeg",
			fileData: append(jpegHeader, make([]byte, 100)...),
			fileSize: int64(len(jpegHeader) + 100),
		},
		{
			name:     "success bmp",
			fileData: append(bmpHeader, make([]byte, 100)...),
			fileSize: int64(len(bmpHeader) + 100),
		},
		{
			name:     "success png",
			fileData: append(pngHeader, make([]byte, 100)...),
			fileSize: int64(len(pngHeader) + 100),
		},
		{
			name:      "file too large",
			fileData:  jpegHeader,
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
			repo := mocks.NewMockuserRepository(ctrl)
			fs := mocks.NewMockFileStorage(ctrl)

			needsRepo := !tc.wantErr || (!errors.Is(tc.errTarget, domain.ErrInvalidInput))
			if needsRepo {
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
				fs.EXPECT().Save(gomock.Any(), gomock.Any()).Return("/uploads/new.jpg", nil)
				repo.EXPECT().UpdateAvatarURL(gomock.Any(), int64(1), "/uploads/new.jpg").Return(nil)
				fs.EXPECT().Delete(user.AvatarURL).Return(nil)
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			}

			uc := usecase.New(repo, fs, 2<<20)

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
	repo := mocks.NewMockuserRepository(ctrl)
	fs := mocks.NewMockFileStorage(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	fs.EXPECT().Save(gomock.Any(), gomock.Any()).Return("", errors.New("disk full"))

	uc := usecase.New(repo, fs, 2<<20)

	data := append(jpegHeader, make([]byte, 100)...)
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

	repo := mocks.NewMockuserRepository(ctrl)
	fs := mocks.NewMockFileStorage(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	fs.EXPECT().Save(gomock.Any(), gomock.Any()).Return("/uploads/new.jpg", nil)
	repo.EXPECT().UpdateAvatarURL(gomock.Any(), int64(1), "/uploads/new.jpg").Return(nil)
	fs.EXPECT().Delete("/uploads/old-avatar.jpg").Return(nil)
	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)

	uc := usecase.New(repo, fs, 2<<20)

	data := append(jpegHeader, make([]byte, 100)...)
	_, err := uc.UploadAvatar(t.Context(), 1, bytes.NewReader(data), int64(len(data)), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadAvatar_UpdateAvatarURLError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	user := testUser()
	repo := mocks.NewMockuserRepository(ctrl)
	fs := mocks.NewMockFileStorage(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	fs.EXPECT().Save(gomock.Any(), gomock.Any()).Return("/uploads/new.jpg", nil)
	repo.EXPECT().UpdateAvatarURL(gomock.Any(), int64(1), "/uploads/new.jpg").Return(errors.New("db error"))
	fs.EXPECT().Delete("/uploads/new.jpg").Return(nil)

	uc := usecase.New(repo, fs, 2<<20)

	data := append(jpegHeader, make([]byte, 100)...)
	_, err := uc.UploadAvatar(t.Context(), 1, bytes.NewReader(data), int64(len(data)), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUploadAvatar_GetByIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockuserRepository(ctrl)
	fs := mocks.NewMockFileStorage(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(nil, domain.ErrNotFound)

	uc := usecase.New(repo, fs, 2<<20)

	data := append(jpegHeader, make([]byte, 100)...)
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

	repo := mocks.NewMockuserRepository(ctrl)
	fs := mocks.NewMockFileStorage(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	fs.EXPECT().Save(gomock.Any(), gomock.Any()).Return("/uploads/new.jpg", nil)
	repo.EXPECT().UpdateAvatarURL(gomock.Any(), int64(1), "/uploads/new.jpg").Return(nil)
	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)

	uc := usecase.New(repo, fs, 2<<20)

	data := append(jpegHeader, make([]byte, 100)...)
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
			repo := mocks.NewMockuserRepository(ctrl)
			fs := mocks.NewMockFileStorage(ctrl)

			if !tc.wantErr {
				repo.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(nil)
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			}

			uc := usecase.New(repo, fs, 5<<20)

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

	repo := mocks.NewMockuserRepository(ctrl)
	fs := mocks.NewMockFileStorage(ctrl)

	repo.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(domain.ErrConflict)

	uc := usecase.New(repo, fs, 5<<20)

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
			repo := mocks.NewMockuserRepository(ctrl)
			fs := mocks.NewMockFileStorage(ctrl)

			repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			if !tc.wantErr {
				repo.EXPECT().UpdatePassword(gomock.Any(), int64(1), gomock.Any()).Return(nil)
			}

			uc := usecase.New(repo, fs, 5<<20)

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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := testUser()
			repo := mocks.NewMockuserRepository(ctrl)
			fs := mocks.NewMockFileStorage(ctrl)

			repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			if !tc.wantErr {
				repo.EXPECT().UpdateEmail(gomock.Any(), int64(1), tc.newEmail).Return(nil)
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
			}

			uc := usecase.New(repo, fs, 5<<20)

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
	repo := mocks.NewMockuserRepository(ctrl)
	fs := mocks.NewMockFileStorage(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(user, nil)
	repo.EXPECT().UpdateEmail(gomock.Any(), int64(1), "new@example.com").Return(domain.ErrConflict)

	uc := usecase.New(repo, fs, 5<<20)

	_, err := uc.ChangeEmail(t.Context(), 1, "new@example.com", "password123")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}
