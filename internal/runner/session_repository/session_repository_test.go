package session_repository

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

func makeNotebook() *domain.Notebook {
	return &domain.Notebook{
		ID:      1,
		OwnerID: 10,
		Title:   "test notebook",
		Blocks: []domain.Block{
			{ID: 100, NotebookID: 1, Content: "print('hello')", Position: 0},
			{ID: 101, NotebookID: 1, Content: "x = 1 + 2", Position: 1},
		},
	}
}

func TestCreateSession_StoresAndRetrievable(t *testing.T) {
	repo := NewExecutionSessionRepository(120 * time.Second)
	nb := makeNotebook()

	session, err := repo.CreateSession(nb, "http://localhost:8080", "session-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.GetSessionID() != "session-1" {
		t.Errorf("expected session ID 'session-1', got %q", session.GetSessionID())
	}

	got, ok := repo.GetSession(nb.ID)
	if !ok {
		t.Fatal("expected session to be found")
	}
	if got.GetSessionID() != "session-1" {
		t.Errorf("expected session ID 'session-1', got %q", got.GetSessionID())
	}
}

func TestGetSession_ExistingKey(t *testing.T) {
	repo := NewExecutionSessionRepository(120 * time.Second)
	nb := makeNotebook()
	_, _ = repo.CreateSession(nb, "http://localhost:8080", "session-2")

	session, ok := repo.GetSession(nb.ID)
	if !ok {
		t.Fatal("expected true for existing key")
	}
	if session == nil {
		t.Fatal("expected non-nil session")
	}
}

func TestGetSession_MissingKey(t *testing.T) {
	repo := NewExecutionSessionRepository(120 * time.Second)

	session, ok := repo.GetSession(999)
	if ok {
		t.Error("expected false for missing key")
	}
	if session != nil {
		t.Error("expected nil session for missing key")
	}
}

func TestDeleteSession_RemovesEntry(t *testing.T) {
	repo := NewExecutionSessionRepository(120 * time.Second)
	nb := makeNotebook()
	_, _ = repo.CreateSession(nb, "http://localhost:8080", "session-3")

	repo.DeleteSession(nb.ID)

	_, ok := repo.GetSession(nb.ID)
	if ok {
		t.Error("expected session to be deleted")
	}
}

func TestDeleteSession_NonExistentKey(t *testing.T) {
	repo := NewExecutionSessionRepository(120 * time.Second)
	repo.DeleteSession(999)
}
