package models

type FileWithDetails struct {
	File        *IPYNBFile       `json:"file"`
	Cells       []IPYNBCell      `json:"cells"`
	Permissions []FilePermission `json:"permissions,omitempty"`
	IsOwner     bool             `json:"is_owner"`
}

type FileListResponse struct {
	Files      []IPYNBFile `json:"files"`
	TotalCount int         `json:"total_count"`
}
