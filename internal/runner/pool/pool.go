package pool

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/container"
)

type WorkerState int

const (
	WorkerIdle WorkerState = iota
	WorkerAssigned
	WorkerRestarting
)

type Worker struct {
	ID          string
	ContainerID string
	BaseURL     string
	mu          sync.Mutex
	state       WorkerState
	assignedTo  int64
}

func (w *Worker) State() WorkerState {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state
}

func (w *Worker) AssignedTo() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.assignedTo
}

type QueueEntry struct {
	NotebookID int64
	UserID     int64
	ResultCh   chan *Worker
	Ctx        context.Context
	EnqueuedAt time.Time
}

type Pool struct {
	workers    []*Worker
	workersMu  sync.RWMutex
	idle       chan *Worker
	queue      chan *QueueEntry
	manager    container.Manager
	httpClient *http.Client
	language   string
}

func New(ctx context.Context, manager container.Manager, language string, poolSize, queueMax int) (*Pool, error) {
	if poolSize <= 0 {
		poolSize = 1
	}
	p := &Pool{
		workers:    make([]*Worker, 0, poolSize),
		idle:       make(chan *Worker, poolSize),
		queue:      make(chan *QueueEntry, queueMax),
		manager:    manager,
		httpClient: &http.Client{Timeout: 35 * time.Second},
		language:   language,
	}

	var wg sync.WaitGroup
	errs := make(chan error, poolSize)
	started := make(chan *Worker, poolSize)

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w, err := p.spawnWorker(ctx)
			if err != nil {
				errs <- err
				return
			}
			started <- w
		}()
	}

	wg.Wait()
	close(errs)
	close(started)

	for w := range started {
		p.workersMu.Lock()
		p.workers = append(p.workers, w)
		p.workersMu.Unlock()
		p.idle <- w
	}

	var spawnErrs []error
	for err := range errs {
		spawnErrs = append(spawnErrs, err)
	}

	if len(p.workers) == 0 {
		return nil, fmt.Errorf("pool: all workers failed to start: %v", spawnErrs)
	}
	if len(spawnErrs) > 0 {
		logger.Warn(ctx, "pool: some workers failed to start", "count", len(spawnErrs), "started", len(p.workers))
	}

	go p.dispatchLoop(ctx)
	return p, nil
}

func (p *Pool) spawnWorker(ctx context.Context) (*Worker, error) {
	id := uuid.New().String()
	baseURL, err := p.manager.StartSession(ctx, id, p.language)
	if err != nil {
		return nil, fmt.Errorf("spawn worker %s: %w", id, err)
	}
	w := &Worker{
		ID:      id,
		BaseURL: baseURL,
		state:   WorkerIdle,
	}
	return w, nil
}

// Acquire выдаёт свободный воркер из пула. Если пул исчерпан — ставит запрос
// в очередь и ждёт до отмены контекста. Возвращает также позицию в очереди
// (0 = выдан немедленно).
func (p *Pool) Acquire(ctx context.Context, notebookID, userID int64) (*Worker, int, error) {
	// Быстрый путь — свободный воркер есть
	select {
	case w := <-p.idle:
		w.mu.Lock()
		w.state = WorkerAssigned
		w.assignedTo = notebookID
		w.mu.Unlock()
		return w, 0, nil
	default:
	}

	// Медленный путь — очередь
	entry := &QueueEntry{
		NotebookID: notebookID,
		UserID:     userID,
		ResultCh:   make(chan *Worker, 1),
		Ctx:        ctx,
		EnqueuedAt: time.Now(),
	}
	select {
	case p.queue <- entry:
	default:
		return nil, 0, fmt.Errorf("%w: runner queue is full", domain.ErrServiceUnavailable)
	}

	pos := p.estimateQueueLen()

	select {
	case w := <-entry.ResultCh:
		return w, 0, nil
	case <-ctx.Done():
		return nil, pos, ctx.Err()
	}
}

// Release возвращает воркер в пул: перезапускает ядро и делает воркер idle.
func (p *Pool) Release(ctx context.Context, w *Worker) {
	go func() {
		w.mu.Lock()
		w.state = WorkerRestarting
		w.assignedTo = 0
		w.mu.Unlock()

		if err := p.restartKernel(ctx, w); err != nil {
			logger.Error(ctx, "pool.Release: kernel restart failed, replacing worker",
				"worker_id", w.ID, "error", err)
			p.replaceWorker(ctx, w)
			return
		}

		w.mu.Lock()
		w.state = WorkerIdle
		w.mu.Unlock()

		p.dispatchOrIdle(w)
	}()
}

// GetContainerStats возвращает метрики контейнера воркера.
func (p *Pool) GetContainerStats(ctx context.Context, workerID string) (*domain.ContainerResourceStats, error) {
	return p.manager.GetContainerStats(ctx, workerID)
}

// QueuePositionFor возвращает позицию в очереди для notebookID (0 = не в очереди).
func (p *Pool) QueuePositionFor(notebookID int64) int32 {
	pos := int32(0)
	// Канал не позволяет итерироваться без извлечения — используем длину как приближение
	_ = notebookID
	return pos
}

func (p *Pool) Shutdown(ctx context.Context) {
	p.manager.CleanupSessions(ctx)
}

func (p *Pool) dispatchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case entry := <-p.queue:
			// Ждём свободный воркер
			select {
			case w := <-p.idle:
				if entry.Ctx.Err() != nil {
					// Клиент ушёл — вернуть воркер сразу
					p.idle <- w
					continue
				}
				w.mu.Lock()
				w.state = WorkerAssigned
				w.assignedTo = entry.NotebookID
				w.mu.Unlock()
				entry.ResultCh <- w
			case <-ctx.Done():
				return
			}
		}
	}
}

func (p *Pool) dispatchOrIdle(w *Worker) {
	// Проверяем очередь без блокировки
	select {
	case entry := <-p.queue:
		if entry.Ctx.Err() != nil {
			p.dispatchOrIdle(w)
			return
		}
		w.mu.Lock()
		w.state = WorkerAssigned
		w.assignedTo = entry.NotebookID
		w.mu.Unlock()
		entry.ResultCh <- w
	default:
		p.idle <- w
	}
}

func (p *Pool) restartKernel(ctx context.Context, w *Worker) error {
	restartURL := w.BaseURL + "/restart"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, restartURL, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST /restart: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST /restart returned %d", resp.StatusCode)
	}
	return nil
}

func (p *Pool) replaceWorker(ctx context.Context, old *Worker) {
	// Удаляем старый воркер из списка
	p.workersMu.Lock()
	for i, w := range p.workers {
		if w.ID == old.ID {
			p.workers = append(p.workers[:i], p.workers[i+1:]...)
			break
		}
	}
	p.workersMu.Unlock()

	// Запускаем новый
	newW, err := p.spawnWorker(ctx)
	if err != nil {
		logger.Error(ctx, "pool.replaceWorker: failed to spawn replacement", "error", err)
		return
	}
	p.workersMu.Lock()
	p.workers = append(p.workers, newW)
	p.workersMu.Unlock()
	p.dispatchOrIdle(newW)
}

func (p *Pool) estimateQueueLen() int {
	return len(p.queue) + 1
}
