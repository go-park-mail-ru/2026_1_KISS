package container

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner"
)

func waitUntilReady(ctx context.Context, httpClient *http.Client, baseURL string, timeout, interval time.Duration) error {
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	healthCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	healthURL := strings.TrimRight(baseURL, "/") + "/health"
	for {
		req, err := http.NewRequestWithContext(healthCtx, http.MethodGet, healthURL, nil)
		if err != nil {
			return fmt.Errorf("build health request: %w", err)
		}

		resp, err := httpClient.Do(req)
		fmt.Printf("err: %v\n", err)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-healthCtx.Done():
			return fmt.Errorf("%w: %s", runner.ErrContainerNotReady, healthURL)
		case <-time.After(interval):
		}
	}
}
