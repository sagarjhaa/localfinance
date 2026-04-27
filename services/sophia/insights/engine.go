package insights

import (
	"context"
)

// DismissalFetcher loads the set of dismissed insight keys for a user. The real
// implementation (in part B) will hit Thesaurus's
// GET /internal/users/:user_id/dismissed-insights endpoint. For tests, use a
// FakeDismissalFetcher (defined in engine_test.go).
type DismissalFetcher interface {
	ListDismissedKeys(ctx context.Context, userID string) (map[string]struct{}, error)
}

// Engine runs the full set of rules over a RuleContext and filters out anything
// the user has dismissed. It owns no state beyond its rule list and dismissal
// fetcher; safe to share across goroutines.
type Engine struct {
	rules   []Rule
	dismiss DismissalFetcher
}

// NewEngine constructs an engine with all built-in rules registered. Pass nil for
// dismiss to disable dismissal filtering (useful in tests).
func NewEngine(dismiss DismissalFetcher) *Engine {
	return &Engine{
		rules:   AllRules(),
		dismiss: dismiss,
	}
}

// Run executes every rule against rc, concatenates the results, then strips any
// insight whose Key is in the user's dismissed set. Rule execution order matches
// AllRules(); callers should not depend on ordering across rules but may rely on
// the stable order within a single rule's output.
func (e *Engine) Run(ctx context.Context, rc RuleContext) ([]Insight, error) {
	var dismissed map[string]struct{}
	if e.dismiss != nil {
		var err error
		dismissed, err = e.dismiss.ListDismissedKeys(ctx, rc.UserID)
		if err != nil {
			return nil, err
		}
	}

	out := make([]Insight, 0)
	for _, rule := range e.rules {
		for _, ins := range rule(rc) {
			if _, isDismissed := dismissed[ins.Key]; isDismissed {
				continue
			}
			out = append(out, ins)
		}
	}
	return out, nil
}
