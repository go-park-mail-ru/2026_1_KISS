package models

import (
	"time"
)

type ProgrammingLanguage string

const (
	LangPython ProgrammingLanguage = "python"
	LangR      ProgrammingLanguage = "r"
)

type IPYNBFile struct {
	ID                  int64               `json:"id" db:"id"`
	OwnerID             int64               `json:"owner_id" db:"owner_id"`
	Title               string              `json:"title" db:"title"`
	NbFormat            int                 `json:"nbformat" db:"nbformat"`
	NbFormatMinor       int                 `json:"nbformat_minor" db:"nbformat_minor"`
	ProgrammingLanguage ProgrammingLanguage `json:"programming_language" db:"programming_language"`
	CreatedAt           time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at" db:"updated_at"`
}

type CellType string

const (
	CellTypeCode     CellType = "code"
	CellTypeMarkdown CellType = "markdown"
	CellTypeRaw      CellType = "raw"
)

type IPYNBCell struct {
	ID             int64     `json:"id" db:"id"`
	FileID         int64     `json:"file_id" db:"file_id"`
	OrderIndex     int       `json:"order_index" db:"order_index"`
	CellType       CellType  `json:"cell_type" db:"cell_type"`
	Source         string    `json:"source" db:"source"`
	ExecutionCount *int      `json:"execution_count,omitempty" db:"execution_count"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type CreateFileRequest struct {
	Title               string              `json:"title" validate:"required"`
	ProgrammingLanguage ProgrammingLanguage `json:"programming_language" validate:"required,oneof=python r"`
	NbFormat            *int                `json:"nbformat,omitempty"`
	NbFormatMinor       *int                `json:"nbformat_minor,omitempty"`
}

type UpdateFileRequest struct {
	Title               *string              `json:"title,omitempty"`
	ProgrammingLanguage *ProgrammingLanguage `json:"programming_language,omitempty"`
}

type CreateCellRequest struct {
	CellType   CellType `json:"cell_type" validate:"required,oneof=code markdown raw"`
	Source     string   `json:"source"`
	OrderIndex *int     `json:"order_index,omitempty"`
}

type UpdateCellRequest struct {
	Source     *string   `json:"source,omitempty"`
	CellType   *CellType `json:"cell_type,omitempty"`
	OrderIndex *int      `json:"order_index,omitempty"`
}
