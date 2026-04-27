package monthreview

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sagarjhaa/localfinance/internal/insights"
	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

type fakeFetcher struct {
	txns []models.TransactionRef
	err  error
	last time.Time
}

func (f *fakeFetcher) FetchTransactionsSince(_ string, start time.Time) ([]models.TransactionRef, error) {
	f.last = start
	return f.txns, f.err
}

type fakeEngine struct {
	gotCount int
	output   []insights.Insight
	err      error
}

func (e *fakeEngine) Run(_ context.Context, rc insights.RuleContext) ([]insights.Insight, error) {
	e.gotCount = len(rc.Transactions)
	return e.output, e.err
}

type fakeNarrator struct {
	called bool
	out    insights.MonthInReviewNarrative
	err    error
}

func (n *fakeNarrator) Narrate(_ context.Context, _ []insights.Insight, _ insights.Period) (insights.MonthInReviewNarrative, error) {
	n.called = true
	return n.out, n.err
}

func TestService_Generate_FiltersByPeriod(t *testing.T) {
	fetcher := &fakeFetcher{
		txns: []models.TransactionRef{
			{ID: "in-mid", Date: "2026-04-15", Amount: 10, Category: "Food", Description: "A"},
			{ID: "before", Date: "2026-03-31", Amount: 99, Category: "Food", Description: "Out"},
			{ID: "after", Date: "2026-05-01", Amount: 99, Category: "Food", Description: "Out"},
			{ID: "in-start", Date: "2026-04-01", Amount: 5, Category: "Food", Description: "B"},
		},
	}
	eng := &fakeEngine{output: []insights.Insight{{Key: "k1", RuleID: "x", Title: "X", Description: "y"}}}
	nar := &fakeNarrator{out: insights.MonthInReviewNarrative{
		Overall:    "ok",
		PerInsight: map[string]string{"k1": "polished"},
		Source:     "template",
	}}

	s := NewService(eng, nar, fetcher)
	s.Clock = func() time.Time { return time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC) }

	rev, err := s.Generate(context.Background(), "user-1", Period{Year: 2026, Month: 4})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if eng.gotCount != 2 {
		t.Fatalf("expected 2 in-window txns, got %d", eng.gotCount)
	}
	if rev.Period != "2026-04" || rev.UserID != "user-1" {
		t.Fatalf("review: %+v", rev)
	}
	if rev.Narrative.Source != "template" {
		t.Fatalf("narrative: %+v", rev.Narrative)
	}
	if len(rev.Insights) != 1 || rev.Insights[0].Description != "polished" {
		t.Fatalf("expected narrator description on output insight, got %+v", rev.Insights)
	}
	if !nar.called {
		t.Fatalf("narrator not called")
	}
	if _, ok := s.Cache.Get("user-1", Period{Year: 2026, Month: 4}); !ok {
		t.Fatalf("cache not populated")
	}
}

func TestService_GetOrGenerate_CacheHit(t *testing.T) {
	fetcher := &fakeFetcher{}
	eng := &fakeEngine{}
	nar := &fakeNarrator{}
	s := NewService(eng, nar, fetcher)

	preset := MonthReview{UserID: "u", Period: "2026-04"}
	s.Cache.Set("u", Period{Year: 2026, Month: 4}, preset)

	got, err := s.GetOrGenerate(context.Background(), "u", Period{Year: 2026, Month: 4})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Period != "2026-04" {
		t.Fatalf("got %+v", got)
	}
	if nar.called {
		t.Fatalf("narrator called on cache hit")
	}
}

func TestService_GetOrGenerate_CacheMiss(t *testing.T) {
	fetcher := &fakeFetcher{txns: []models.TransactionRef{{Date: "2026-04-10", Amount: 1}}}
	eng := &fakeEngine{output: []insights.Insight{{Key: "k", RuleID: "r", Title: "T"}}}
	nar := &fakeNarrator{out: insights.MonthInReviewNarrative{Overall: "x", PerInsight: map[string]string{}}}
	s := NewService(eng, nar, fetcher)

	if _, err := s.GetOrGenerate(context.Background(), "u", Period{Year: 2026, Month: 4}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !nar.called {
		t.Fatalf("expected narrator called on miss")
	}
}

func TestService_Generate_PropagatesFetchError(t *testing.T) {
	fetcher := &fakeFetcher{err: errors.New("boom")}
	s := NewService(&fakeEngine{}, &fakeNarrator{}, fetcher)
	if _, err := s.Generate(context.Background(), "u", Period{Year: 2026, Month: 4}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_Generate_PropagatesEngineError(t *testing.T) {
	fetcher := &fakeFetcher{txns: []models.TransactionRef{{Date: "2026-04-10", Amount: 1}}}
	eng := &fakeEngine{err: errors.New("engine boom")}
	s := NewService(eng, &fakeNarrator{}, fetcher)
	if _, err := s.Generate(context.Background(), "u", Period{Year: 2026, Month: 4}); err == nil {
		t.Fatalf("expected error from engine")
	}
}

func TestService_Generate_NarratorErrorIsTolerated(t *testing.T) {
	fetcher := &fakeFetcher{txns: []models.TransactionRef{{Date: "2026-04-10", Amount: 1}}}
	eng := &fakeEngine{output: []insights.Insight{{Key: "k", RuleID: "r", Title: "T"}}}
	nar := &fakeNarrator{err: errors.New("llm down")}
	s := NewService(eng, nar, fetcher)
	rev, err := s.Generate(context.Background(), "u", Period{Year: 2026, Month: 4})
	if err != nil {
		t.Fatalf("expected narrator error to be swallowed, got %v", err)
	}
	if rev.Narrative.Source != "template" {
		t.Fatalf("expected fallback narrative, got %+v", rev.Narrative)
	}
}
