package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/ai"
	"github.com/sagarjhaa/localfinance/internal/insights"
	"github.com/sagarjhaa/localfinance/internal/monthreview"
)

type stubFetcher struct{ txns []ai.TransactionRef }

func (s *stubFetcher) FetchTransactionsSince(_ string, _ time.Time) ([]ai.TransactionRef, error) {
	return s.txns, nil
}

type stubEngine struct{}

func (stubEngine) Run(_ context.Context, _ insights.RuleContext) ([]insights.Insight, error) {
	return []insights.Insight{{Key: "k", RuleID: "test", Title: "t", Description: "d"}}, nil
}

type stubNarrator struct{}

func (stubNarrator) Narrate(_ context.Context, _ []insights.Insight, _ insights.Period) (insights.MonthInReviewNarrative, error) {
	return insights.MonthInReviewNarrative{
		Overall:    "ok",
		PerInsight: map[string]string{},
		Source:     "stub",
	}, nil
}

func newMonthReviewTestRouter() (*gin.Engine, *monthreview.Service) {
	gin.SetMode(gin.TestMode)
	svc := &monthreview.Service{
		Engine:       stubEngine{},
		Narrator:     stubNarrator{},
		Transactions: &stubFetcher{txns: []ai.TransactionRef{{Date: "2026-04-12", Amount: 10}}},
		Cache:        monthreview.NewMemoryCache(),
		Clock:        func() time.Time { return time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC) },
	}
	// Periods endpoint isn't exercised by these tests so a nil db is fine.
	h := NewMonthReviewHandler(svc, nil)
	r := gin.New()
	r.POST("/api/v1/internal/month-review/generate", h.Generate)
	r.GET("/api/v1/month-review/:period", h.Get)
	r.DELETE("/api/v1/month-review/:period", h.Delete)
	return r, svc
}

func TestMonthReview_InternalGenerate(t *testing.T) {
	r, svc := newMonthReviewTestRouter()

	body, _ := json.Marshal(map[string]string{"user_id": "u-1", "period": "2026-04"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/month-review/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var rev monthreview.MonthReview
	if err := json.Unmarshal(w.Body.Bytes(), &rev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rev.Period != "2026-04" || rev.UserID != "u-1" {
		t.Fatalf("review: %+v", rev)
	}
	if _, ok := svc.Cache.Get("u-1", monthreview.Period{Year: 2026, Month: 4}); !ok {
		t.Fatalf("cache not populated")
	}
}

func TestMonthReview_PublicGet_QueryUserID(t *testing.T) {
	r, _ := newMonthReviewTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/month-review/2026-04?user_id=u-2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestMonthReview_PublicGet_BadPeriod(t *testing.T) {
	r, _ := newMonthReviewTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/month-review/bogus?user_id=u", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestMonthReview_PublicGet_MissingUser(t *testing.T) {
	r, _ := newMonthReviewTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/month-review/2026-04", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestMonthReview_DeleteInvalidates(t *testing.T) {
	r, svc := newMonthReviewTestRouter()
	svc.Cache.Set("u-3", monthreview.Period{Year: 2026, Month: 4}, monthreview.MonthReview{UserID: "u-3", Period: "2026-04"})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/month-review/2026-04?user_id=u-3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if _, ok := svc.Cache.Get("u-3", monthreview.Period{Year: 2026, Month: 4}); ok {
		t.Fatalf("cache not invalidated")
	}
}
