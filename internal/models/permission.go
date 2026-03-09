package models

type PermissionLevel string

const (
	PermReadOnly PermissionLevel = "readonly"
	PermComment  PermissionLevel = "comment"
	PermEdit     PermissionLevel = "edit"
)

type FilePermission struct {
	FileID          int64           `json:"file_id" db:"file_id"`
	UserID          int64           `json:"user_id" db:"user_id"`
	PermissionLevel PermissionLevel `json:"permission_level" db:"permission_level"`
}

type ShareFileRequest struct {
	Email           string          `json:"email" validate:"required,email"`
	PermissionLevel PermissionLevel `json:"permission_level" validate:"required,oneof=readonly comment edit"`
}
