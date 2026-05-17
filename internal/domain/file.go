package domain

import "time"

type FileCategory string

const (
	FileCategoryAvatar   FileCategory = "avatars"
	FileCategoryFeedback FileCategory = "feedback"
	FileCategoryDataset  FileCategory = "datasets"
	FileCategoryGeneral  FileCategory = "files"
	FileCategorySession  FileCategory = "sessions"
)

func ValidFileCategory(c FileCategory) bool {
	switch c {
	case FileCategoryAvatar, FileCategoryFeedback, FileCategoryDataset, FileCategoryGeneral, FileCategorySession:
		return true
	default:
		return false
	}
}

type File struct {
	ID             string
	OwnerID        int64
	NotebookID     *int64
	Category       FileCategory
	Filename       string
	StorageKey     string
	URL            string
	MIMEType       string
	Size           int64
	CreatedAt      time.Time
	IsPublic       bool
	ShareToken     *string
	ShareExpiresAt *time.Time
	DownloadsCount int64
	YourPermission string
	OwnerEmail     string
}

type StorageStats struct {
	TotalFiles      int64
	TotalSizeBytes  int64
	FilesByCategory map[FileCategory]int64
	SizeByCategory  map[FileCategory]int64
}
