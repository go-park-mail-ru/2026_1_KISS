//go:generate mockgen -destination=../../mocks/notebook_session_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/runner/notebook_session NotebookSession
package notebook_session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/utils"
)

type NotebookSession interface {
	GetSessionID() string
	LastActivity() time.Time
	ExecuteFromPosition(ctx context.Context, notebook *domain.Notebook, startPosition int) ([]*domain.BlockExecutionResult, error)
	ExecuteBlock(ctx context.Context, block domain.Block) (*domain.BlockExecutionResult, error)
	//UpdateAndExecuteFromBlock(ctx context.Context, notebook *domain.Notebook, blockID int64, newContent string) ([]*domain.BlockExecutionResult, error)
	//Reset()
}

func NewNotebookSession(NotebookID int64,
	SessionID string,
	BaseURL string,
	LastExecuted int, // Position of last successfully executed block
	BlockStates map[int64]*domain.BlockState,
) NotebookSession {

	s := &notebookSession{
		NotebookID:   NotebookID,
		SessionID:    SessionID,
		BaseURL:      BaseURL,
		LastExecuted: LastExecuted,
		BlockStates:  BlockStates,
		client: &http.Client{
			Timeout: 135 * time.Second,
		},
	}
	s.lastActivity.Store(time.Now().UnixNano())
	return s
}

type notebookSession struct {
	NotebookID   int64
	SessionID    string
	BaseURL      string
	LastExecuted int // Position of last successfully executed block
	BlockStates  map[int64]*domain.BlockState
	mu           sync.RWMutex
	client       *http.Client
	lastActivity atomic.Int64 // Unix nanoseconds, updated on every execution
}

func (s *notebookSession) GetSessionID() string {
	return s.SessionID
}

func (s *notebookSession) LastActivity() time.Time {
	return time.Unix(0, s.lastActivity.Load())
}

func (s *notebookSession) ExecuteFromPosition(
	ctx context.Context, notebook *domain.Notebook, startPosition int,
) ([]*domain.BlockExecutionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var results []*domain.BlockExecutionResult

	blocks := notebook.Blocks
	// TODO предполагается что блоки отсортрованы по position - надо это проверить

	for i := startPosition; i < len(blocks); i++ {
		block := blocks[i]
		if block.Type != "code" {
			continue
		}
		if strings.TrimSpace(block.Content) == "" {
			continue
		}
		state, exists := s.BlockStates[block.ID]
		currentHash := utils.ComputeHash(block.Content)

		needsExecution := false
		if !exists {
			needsExecution = true
			state = &domain.BlockState{
				BlockID:   block.ID,
				Position:  block.Position,
				Hash:      currentHash,
				Executed:  false,
				UpdatedAt: time.Now(),
			}
			s.BlockStates[block.ID] = state
		} else if state.Hash != currentHash {
			needsExecution = true
			state.Hash = currentHash
			state.Executed = false
			s.markSubsequentForExecution(block.Position, notebook.Blocks)
		} else if !state.Executed {
			needsExecution = true
		}

		if needsExecution {
			if i > 0 && s.hasDependencyChanges(block.Position, i-1, notebook.Blocks) {
				state.Executed = false
			}

			result, err := s.ExecuteBlock(ctx, block)
			if err != nil {
				results = append(results, &domain.BlockExecutionResult{
					BlockID:  block.ID,
					Position: block.Position,
					Error:    err,
					ErrorMsg: err.Error(),
				})
				return results, fmt.Errorf("block %d execution failed: %w", block.Position, err)
			}

			state.Executed = true
			state.Result = result
			results = append(results, result)
			s.LastExecuted = block.Position
		} else {
			if state.Result != nil {
				results = append(results, state.Result)
			}
		}
	}

	return results, nil
}

func (s *notebookSession) ExecuteBlock(ctx context.Context, block domain.Block) (*domain.BlockExecutionResult, error) {
	startTime := time.Now()
	req := domain.ExecuteRequest{
		Code:    block.Content,
		Timeout: 120,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// if port is not specified - use 8080 as default (when network in docker is bridge ew should use default)
	var fullURL string
	if strings.Count(s.BaseURL, ":") > 1 {
		fullURL = s.BaseURL + "/execute"
	} else {
		fullURL = s.BaseURL + ":8080/execute"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute block: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("execution failed with status %d: %s", resp.StatusCode, string(body))
	}

	var execResp domain.ExecuteResponse
	if err := json.Unmarshal(body, &execResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var stdoutLines, stderrLines []string
	if execResp.Stdout != "" {
		stdoutLines = []string{execResp.Stdout}
	}
	if execResp.Stderr != "" {
		stderrLines = []string{execResp.Stderr}
	}

	result := &domain.BlockExecutionResult{
		BlockID:    block.ID,
		Position:   block.Position,
		Stdout:     stdoutLines,
		Stderr:     stderrLines,
		Result:     execResp.Result,
		ExecutedAt: time.Now(),
		Duration:   time.Since(startTime),
	}

	s.lastActivity.Store(time.Now().UnixNano())
	return result, nil
}

func (s *notebookSession) markSubsequentForExecution(changedPosition int, blocks []domain.Block) {
	for _, block := range blocks {
		if block.Position > changedPosition {
			if state, exists := s.BlockStates[block.ID]; exists {
				state.Executed = false
			}
		}
	}
}

func (s *notebookSession) hasDependencyChanges(currentPos int, lastCheckedPos int, blocks []domain.Block) bool {
	for i := 0; i <= lastCheckedPos; i++ {
		block := blocks[i]
		if state, exists := s.BlockStates[block.ID]; exists {
			if state.Result != nil && state.UpdatedAt.After(state.Result.ExecutedAt) {
				return true
			}
		}
	}
	return false
}
