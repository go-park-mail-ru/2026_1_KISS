package hub

import (
	"sync"

	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

const defaultBufSize = 64

type Hub struct {
	mu   sync.RWMutex
	subs map[int64]map[string]chan *pb.NotebookEvent
}

func New() *Hub {
	return &Hub{
		subs: make(map[int64]map[string]chan *pb.NotebookEvent),
	}
}

func (h *Hub) Subscribe(notebookID int64, connID string, bufSize int) chan *pb.NotebookEvent {
	if bufSize <= 0 {
		bufSize = defaultBufSize
	}
	ch := make(chan *pb.NotebookEvent, bufSize)
	h.mu.Lock()
	if h.subs[notebookID] == nil {
		h.subs[notebookID] = make(map[string]chan *pb.NotebookEvent)
	}
	h.subs[notebookID][connID] = ch
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(notebookID int64, connID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns, ok := h.subs[notebookID]
	if !ok {
		return
	}
	if ch, exists := conns[connID]; exists {
		close(ch)
		delete(conns, connID)
	}
	if len(conns) == 0 {
		delete(h.subs, notebookID)
	}
}

func (h *Hub) Publish(notebookID int64, event *pb.NotebookEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs[notebookID] {
		select {
		case ch <- event:
		default:
		}
	}
}
