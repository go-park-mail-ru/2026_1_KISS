package models

import "time"

type OutputType string

const (
	OutputTypeStream        OutputType = "stream"
	OutputTypeExecuteResult OutputType = "execute_result"
	OutputTypeError         OutputType = "error"
	OutputTypeDisplayData   OutputType = "display_data"
)

type CellOutput struct {
	ID          int64      `json:"id" db:"id"`
	CellID      int64      `json:"cell_id" db:"cell_id"`
	OrderIndex  int        `json:"order_index" db:"order_index"`
	OutputType  OutputType `json:"output_type" db:"output_type"`
	TextContent *string    `json:"text_content,omitempty" db:"text_content"` // может быть null для бинарных данных
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}
