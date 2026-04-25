package hub_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/hub"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

func TestHub_SubscribeAndPublish(t *testing.T) {
	h := hub.New()
	ch := h.Subscribe(1, "conn-1", 4)

	event := &pb.NotebookEvent{NotebookId: 1, ActorId: 42, Type: pb.NotebookEvent_BLOCK_UPDATED}
	h.Publish(1, event)

	got := <-ch
	assert.Equal(t, event, got)
}

func TestHub_Unsubscribe_ClosesChannel(t *testing.T) {
	h := hub.New()
	ch := h.Subscribe(1, "conn-1", 4)
	h.Unsubscribe(1, "conn-1")

	_, open := <-ch
	assert.False(t, open, "channel must be closed after Unsubscribe")
}

func TestHub_Publish_DropsOnFullBuffer(t *testing.T) {
	h := hub.New()
	ch := h.Subscribe(1, "conn-slow", 1)

	ev := &pb.NotebookEvent{NotebookId: 1}
	h.Publish(1, ev)
	h.Publish(1, ev) // buffer full — must not block

	assert.Len(t, ch, 1)
}

func TestHub_Publish_IsolatesNotebooks(t *testing.T) {
	h := hub.New()
	ch1 := h.Subscribe(1, "c1", 4)
	ch2 := h.Subscribe(2, "c2", 4)

	h.Publish(1, &pb.NotebookEvent{NotebookId: 1})

	assert.Len(t, ch1, 1)
	assert.Len(t, ch2, 0, "notebook 2 must not receive event for notebook 1")
}

func TestHub_Unsubscribe_CleansUpEmptyNotebook(t *testing.T) {
	h := hub.New()
	h.Subscribe(5, "c1", 4)
	h.Unsubscribe(5, "c1")

	// publishing to notebook with no subscribers must not panic
	assert.NotPanics(t, func() {
		h.Publish(5, &pb.NotebookEvent{NotebookId: 5})
	})
}
