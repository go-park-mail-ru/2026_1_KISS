package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/profile/usecase"
)

type mockUserRepo struct {
	getByIDFn         func(ctx context.Context, id int64) (*domain.User, error)
	getByEmailFn      func(ctx context.Context, email string) (*domain.User, error)
	updateAvatarURLFn func(ctx context.Context, userID int64, avatarURL string) error
	updateProfileFn   func(ctx context.Context, user *domain.User) error
	updatePasswordFn  func(ctx context.Context, userID int64, passwordHash string) error
	updateEmailFn     func(ctx context.Context, userID int64, email string) error
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) UpdateAvatarURL(ctx context.Context, userID int64, avatarURL string) error {
	if m.updateAvatarURLFn != nil {
		return m.updateAvatarURLFn(ctx, userID, avatarURL)
	}
	return nil
}

func (m *mockUserRepo) UpdateProfile(ctx context.Context, user *domain.User) error {
	if m.updateProfileFn != nil {
		return m.updateProfileFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, userID, passwordHash)
	}
	return nil
}

func (m *mockUserRepo) UpdateEmail(ctx context.Context, userID int64, email string) error {
	if m.updateEmailFn != nil {
		return m.updateEmailFn(ctx, userID, email)
	}
	return nil
}

type mockFileStorage struct {
	saveFn   func(filename string, data io.Reader) (string, error)
	deleteFn func(path string) error
}

func (m *mockFileStorage) Save(filename string, data io.Reader) (string, error) {
	if m.saveFn != nil {
		return m.saveFn(filename, data)
	}
	return "/uploads/" + filename, nil
}

func (m *mockFileStorage) Delete(path string) error {
	if m.deleteFn != nil {
		return m.deleteFn(path)
	}
	return nil
}

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
			user := testUser()
			repo := &mockUserRepo{
				getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
					return user, nil
				},
			}
			fs := &mockFileStorage{}
			uc := usecase.New(repo, fs, 2<<20)

			_, err := uc.UploadAvatar(context.Background(), 1, bytes.NewReader(tc.fileData), tc.fileSize, "")
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
	user := testUser()
	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
			return user, nil
		},
	}
	fs := &mockFileStorage{
		saveFn: func(_ string, _ io.Reader) (string, error) {
			return "", errors.New("disk full")
		},
	}
	uc := usecase.New(repo, fs, 2<<20)

	data := append(jpegHeader, make([]byte, 100)...)
	_, err := uc.UploadAvatar(context.Background(), 1, bytes.NewReader(data), int64(len(data)), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUploadAvatar_DeletesOldAvatar(t *testing.T) {
	user := testUser()
	user.AvatarURL = "/uploads/old-avatar.jpg"
	var deletedPath string

	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
			return user, nil
		},
	}
	fs := &mockFileStorage{
		deleteFn: func(path string) error {
			deletedPath = path
			return nil
		},
	}
	uc := usecase.New(repo, fs, 2<<20)

	data := append(jpegHeader, make([]byte, 100)...)
	_, err := uc.UploadAvatar(context.Background(), 1, bytes.NewReader(data), int64(len(data)), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedPath != "/uploads/old-avatar.jpg" {
		t.Errorf("expected old avatar to be deleted, got deleted path: %q", deletedPath)
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
			user := testUser()
			repo := &mockUserRepo{
				getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
					return user, nil
				},
			}
			uc := usecase.New(repo, &mockFileStorage{}, 5<<20)

			_, err := uc.UpdateProfile(context.Background(), 1, tc.username, tc.status, tc.description)
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
	repo := &mockUserRepo{
		updateProfileFn: func(_ context.Context, _ *domain.User) error {
			return domain.ErrConflict
		},
	}
	uc := usecase.New(repo, &mockFileStorage{}, 5<<20)

	_, err := uc.UpdateProfile(context.Background(), 1, "validuser", "", "")
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
			user := testUser()
			repo := &mockUserRepo{
				getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
					return user, nil
				},
			}
			uc := usecase.New(repo, &mockFileStorage{}, 5<<20)

			err := uc.ChangePassword(context.Background(), 1, tc.currentPass, tc.newPass)
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
			user := testUser()
			repo := &mockUserRepo{
				getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
					return user, nil
				},
			}
			uc := usecase.New(repo, &mockFileStorage{}, 5<<20)

			_, err := uc.ChangeEmail(context.Background(), 1, tc.newEmail, tc.password)
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
	user := testUser()
	repo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ int64) (*domain.User, error) {
			return user, nil
		},
		updateEmailFn: func(_ context.Context, _ int64, _ string) error {
			return domain.ErrConflict
		},
	}
	uc := usecase.New(repo, &mockFileStorage{}, 5<<20)

	_, err := uc.ChangeEmail(context.Background(), 1, "new@example.com", "password123")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}
