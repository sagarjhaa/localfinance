package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/sagarjhaa/localfinance/internal/data/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB stands up an in-memory SQLite DB so the router can be constructed
// without a live Postgres instance. Only the tables touched by /health-related
// startup are migrated.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserSession{},
		&models.Account{},
		&models.Transaction{},
		&models.DismissedInsight{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestGinRouterHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)
	r := NewGinRouter(db)

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v (body=%q)", err, body)
	}
	if got["status"] != "healthy" {
		t.Fatalf("status field: got %v want healthy", got["status"])
	}
	if got["service"] != "localfinance" {
		t.Fatalf("service field: got %v want localfinance", got["service"])
	}
}
