package internalapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.DismissedInsight{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	h := NewDismissedInsightsHandler(db)
	r := gin.New()
	r.POST("/internal/users/:user_id/dismissed-insights", h.Create)
	r.GET("/internal/users/:user_id/dismissed-insights", h.List)
	r.DELETE("/internal/users/:user_id/dismissed-insights/:insight_key", h.Delete)
	return r, db
}

func TestDismissedInsightsHandler_PostGetDeleteFlow(t *testing.T) {
	r, _ := newTestRouter(t)
	uid := uuid.New().String()

	// POST first dismissal → 201
	body, _ := json.Marshal(map[string]string{"insight_key": "k-abc", "rule_id": "rule.high_spend"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		"/internal/users/"+uid+"/dismissed-insights",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on first POST, got %d: %s", w.Code, w.Body.String())
	}
	var created models.DismissedInsight
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.InsightKey != "k-abc" {
		t.Fatalf("unexpected insight_key %q", created.InsightKey)
	}

	// POST duplicate → 200 with same row
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost,
		"/internal/users/"+uid+"/dismissed-insights",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on duplicate POST, got %d: %s", w.Code, w.Body.String())
	}
	var dup models.DismissedInsight
	_ = json.Unmarshal(w.Body.Bytes(), &dup)
	if dup.ID != created.ID {
		t.Fatalf("duplicate POST returned different ID: %s vs %s", dup.ID, created.ID)
	}

	// GET → list of 1
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/internal/users/"+uid+"/dismissed-insights", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET, got %d", w.Code)
	}
	var list []models.DismissedInsight
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 row in list, got %d", len(list))
	}

	// DELETE → 200
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete,
		"/internal/users/"+uid+"/dismissed-insights/k-abc", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on DELETE, got %d", w.Code)
	}

	// DELETE again → 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete,
		"/internal/users/"+uid+"/dismissed-insights/k-abc", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on second DELETE, got %d", w.Code)
	}

	// GET → empty
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/internal/users/"+uid+"/dismissed-insights", nil)
	r.ServeHTTP(w, req)
	var emptyList []models.DismissedInsight
	_ = json.Unmarshal(w.Body.Bytes(), &emptyList)
	if len(emptyList) != 0 {
		t.Fatalf("expected empty list after delete, got %d", len(emptyList))
	}
}

func TestDismissedInsightsHandler_BadUserID(t *testing.T) {
	r, _ := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/internal/users/not-a-uuid/dismissed-insights", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad uuid, got %d", w.Code)
	}
}
