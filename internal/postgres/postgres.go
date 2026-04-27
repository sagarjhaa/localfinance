// Package postgres manages the embedded Postgres lifecycle for the .app
// build path. In dev (DB_HOST set), the binary connects to a host Postgres
// and this package is unused.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	embedded "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

// Manager owns the lifecycle of an embedded Postgres instance.
type Manager struct {
	pg      *embedded.EmbeddedPostgres
	port    uint32
	dataDir string
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

// Start launches the embedded Postgres process.
func (m *Manager) Start(_ context.Context) error { return m.pg.Start() }

// Stop terminates the embedded Postgres process.
func (m *Manager) Stop(_ context.Context) error { return m.pg.Stop() }

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

func findFreePort() (uint32, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint32(l.Addr().(*net.TCPAddr).Port), nil
}
