package insights

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// keyTimeFormat is the canonical date stamp baked into Insight.Key. Day precision
// is intentional: we want dismissals to stick across re-runs on the same day, but
// re-emit if the same finding recurs in a different period.
const keyTimeFormat = "2006-01-02"

// MakeKey produces a deterministic SHA-256 hex digest used to identify an insight
// across recomputation runs. The same logical finding must produce the same key
// every time — otherwise dismissals resurrect.
//
// The key inputs are:
//   - ruleID: the stable Rule* constant
//   - periodStart, periodEnd: quantized to day (see QuantizeDay)
//   - entities: variable parts (merchant, category, etc.). Caller is responsible
//     for passing them in a fixed order per rule. They are lowercased and trimmed
//     but NOT sorted (order encodes meaning, e.g. "category" vs "merchant").
func MakeKey(ruleID string, periodStart, periodEnd time.Time, entities ...string) string {
	parts := make([]string, 0, 3+len(entities))
	parts = append(parts, ruleID)
	parts = append(parts, QuantizeDay(periodStart).UTC().Format(keyTimeFormat))
	parts = append(parts, QuantizeDay(periodEnd).UTC().Format(keyTimeFormat))
	for _, e := range entities {
		parts = append(parts, strings.ToLower(strings.TrimSpace(e)))
	}
	joined := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:])
}

// QuantizeDay zeroes the time component (in UTC) so insights bucket cleanly by day.
// Using UTC avoids surprises when the engine runs across DST transitions.
func QuantizeDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

