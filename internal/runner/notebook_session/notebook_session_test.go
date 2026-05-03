package notebook_session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func TestNewNotebookSession(t *testing.T) {
	blockStates := make(map[int64]*domain.BlockState)
	session := NewNotebookSession(
		1,
		"session-123",
		"http://localhost:8080",
		0,
		blockStates,
		30*time.Second,
	)

	ns, ok := session.(*notebookSession)
	if !ok {
		t.Fatal("expected notebookSession type")
	}

	if ns.GetSessionID() != "session-123" {
		t.Errorf("expected session ID session-123, got %s", ns.GetSessionID())
	}
	if ns.NotebookID != 1 {
		t.Errorf("expected notebook ID 1, got %d", ns.NotebookID)
	}
	if ns.execTimeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", ns.execTimeout)
	}
}

func TestNewNotebookSession_DefaultTimeout(t *testing.T) {
	blockStates := make(map[int64]*domain.BlockState)
	session := NewNotebookSession(
		1,
		"session-123",
		"http://localhost:8080",
		0,
		blockStates,
		0,
	)

	ns, ok := session.(*notebookSession)
	if !ok {
		t.Fatal("expected notebookSession type")
	}

	if ns.execTimeout != 120*time.Second {
		t.Errorf("expected default timeout 120s, got %v", ns.execTimeout)
	}
}

func TestGetSessionID(t *testing.T) {
	blockStates := make(map[int64]*domain.BlockState)
	session := NewNotebookSession(
		1,
		"my-session-id",
		"http://localhost:8080",
		0,
		blockStates,
		30*time.Second,
	)

	if session.GetSessionID() != "my-session-id" {
		t.Errorf("expected my-session-id, got %s", session.GetSessionID())
	}
}

func TestLastActivity(t *testing.T) {
	blockStates := make(map[int64]*domain.BlockState)
	session := NewNotebookSession(
		1,
		"session-123",
		"http://localhost:8080",
		0,
		blockStates,
		30*time.Second,
	)

	lastActivity := session.LastActivity()
	if lastActivity.IsZero() {
		t.Fatal("expected non-zero last activity")
	}

	if time.Since(lastActivity) > 1*time.Second {
		t.Errorf("expected recent activity, got %v ago", time.Since(lastActivity))
	}
}

func TestExecuteBlock_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := domain.ExecuteResponse{
			Result: "5",
			Stdout: "output",
			Stderr: "",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	blockStates := make(map[int64]*domain.BlockState)
	session := NewNotebookSession(
		1,
		"session-123",
		server.URL,
		0,
		blockStates,
		30*time.Second,
	)

	block := domain.Block{
		ID:       1,
		Position: 0,
		Type:     "code",
		Content:  "print('hello')",
	}

	result, err := session.ExecuteBlock(context.Background(), block)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.BlockID != 1 {
		t.Errorf("expected block ID 1, got %d", result.BlockID)
	}
	if result.Result != "5" {
		t.Errorf("expected result '5', got %s", result.Result)
	}
	if len(result.Stdout) == 0 {
		t.Fatal("expected stdout to be present")
	}
}

func TestExecuteBlock_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("execution error"))
	}))
	defer server.Close()

	blockStates := make(map[int64]*domain.BlockState)
	session := NewNotebookSession(
		1,
		"session-123",
		server.URL,
		0,
		blockStates,
		30*time.Second,
	)

	block := domain.Block{
		ID:       1,
		Position: 0,
		Type:     "code",
		Content:  "print('hello')",
	}

	_, err := session.ExecuteBlock(context.Background(), block)
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestExecuteFromPosition_EmptyBlocks(t *testing.T) {
	blockStates := make(map[int64]*domain.BlockState)
	session := NewNotebookSession(
		1,
		"session-123",
		"http://localhost:8080",
		0,
		blockStates,
		30*time.Second,
	)

	notebook := &domain.Notebook{
		Blocks: []domain.Block{},
	}

	results, err := session.ExecuteFromPosition(context.Background(), notebook, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestExecuteFromPosition_SkipsNonCodeBlocks(t *testing.T) {
	blockStates := make(map[int64]*domain.BlockState)
	session := NewNotebookSession(
		1,
		"session-123",
		"http://localhost:8080",
		0,
		blockStates,
		30*time.Second,
	)

	notebook := &domain.Notebook{
		Blocks: []domain.Block{
			{
				ID:       1,
				Position: 0,
				Type:     "markdown",
				Content:  "# Title",
			},
			{
				ID:       2,
				Position: 1,
				Type:     "code",
				Content:  "   ",
			},
		},
	}

	results, err := session.ExecuteFromPosition(context.Background(), notebook, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
