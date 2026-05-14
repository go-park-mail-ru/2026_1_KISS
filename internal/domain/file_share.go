package domain

import "time"

const (
	FileShareLevelView     = "view"
	FileShareLevelDownload = "download"

	FilePermissionOwner    = "owner"
	FilePermissionView     = "view"
	FilePermissionDownload = "download"
	FilePermissionPublic   = "public"
)

type FileShare struct {
	FileID    string
	UserID    int64
	Email     string
	Level     string
	CreatedAt time.Time
}

func ValidFileShareLevel(level string) bool {
	return level == FileShareLevelView || level == FileShareLevelDownload
}
