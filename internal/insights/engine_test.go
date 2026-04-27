package insights

import (
	"context"
	"errors"
	"testing"
)

// FakeDismissalFetcher is an in-memory DismissalFetcher used by engine tests.
type FakeDismissalFetcher struct {
	keysByUser map[string]map[string]struct{}
	err        error
}

func newFakeDismissals() *FakeDismissalFetcher {
	return &FakeDismissalFetcher{keysByUser: map[string]map[string]struct{}{}}
}

func (f *FakeDismissalFetcher) Dismiss(userID, key string) {
	m, ok := f.keysByUser[userID]
	if !ok {
		m = map[string]struct{}{}
		f.keysByUser[userID] = m
	}
	m[key] = struct{}{}
}

func (f *FakeDismissalFetcher) ListDismissedKeys(_ context.Context, userID string) (map[string]struct{}, error) {
	if f.err != nil {
		return nil, f.err
	}
	if m, ok := f.keysByUser[userID]; ok {
		return m, nil
	}
	return map[string]struct{}{}, nil
}

func TestEngine_RunsAllRules(t *testing.T) {
	rc := RuleContext{
		UserID:       "u1",
		Transactions: makeFixtureSet(),
		Now:          fixedNow,
	}
	eng := NewEngine(nil)
	got, err := eng.Run(context.Background(), rc)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	seen := map[string]bool{}
	for _, ins := range got {
		seen[ins.RuleID] = true
	}
	wantRules := []string{
		RuleCategoryShift,
		RuleNewRecurringMerchant,
		RuleDayOfWeekCluster,
		RulePriceEscalation,
		RuleAnomalyCluster,
	}
	for _, r := range wantRules {
		if !seen[r] {
			t.Errorf("expected rule %s to fire on fixture set, but it did not", r)
		}
	}
}

func TestEngine_FiltersDismissedKeys(t *testing.T) {
	rc := RuleContext{
		UserID:       "u1",
		Transactions: makeFixtureSet(),
		Now:          fixedNow,
	}

	// First, run with no dismissals to discover keys.
	eng := NewEngine(nil)
	all, err := eng.Run(context.Background(), rc)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(all) < 2 {
		t.Fatalf("need ≥2 insights to test filtering, got %d", len(all))
	}
	dismissKey := all[0].Key

	dismiss := newFakeDismissals()
	dismiss.Dismiss("u1", dismissKey)

	eng2 := NewEngine(dismiss)
	filtered, err := eng2.Run(context.Background(), rc)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(filtered) != len(all)-1 {
		t.Errorf("expected %d insights after dismissing 1, got %d", len(all)-1, len(filtered))
	}
	for _, ins := range filtered {
		if ins.Key == dismissKey {
			t.Errorf("dismissed key %s leaked through", dismissKey)
		}
	}

	// Other user's dismissals must not affect u1.
	dismiss.Dismiss("u2", all[1].Key)
	again, err := eng2.Run(context.Background(), rc)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(again) != len(filtered) {
		t.Errorf("other user's dismissal affected u1: %d vs %d", len(again), len(filtered))
	}
}

func TestEngine_PropagatesFetcherError(t *testing.T) {
	dismiss := newFakeDismissals()
	dismiss.err = errors.New("thesaurus down")

	eng := NewEngine(dismiss)
	_, err := eng.Run(context.Background(), RuleContext{UserID: "u1", Now: fixedNow})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "thesaurus down" {
		t.Errorf("expected wrapped error, got %v", err)
	}
}
