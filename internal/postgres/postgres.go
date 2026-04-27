// Package postgres manages the embedded Postgres lifecycle for the .app
// build path. In dev (DB_HOST set), the binary connects to a host Postgres
// and this package is unused.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	embedded "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

// Manager owns the lifecycle of an embedded Postgres instance.
//
// On Start(), if the data dir already has a running Postgres (from a
// previous binary that crashed without cleaning up), Manager reconnects to
// it instead of failing on the lock file. If the lock file is stale
// (process is dead), Manager removes it and starts fresh.
type Manager struct {
	pg       *embedded.EmbeddedPostgres
	port     uint32
	dataDir  string
	reusing  bool // true when Start adopted an existing instance
}

// New creates a Manager for an embedded Postgres listening on port (0 = pick
// free) with data files under dataDir. Caller must Start before Ping/DSN.
func New(dataDir string, port uint32) (*Manager, error) {
	if port == 0 {
		var err error
		port, err = findFreePort()
		if err != nil {
			return nil, fmt.Errorf("postgres: find free port: %w", err)
		}
	}
	cfg := embedded.DefaultConfig().
		Port(port).
		DataPath(dataDir).
		Database("localfinance").
		Username("postgres").
		Password("postgres")
	pg := embedded.NewDatabase(cfg)
	return &Manager{pg: pg, port: port, dataDir: dataDir}, nil
}

// Start launches the embedded Postgres process — or adopts an existing one
// from a previous launch if its lock file is still valid.
func (m *Manager) Start(ctx context.Context) error {
	if existingPort, ok := readRunningPostgres(m.dataDir); ok {
		// Adopt: skip embedded.Start, reuse the running instance.
		m.port = existingPort
		m.reusing = true
		slog.Info("adopting existing embedded postgres", "port", existingPort, "data", m.dataDir)
		// Verify connectivity before trusting it.
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := m.Ping(pingCtx); err != nil {
			// Lock file claims a live PID but we can't connect. Treat as
			// stale and clean up.
			slog.Warn("adopted postgres unreachable — clearing lock and starting fresh", "err", err)
			_ = os.Remove(filepath.Join(m.dataDir, "postmaster.pid"))
			m.reusing = false
		} else {
			return nil
		}
	}
	clearStaleLock(m.dataDir)
	return m.pg.Start()
}

// Stop terminates the embedded Postgres process. No-op if we adopted an
// existing instance — that one keeps running, leaving the data dir in a
// state the next Start can adopt or clean up.
func (m *Manager) Stop(_ context.Context) error {
	if m.reusing {
		return nil
	}
	return m.pg.Stop()
}

// DSN returns a libpq-style connection string for the embedded instance.
func (m *Manager) DSN() string {
	return fmt.Sprintf("host=localhost port=%d user=postgres password=postgres dbname=localfinance sslmode=disable", m.port)
}

// Port returns the TCP port the embedded instance is bound to.
func (m *Manager) Port() uint32 { return m.port }

// Ping opens a short-lived connection to verify the server is up.
func (m *Manager) Ping(ctx context.Context) error {
	db, err := sql.Open("postgres", m.DSN())
	if err != nil {
		return fmt.Errorf("postgres: open: %w", err)
	}
	defer db.Close()
	return db.PingContext(ctx)
}

// readRunningPostgres parses postmaster.pid in dataDir and returns the port
// IFF the PID listed is still alive. Returns false otherwise.
//
// postmaster.pid format (one value per line):
//   1: PID
//   2: data dir
//   3: start time
//   4: port number
//   5: socket dir
//   6: listen address
//   ...
func readRunningPostgres(dataDir string) (uint32, bool) {
	data, err := os.ReadFile(filepath.Join(dataDir, "postmaster.pid"))
	if err != nil {
		return 0, false
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 4 {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0, false
	}
	// Signal 0 doesn't deliver a signal — just probes whether we can.
	// Returns nil if process exists; ESRCH if it doesn't.
	if err := syscall.Kill(pid, 0); err != nil {
		return 0, false
	}
	port, err := strconv.Atoi(strings.TrimSpace(lines[3]))
	if err != nil {
		return 0, false
	}
	return uint32(port), true
}

// clearStaleLock removes postmaster.pid if it points at a dead PID. Without
// this, embedded.Start would refuse with "lock file already exists".
func clearStaleLock(dataDir string) {
	pidFile := filepath.Join(dataDir, "postmaster.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}
	pidStr := strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0])
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		// Garbled file — best to remove and start fresh.
		_ = os.Remove(pidFile)
		return
	}
	if err := syscall.Kill(pid, 0); err != nil {
		// Process is dead. Remove the stale lock.
		_ = os.Remove(pidFile)
		slog.Info("removed stale postgres lock", "pid", pid, "data", dataDir)
	}
}

func findFreePort() (uint32, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint32(l.Addr().(*net.TCPAddr).Port), nil
}
