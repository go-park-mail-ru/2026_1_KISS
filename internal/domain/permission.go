package domain

const (
	PermissionReadOnly = "readonly"
	PermissionEditor   = "editor"
)

type FilePermission struct {
	NotebookID      int64
	UserID          int64
	PermissionLevel string
}
