package domain

import (
	"fmt"
	"time"
)

type ExecuteRequest struct {
	Code    string  `json:"code"`
	Timeout float64 `json:"timeout,omitempty"`
}

type ExecuteResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Result string `json:"result"`
}

type BlockExecutionResult struct {
	BlockID    int64         `json:"block_id"`
	Position   int           `json:"position"`
	Stdout     []string      `json:"stdout,omitempty"`
	Stderr     []string      `json:"stderr,omitempty"`
	Result     string        `json:"result,omitempty"`
	Error      error         `json:"-"`
	ExecutedAt time.Time     `json:"executed_at"`
	Duration   time.Duration `json:"duration"`
}

func (r *BlockExecutionResult) String() string {
	if r == nil {
		return "<nil>"
	}

	var result string
	result += fmt.Sprintf("Block #%d (ID: %d) executed at %s, ",
		r.Position, r.BlockID, r.ExecutedAt.Format("2006-01-02 15:04:05.000"))
	result += fmt.Sprintf("Duration: %v\n", r.Duration)

	if r.Error != nil {
		result += fmt.Sprintf("Error: %v\n", r.Error)
		return result
	}

	if len(r.Stdout) > 0 {
		result += "STDOUT:\n"
		for i, line := range r.Stdout {
			if line != "" {
				result += fmt.Sprintf("  [%d] %s\n", i+1, line)
			}
		}
	}

	if len(r.Stderr) > 0 {
		result += "STDERR:\n"
		for i, line := range r.Stderr {
			if line != "" {
				result += fmt.Sprintf("  [%d] %s\n", i+1, line)
			}
		}
	}

	if r.Result != "" {
		result += fmt.Sprintf("RESULT:\n  %s\n", r.Result)
	}

	if len(r.Stdout) == 0 && len(r.Stderr) == 0 && r.Result == "" && r.Error == nil {
		result += "No output or result produced\n"
	}

	return result
}

type BlockState struct {
	BlockID   int64
	Position  int
	Hash      string // Hash of block content to detect changes
	Executed  bool
	Result    *BlockExecutionResult
	UpdatedAt time.Time
}
