package container

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultAttemptInterval = 250 * time.Millisecond
	defaultTimeout         = 15 * time.Second
)

func waitUntilReady(ctx context.Context, httpClient *http.Client, baseURL string, timeout, interval time.Duration) error {
	if interval <= 0 {
		interval = defaultAttemptInterval
	}
	if timeout <= 0 {
		timeout = defaultTimeout
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
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-healthCtx.Done():
			return fmt.Errorf("%w: %s", ErrContainerNotReady, healthURL)
		case <-time.After(interval):
		}
	}
}
