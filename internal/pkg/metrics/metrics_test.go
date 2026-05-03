package metrics

import (
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

func TestShutdownMetricsServer_NilServer(t *testing.T) {
	ShutdownMetricsServer(nil)
}

func TestShutdownMetricsServer(t *testing.T) {
	srv := StartMetricsServer(":0")
	time.Sleep(100 * time.Millisecond)
	ShutdownMetricsServer(srv)
}
