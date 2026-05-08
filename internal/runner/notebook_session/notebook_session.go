//go:generate go run go.uber.org/mock/mockgen -destination=../../mocks/notebook_session_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/runner/notebook_session NotebookSession
package notebook_session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
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
	ExecuteBlockStreaming(ctx context.Context, block domain.Block, onChunk func(chunkType, data string)) (*domain.BlockExecutionResult, error)
}

func NewNotebookSession(NotebookID int64,
	SessionID string,
	BaseURL string,
	LastExecuted int,
	BlockStates map[int64]*domain.BlockState,
	execTimeout time.Duration,
) NotebookSession {
	if execTimeout == 0 {
		execTimeout = 120 * time.Second
	}
	s := &notebookSession{
		NotebookID:   NotebookID,
		SessionID:    SessionID,
		BaseURL:      BaseURL,
		LastExecuted: LastExecuted,
		BlockStates:  BlockStates,
		execTimeout:  execTimeout,
		client: &http.Client{
			Timeout: execTimeout + 15*time.Second,
		},
	}
	s.lastActivity.Store(time.Now().UnixNano())
	return s
}

type notebookSession struct {
	NotebookID   int64
	SessionID    string
	BaseURL      string
	LastExecuted int
	BlockStates  map[int64]*domain.BlockState
	execTimeout  time.Duration
	mu           sync.RWMutex
	client       *http.Client
	lastActivity atomic.Int64
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

	blocks := make([]domain.Block, len(notebook.Blocks))
	copy(blocks, notebook.Blocks)
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].Position < blocks[j].Position })

	for _, block := range blocks {
		if state, exists := s.BlockStates[block.ID]; exists {
			if state.Position != block.Position {
				for _, b := range blocks {
					if st, ok := s.BlockStates[b.ID]; ok {
						st.Position = b.Position
						st.Executed = false
					}
				}
				break
			}
		}
	}

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
		Timeout: s.execTimeout.Seconds(),
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var fullURL string
	parsed, err := url.Parse(s.BaseURL)
	if err != nil || parsed.Port() != "" {
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
		Outputs:    execResp.Outputs,
		ExecutedAt: time.Now(),
		Duration:   time.Since(startTime),
	}

	s.lastActivity.Store(time.Now().UnixNano())
	return result, nil
}

func (s *notebookSession) ExecuteBlockStreaming(ctx context.Context, block domain.Block, onChunk func(chunkType, data string)) (*domain.BlockExecutionResult, error) {
	startTime := time.Now()
	req := domain.ExecuteRequest{
		Code:    block.Content,
		Timeout: s.execTimeout.Seconds(),
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var fullURL string
	parsed, err := url.Parse(s.BaseURL)
	if err != nil || parsed.Port() != "" {
		fullURL = s.BaseURL + "/execute/stream"
	} else {
		fullURL = s.BaseURL + ":8080/execute/stream"
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("execution failed with status %d: %s", resp.StatusCode, string(body))
	}

	var stdoutChunks, stderrChunks []string
	var resultText string
	var outputs []domain.OutputItem

	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk struct {
			Type     string `json:"type"`
			Data     string `json:"data"`
			MimeType string `json:"mime_type,omitempty"`
		}
		if err := decoder.Decode(&chunk); err != nil {
			break // EOF or read error
		}
		switch chunk.Type {
		case "stdout":
			stdoutChunks = append(stdoutChunks, chunk.Data)
			onChunk("stdout", chunk.Data)
		case "stderr":
			stderrChunks = append(stderrChunks, chunk.Data)
			onChunk("stderr", chunk.Data)
		case "result":
			resultText = chunk.Data
		case "output":
			outputs = append(outputs, domain.OutputItem{MimeType: chunk.MimeType, Data: chunk.Data})
		case "error":
			stderrChunks = append(stderrChunks, chunk.Data)
			onChunk("stderr", chunk.Data)
		}
	}

	result := &domain.BlockExecutionResult{
		BlockID:    block.ID,
		Position:   block.Position,
		Stdout:     stdoutChunks,
		Stderr:     stderrChunks,
		Result:     resultText,
		Outputs:    outputs,
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
