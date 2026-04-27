package monthreview

import (
	"context"
	"fmt"
	"time"

	"github.com/sagarjhaa/localfinance/internal/insights"
	models "github.com/sagarjhaa/localfinance/internal/ai"
)

// engineRunner is the slice of *insights.Engine the service depends on.
// Defined as an interface so tests can substitute a fake without spinning up
// the full deterministic engine.
type engineRunner interface {
	Run(ctx context.Context, rc insights.RuleContext) ([]insights.Insight, error)
}

// narratorRunner is the slice of insights.Narrator the service depends on.
// Identical signature; aliased so the Service struct field is named clearly.
type narratorRunner interface {
	Narrate(ctx context.Context, ins []insights.Insight, period insights.Period) (insights.MonthInReviewNarrative, error)
}

// TransactionFetcher matches the interface insights_handler uses to pull data
// from Thesaurus. Redeclared here so callers don't need to import the handlers
// package just to satisfy the Service struct.
type TransactionFetcher interface {
	FetchTransactionsSince(userID string, start time.Time) ([]models.TransactionRef, error)
}

// Service generates month-review summaries for a (user, period) tuple.
// All dependencies are injected so tests can substitute fakes.
type Service struct {
	Engine       engineRunner
	Narrator     narratorRunner
	Transactions TransactionFetcher
	Cache        Cache
	Clock        func() time.Time
}

// NewService returns a Service with sane defaults (NewMemoryCache, time.Now).
func NewService(engine engineRunner, narrator narratorRunner, fetcher TransactionFetcher) *Service {
	return &Service{
		Engine:       engine,
		Narrator:     narrator,
		Transactions: fetcher,
		Cache:        NewMemoryCache(),
		Clock:        func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock()
	}
	return time.Now().UTC()
}

// Generate runs the engine + narrator for `period`, stores the result in the
// cache (overwriting any prior entry), and returns it.
//
// Transactions are fetched starting at the period's start, then filtered to the
// [start, end) window so transactions outside the period (e.g. fetcher
// returning a wider range) are excluded before the engine sees them.
func (s *Service) Generate(ctx context.Context, userID string, period Period) (MonthReview, error) {
	if userID == "" {
		return MonthReview{}, fmt.Errorf("user_id is required")
	}
	start, end := period.StartEnd()

	all, err := s.Transactions.FetchTransactionsSince(userID, start)
	if err != nil {
		return MonthReview{}, fmt.Errorf("fetch transactions: %w", err)
	}

	filtered := filterByWindow(all, start, end)

	rc := insights.RuleContext{
		UserID:       userID,
		Transactions: filtered,
		Now:          s.now(),
	}
	engineInsights, err := s.Engine.Run(ctx, rc)
	if err != nil {
		return MonthReview{}, fmt.Errorf("run engine: %w", err)
	}

	narrative, err := s.Narrator.Narrate(ctx, engineInsights, insights.Period{
		Start: start,
		End:   end,
		Label: period.String(),
	})
	if err != nil {
		// Narrator must never hard-fail the request; degrade to empty narrative.
		narrative = insights.MonthInReviewNarrative{
			PerInsight: map[string]string{},
			Source:     "template",
		}
	}

	publicInsights := make([]models.FinancialInsight, 0, len(engineInsights))
	for _, ins := range engineInsights {
		fi := ins.ToFinancialInsight()
		// Prefer narrator prose over the rule's raw description so UI gets
		// uniform, polished copy.
		if narrText, ok := narrative.PerInsight[ins.Key]; ok && narrText != "" {
			fi.Description = narrText
		}
		publicInsights = append(publicInsights, fi)
	}

	review := MonthReview{
		UserID:      userID,
		Period:      period.String(),
		GeneratedAt: s.now(),
		Insights:    publicInsights,
		Narrative:   narrative,
	}
	s.Cache.Set(userID, period, review)
	return review, nil
}

// GetOrGenerate returns a cached review, falling back to Generate if missing.
func (s *Service) GetOrGenerate(ctx context.Context, userID string, period Period) (MonthReview, error) {
	if v, ok := s.Cache.Get(userID, period); ok {
		return v, nil
	}
	return s.Generate(ctx, userID, period)
}

// Invalidate evicts the cache entry for (user, period) so the next read
// triggers a re-generation.
func (s *Service) Invalidate(userID string, period Period) {
	s.Cache.Invalidate(userID, period)
}

// filterByWindow returns transactions whose Date (YYYY-MM-DD or RFC3339-prefixed)
// falls in [start, end). Transactions with unparseable dates are dropped — the
// engine's rules require valid dates anyway, so silently keeping them creates
// inconsistent behavior across rules.
func filterByWindow(in []models.TransactionRef, start, end time.Time) []models.TransactionRef {
	out := make([]models.TransactionRef, 0, len(in))
	for _, tx := range in {
		ts := tx.Date
		if len(ts) >= 10 {
			ts = ts[:10]
		}
		t, err := time.Parse("2006-01-02", ts)
		if err != nil {
			continue
		}
		t = t.UTC()
		if (t.Equal(start) || t.After(start)) && t.Before(end) {
			out = append(out, tx)
		}
	}
	return out
}
