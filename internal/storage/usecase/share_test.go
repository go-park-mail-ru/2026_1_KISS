package usecase

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestShareFile_OwnerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, shareRepo := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 1}, nil)
	shareRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, s *domain.FileShare) error {
		s.CreatedAt = time.Now()
		return nil
	})

	share, err := uc.ShareFile(context.Background(), 1, "file-1", 2, "download")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if share.Level != "download" {
		t.Errorf("expected level download, got %s", share.Level)
	}
}

func TestShareFile_InvalidLevel(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, _, _, _ := newTestUsecaseWithShare(ctrl)

	_, err := uc.ShareFile(context.Background(), 1, "file-1", 2, "wat")
	if err == nil {
		t.Fatal("expected error for invalid level")
	}
}

func TestShareFile_SelfShareRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 1}, nil)

	_, err := uc.ShareFile(context.Background(), 1, "file-1", 1, "view")
	if err == nil {
		t.Fatal("expected error for self share")
	}
}

func TestShareFile_NotOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 99}, nil)

	_, err := uc.ShareFile(context.Background(), 1, "file-1", 2, "view")
	if err != domain.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRevokeShare_NotOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 99}, nil)

	err := uc.RevokeShare(context.Background(), 1, "file-1", 2)
	if err != domain.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestListShares_OwnerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, shareRepo := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 1}, nil)
	shareRepo.EXPECT().GetByFileID(gomock.Any(), "file-1").Return([]domain.FileShare{
		{FileID: "file-1", UserID: 2, Level: "view", Email: "x@y.z"},
	}, nil)

	shares, err := uc.ListShares(context.Background(), 1, "file-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("expected 1 share, got %d", len(shares))
	}
}

func TestSetFilePublic_Enable_GeneratesToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 1}, nil)
	repo.EXPECT().SetPublic(gomock.Any(), "file-1", int64(1), true, gomock.Any(), gomock.Any()).Return(nil)

	exp := time.Now().Add(24 * time.Hour)
	file, err := uc.SetFilePublic(context.Background(), 1, "file-1", true, &exp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.ShareToken == nil {
		t.Fatal("expected share token to be generated")
	}
	if !file.IsPublic {
		t.Errorf("expected file to be public")
	}
}

func TestSetFilePublic_Disable_ClearsToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	tok := "old-token"
	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 1, ShareToken: &tok, IsPublic: true}, nil)
	repo.EXPECT().SetPublic(gomock.Any(), "file-1", int64(1), false, (*string)(nil), (*time.Time)(nil)).Return(nil)

	file, err := uc.SetFilePublic(context.Background(), 1, "file-1", false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.ShareToken != nil {
		t.Errorf("expected share token to be cleared")
	}
}

func TestGetSharedFileByToken_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	past := time.Now().Add(-time.Hour)
	repo.EXPECT().GetByShareToken(gomock.Any(), "tok-1").Return(&domain.File{
		ID: "file-1", IsPublic: true, ShareExpiresAt: &past,
	}, nil)

	_, err := uc.GetSharedFileByToken(context.Background(), "tok-1")
	if err == nil {
		t.Fatal("expected expired error")
	}
}

func TestGetSharedFileByToken_NotPublic(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByShareToken(gomock.Any(), "tok-1").Return(&domain.File{
		ID: "file-1", IsPublic: false,
	}, nil)

	_, err := uc.GetSharedFileByToken(context.Background(), "tok-1")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetSharedFileByToken_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByShareToken(gomock.Any(), "tok-1").Return(&domain.File{
		ID: "file-1", IsPublic: true,
	}, nil)

	file, err := uc.GetSharedFileByToken(context.Background(), "tok-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.YourPermission != domain.FilePermissionPublic {
		t.Errorf("expected public permission, got %s", file.YourPermission)
	}
}

func TestRenameFile_InvalidName(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, _, _, _ := newTestUsecaseWithShare(ctrl)

	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only spaces", "   "},
		{"path separator slash", "foo/bar.txt"},
		{"path separator backslash", `foo\bar.txt`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.RenameFile(context.Background(), 1, "file-1", tc.input)
			if err == nil {
				t.Fatalf("expected error for %q", tc.input)
			}
		})
	}
}

func TestRenameFile_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().Rename(gomock.Any(), "file-1", int64(1), "new.txt").Return(nil)
	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 1, Filename: "new.txt"}, nil)

	file, err := uc.RenameFile(context.Background(), 1, "file-1", "new.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.Filename != "new.txt" {
		t.Errorf("expected new.txt, got %s", file.Filename)
	}
}

func TestGetDownloadable_Owner(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 1}, nil)

	_, ok, err := uc.GetDownloadable(context.Background(), "file-1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("expected owner to be allowed")
	}
}

func TestGetDownloadable_PublicNotExpired(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 99, IsPublic: true}, nil)

	_, ok, err := uc.GetDownloadable(context.Background(), "file-1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("expected public file to be downloadable")
	}
}

func TestGetDownloadable_PublicExpired(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, shareRepo := newTestUsecaseWithShare(ctrl)

	past := time.Now().Add(-time.Hour)
	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 99, IsPublic: true, ShareExpiresAt: &past}, nil)
	shareRepo.EXPECT().GetPermission(gomock.Any(), "file-1", int64(5)).Return(nil, domain.ErrNotFound)

	_, ok, err := uc.GetDownloadable(context.Background(), "file-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected expired public to be not downloadable")
	}
}

func TestGetDownloadable_ViewOnlyShareDenies(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, shareRepo := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 99}, nil)
	shareRepo.EXPECT().GetPermission(gomock.Any(), "file-1", int64(5)).Return(&domain.FileShare{
		FileID: "file-1", UserID: 5, Level: "view",
	}, nil)

	_, ok, _ := uc.GetDownloadable(context.Background(), "file-1", 5)
	if ok {
		t.Errorf("expected view-only share to be denied download")
	}
}

func TestListSharedWithMe_DefaultLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, _, _, shareRepo := newTestUsecaseWithShare(ctrl)

	shareRepo.EXPECT().ListByUserID(gomock.Any(), int64(1), 20, 0).Return(nil, 0, nil)

	_, _, err := uc.ListSharedWithMe(context.Background(), 1, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListSharedWithMe_ClampsLargeLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, _, _, shareRepo := newTestUsecaseWithShare(ctrl)

	shareRepo.EXPECT().ListByUserID(gomock.Any(), int64(1), 100, 0).Return(nil, 0, nil)

	_, _, _ = uc.ListSharedWithMe(context.Background(), 1, 500, 0)
}

func TestIncrementDownloadCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, _ := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().IncrementDownloads(gomock.Any(), "file-1").Return(nil)

	if err := uc.IncrementDownloadCount(context.Background(), "file-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDownloadable_DownloadShareAllows(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc, repo, _, shareRepo := newTestUsecaseWithShare(ctrl)

	repo.EXPECT().GetByID(gomock.Any(), "file-1").Return(&domain.File{ID: "file-1", OwnerID: 99}, nil)
	shareRepo.EXPECT().GetPermission(gomock.Any(), "file-1", int64(5)).Return(&domain.FileShare{
		FileID: "file-1", UserID: 5, Level: "download",
	}, nil)

	_, ok, err := uc.GetDownloadable(context.Background(), "file-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("expected download share to allow")
	}
}
