package domain

type FilePermission struct {
	NotebookID      int64
	UserID          int64
	PermissionLevel string
}
