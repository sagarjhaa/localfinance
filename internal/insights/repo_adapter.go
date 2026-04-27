package insights

import (
	"context"

	"github.com/sagarjhaa/localfinance/internal/data"
)

// RepoFetcher adapts data.Repositories to the engine's DismissalFetcher
// interface. Used in production wiring; tests still use the existing
// FakeDismissalFetcher in engine_test.go.
type RepoFetcher struct{ Repos data.Repositories }

func (f *RepoFetcher) ListDismissedKeys(ctx context.Context, userID string) (map[string]struct{}, error) {
	return f.Repos.ListDismissedInsightKeys(ctx, userID)
}
