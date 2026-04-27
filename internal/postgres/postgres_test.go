package postgres

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test — downloads Postgres binaries on first run")
	}
	tmp := filepath.Join(t.TempDir(), "pg")
	mgr, err := New(tmp, 0)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	ctx := context.Background()
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer mgr.Stop(ctx)

	if mgr.DSN() == "" {
		t.Fatal("expected DSN")
	}
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if err := mgr.Ping(ctx); err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("postgres never became ready")
}
