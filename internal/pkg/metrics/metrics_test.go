package metrics

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestStartMetricsServer(t *testing.T) {
	srv := StartMetricsServer(":0")
	if srv == nil {
		t.Fatal("expected non-nil server")
	}

	if srv.Addr == "" {
		t.Fatal("expected non-empty address")
	}

	if srv.Handler == nil {
		t.Fatal("expected non-nil handler")
	}

	if srv.ReadHeaderTimeout == 0 {
		t.Fatal("expected non-zero ReadHeaderTimeout")
	}

	ShutdownMetricsServer(srv)
}

func TestStartMetricsServer_MetricsEndpoint(t *testing.T) {
	srv := StartMetricsServer(":0")
	defer ShutdownMetricsServer(srv)

	time.Sleep(100 * time.Millisecond)

	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestShutdownMetricsServer_NilServer(t *testing.T) {
	ShutdownMetricsServer(nil)
}

func TestShutdownMetricsServer(t *testing.T) {
	srv := StartMetricsServer(":0")
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe()
	}()

	ShutdownMetricsServer(srv)

	select {
	case err := <-done:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("shutdown timeout")
	}
}
