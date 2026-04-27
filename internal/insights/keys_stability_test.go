package insights

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestKeyStability_AcrossRuns confirms running every rule twice over the same
// fixture produces the exact same set of Insight.Key values. If this fails,
// dismissals will resurrect after recomputation — a critical bug.
func TestKeyStability_AcrossRuns(t *testing.T) {
	rc := RuleContext{
		UserID:       "user-1",
		Transactions: makeFixtureSet(),
		Now:          fixedNow,
	}

	keysA := collectKeys(t, rc)
	keysB := collectKeys(t, rc)

	if len(keysA) == 0 {
		t.Fatalf("fixture produced zero insights — fixture is not exercising the rules")
	}
	if len(keysA) != len(keysB) {
		t.Fatalf("run A produced %d keys, run B produced %d", len(keysA), len(keysB))
	}
	for i := range keysA {
		if keysA[i] != keysB[i] {
			t.Errorf("key mismatch at index %d: %s vs %s", i, keysA[i], keysB[i])
		}
	}
}

// TestKeyStability_FunctionPure verifies MakeKey is order-stable and deterministic
// for the same inputs but DIFFERENT for different inputs.
func TestKeyStability_FunctionPure(t *testing.T) {
	a := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	b := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	k1 := MakeKey("rule_x", a, b, "Dining", "Coffee")
	k2 := MakeKey("rule_x", a, b, "Dining", "Coffee")
	k3 := MakeKey("rule_x", a, b, "Coffee", "Dining") // different order
	k4 := MakeKey("rule_y", a, b, "Dining", "Coffee") // different rule

	if k1 != k2 {
		t.Errorf("identical inputs produced different keys: %s != %s", k1, k2)
	}
	if k1 == k3 {
		t.Errorf("entity order should affect key: got same key %s", k1)
	}
	if k1 == k4 {
		t.Errorf("rule id should affect key")
	}
	if len(k1) != 64 {
		t.Errorf("expected 64-char hex digest, got %d", len(k1))
	}
}

// TestKeyStability_DayQuantize ensures different times within the same UTC day
// quantize to the same value, but different days do not.
func TestKeyStability_DayQuantize(t *testing.T) {
	morning := time.Date(2026, 4, 22, 1, 30, 0, 0, time.UTC)
	evening := time.Date(2026, 4, 22, 23, 59, 0, 0, time.UTC)
	nextDay := time.Date(2026, 4, 23, 0, 0, 1, 0, time.UTC)

	if !QuantizeDay(morning).Equal(QuantizeDay(evening)) {
		t.Errorf("same-day times should quantize equal")
	}
	if QuantizeDay(morning).Equal(QuantizeDay(nextDay)) {
		t.Errorf("different-day times should not quantize equal")
	}
}

func collectKeys(t *testing.T, rc RuleContext) []string {
	t.Helper()
	eng := NewEngine(nil)
	out, err := eng.Run(context.Background(), rc)
	if err != nil {
		t.Fatalf("engine.Run: %v", err)
	}
	keys := make([]string, 0, len(out))
	for _, ins := range out {
		if ins.Key == "" {
			t.Errorf("rule %s produced insight with empty key: %s", ins.RuleID, ins.Title)
		}
		if strings.TrimSpace(ins.Title) == "" {
			t.Errorf("rule %s produced insight with empty title", ins.RuleID)
		}
		keys = append(keys, ins.Key)
	}
	sort.Strings(keys)
	return keys
}
