package insights

import (
	"strings"
	"testing"

	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

func runRule(rule Rule, txs []models.TransactionRef) []Insight {
	return rule(RuleContext{
		UserID:       "u1",
		Transactions: txs,
		Now:          fixedNow,
	})
}

// ---------- detectCategoryShift ----------

func TestDetectCategoryShift(t *testing.T) {
	cases := []struct {
		name     string
		txs      []models.TransactionRef
		wantHits int
		wantSub  string // substring expected in title
	}{
		{
			name:     "empty",
			txs:      nil,
			wantHits: 0,
		},
		{
			name: "below threshold (<20% delta)",
			txs: []models.TransactionRef{
				tx("a1", 10, "X", "Groceries", 100), tx("a2", 12, "X", "Groceries", 100), tx("a3", 14, "X", "Groceries", 100),
				tx("b1", 35, "X", "Groceries", 100), tx("b2", 50, "X", "Groceries", 100), tx("b3", 75, "X", "Groceries", 100),
				tx("b4", 80, "X", "Groceries", 100), tx("b5", 88, "X", "Groceries", 100), tx("b6", 89, "X", "Groceries", 100),
			},
			wantHits: 0,
		},
		{
			name: "skip when too few transactions",
			txs: []models.TransactionRef{
				tx("a1", 5, "X", "Coffee", 100), tx("a2", 6, "X", "Coffee", 100),
				tx("b1", 40, "X", "Coffee", 50),
			},
			wantHits: 0,
		},
		{
			name: "increase >20% triggers",
			txs: []models.TransactionRef{
				// last 30d: 4 x $100 = $400
				tx("a1", 5, "X", "Dining", 100), tx("a2", 8, "X", "Dining", 100),
				tx("a3", 15, "X", "Dining", 100), tx("a4", 20, "X", "Dining", 100),
				// prior 60d (31..89d ago): 4 x $50 = $200 -> 30d-equiv $100
				tx("b1", 35, "X", "Dining", 50), tx("b2", 50, "X", "Dining", 50),
				tx("b3", 70, "X", "Dining", 50), tx("b4", 85, "X", "Dining", 50),
			},
			wantHits: 1,
			wantSub:  "up",
		},
		{
			name: "decrease >20% triggers",
			txs: []models.TransactionRef{
				// last 30d: 3 x $20 = $60
				tx("a1", 5, "X", "Coffee", 20), tx("a2", 10, "X", "Coffee", 20), tx("a3", 20, "X", "Coffee", 20),
				// prior 60d: 6 x $50 = $300 -> 30d-equiv $150
				tx("b1", 35, "X", "Coffee", 50), tx("b2", 45, "X", "Coffee", 50), tx("b3", 55, "X", "Coffee", 50),
				tx("b4", 65, "X", "Coffee", 50), tx("b5", 75, "X", "Coffee", 50), tx("b6", 85, "X", "Coffee", 50),
			},
			wantHits: 1,
			wantSub:  "down",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runRule(detectCategoryShift, tc.txs)
			if len(got) != tc.wantHits {
				t.Fatalf("want %d, got %d: %+v", tc.wantHits, len(got), got)
			}
			if tc.wantSub != "" && !strings.Contains(got[0].Title, tc.wantSub) {
				t.Errorf("title %q missing %q", got[0].Title, tc.wantSub)
			}
			for _, ins := range got {
				if len(ins.EvidenceIDs) == 0 {
					t.Errorf("expected evidence ids")
				}
				if ins.Numbers["current"] == 0 && ins.Numbers["prior_avg"] == 0 {
					t.Errorf("expected numbers populated")
				}
			}
		})
	}
}

// ---------- detectNewRecurringMerchant ----------

func TestDetectNewRecurringMerchant(t *testing.T) {
	cases := []struct {
		name     string
		txs      []models.TransactionRef
		wantHits int
	}{
		{
			name:     "empty",
			txs:      nil,
			wantHits: 0,
		},
		{
			name: "single charge in 30d does not qualify",
			txs: []models.TransactionRef{
				tx("g1", 5, "Netflix", "Entertainment", 15),
			},
			wantHits: 0,
		},
		{
			name: "merchant existed before",
			txs: []models.TransactionRef{
				tx("g1", 5, "Netflix", "Entertainment", 15),
				tx("g2", 25, "Netflix", "Entertainment", 15),
				tx("g0", 50, "Netflix", "Entertainment", 15),
			},
			wantHits: 0,
		},
		{
			name: "two new charges -> hit",
			txs: []models.TransactionRef{
				tx("g1", 5, "ChatGPT Plus", "Software", 20),
				tx("g2", 25, "ChatGPT Plus", "Software", 20),
			},
			wantHits: 1,
		},
		{
			name: "two distinct new merchants -> two hits",
			txs: []models.TransactionRef{
				tx("a1", 5, "ServiceA", "Software", 10), tx("a2", 20, "ServiceA", "Software", 10),
				tx("b1", 7, "ServiceB", "Software", 10), tx("b2", 22, "ServiceB", "Software", 10),
			},
			wantHits: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runRule(detectNewRecurringMerchant, tc.txs)
			if len(got) != tc.wantHits {
				t.Fatalf("want %d hits, got %d: %+v", tc.wantHits, len(got), got)
			}
		})
	}
}

// ---------- detectDayOfWeekCluster ----------

func TestDetectDayOfWeekCluster(t *testing.T) {
	// Helper: Saturday is 4 days before fixedNow (Wed). So daysAgo of Saturday = 4, 11, 18, 25...
	saturdays := func(n int, amt float64, prefix, cat string) []models.TransactionRef {
		out := []models.TransactionRef{}
		for i := 0; i < n; i++ {
			out = append(out, tx(prefix+itoa(i), 4+i*7, prefix, cat, amt))
		}
		return out
	}

	cases := []struct {
		name     string
		txs      []models.TransactionRef
		wantHits int
	}{
		{
			name:     "empty",
			txs:      nil,
			wantHits: 0,
		},
		{
			name: "below count threshold",
			txs: append(saturdays(5, 50, "Cafe", "Coffee"),
				tx("o1", 9, "Other", "Coffee", 5)),
			wantHits: 0,
		},
		{
			name: "evenly spread categories do not trigger",
			txs: func() []models.TransactionRef {
				var out []models.TransactionRef
				for i := 0; i < 12; i++ {
					out = append(out, tx("e"+itoa(i), 5+i*5, "Spread", "Coffee", 30))
				}
				return out
			}(),
			wantHits: 0,
		},
		{
			name: "saturday cluster triggers",
			txs: append(saturdays(11, 50, "WeekendCafe", "Coffee"),
				tx("o1", 9, "Other", "Coffee", 5)),
			wantHits: 1,
		},
		{
			name: "DOW dominant but under $100 absolute -> skip",
			txs: append(saturdays(11, 5, "TinyCafe", "Coffee"),
				tx("o1", 9, "Other", "Coffee", 1)),
			wantHits: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runRule(detectDayOfWeekCluster, tc.txs)
			if len(got) != tc.wantHits {
				t.Fatalf("want %d hits, got %d: %+v", tc.wantHits, len(got), got)
			}
			if tc.wantHits > 0 {
				if got[0].Strings["dow"] != "Saturday" {
					t.Errorf("expected Saturday cluster, got %s", got[0].Strings["dow"])
				}
			}
		})
	}
}

// ---------- detectPriceEscalation ----------

func TestDetectPriceEscalation(t *testing.T) {
	cases := []struct {
		name     string
		txs      []models.TransactionRef
		wantHits int
	}{
		{
			name:     "empty",
			txs:      nil,
			wantHits: 0,
		},
		{
			name: "fewer than 3 charges",
			txs: []models.TransactionRef{
				tx("a", 50, "Gym", "Fitness", 30), tx("b", 20, "Gym", "Fitness", 35),
			},
			wantHits: 0,
		},
		{
			name: "flat price -> no hit",
			txs: []models.TransactionRef{
				tx("a", 70, "Gym", "Fitness", 30), tx("b", 40, "Gym", "Fitness", 30), tx("c", 10, "Gym", "Fitness", 30),
			},
			wantHits: 0,
		},
		{
			name: "rising price triggers",
			txs: []models.TransactionRef{
				tx("a", 80, "Gym", "Fitness", 30),
				tx("b", 50, "Gym", "Fitness", 35),
				tx("c", 20, "Gym", "Fitness", 50),
			},
			wantHits: 1,
		},
		{
			name: "last is barely 5% above median -> no hit",
			txs: []models.TransactionRef{
				tx("a", 80, "Gym", "Fitness", 30),
				tx("b", 50, "Gym", "Fitness", 31),
				tx("c", 20, "Gym", "Fitness", 31.5),
			},
			wantHits: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runRule(detectPriceEscalation, tc.txs)
			if len(got) != tc.wantHits {
				t.Fatalf("want %d hits, got %d: %+v", tc.wantHits, len(got), got)
			}
		})
	}
}

// ---------- detectAnomalyCluster ----------

func TestDetectAnomalyCluster(t *testing.T) {
	baseline := func() []models.TransactionRef {
		out := []models.TransactionRef{}
		for i := 0; i < 10; i++ {
			out = append(out, tx("base"+itoa(i), 40+i*5, "Bus", "Travel", 5))
		}
		return out
	}

	cases := []struct {
		name     string
		txs      []models.TransactionRef
		wantHits int
	}{
		{
			name:     "empty",
			txs:      nil,
			wantHits: 0,
		},
		{
			name:     "baseline only",
			txs:      baseline(),
			wantHits: 0,
		},
		{
			name: "two anomalies (need 3) -> no hit",
			txs: append(baseline(),
				tx("a1", 15, "Flight", "Travel", 2000),
				tx("a2", 14, "Hotel", "Travel", 1800),
			),
			wantHits: 0,
		},
		{
			name: "three anomalies in same week -> hit",
			txs: append(baseline(),
				tx("a1", 15, "Flight", "Travel", 2000),
				tx("a2", 14, "Hotel", "Travel", 1800),
				tx("a3", 12, "Yacht", "Travel", 2500),
			),
			wantHits: 1,
		},
		{
			name: "three anomalies spread over month -> no cluster",
			txs: append(baseline(),
				tx("a1", 25, "Flight", "Travel", 2000),
				tx("a2", 15, "Hotel", "Travel", 1800),
				tx("a3", 5, "Yacht", "Travel", 2500),
			),
			wantHits: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runRule(detectAnomalyCluster, tc.txs)
			if len(got) != tc.wantHits {
				t.Fatalf("want %d hits, got %d: %+v", tc.wantHits, len(got), got)
			}
			if tc.wantHits > 0 {
				if got[0].Priority != PriorityHigh {
					t.Errorf("anomaly cluster should be high priority, got %s", got[0].Priority)
				}
				if len(got[0].EvidenceIDs) < 3 {
					t.Errorf("expected ≥3 evidence ids, got %d", len(got[0].EvidenceIDs))
				}
			}
		})
	}
}

// ---------- conversion ----------

func TestToFinancialInsight(t *testing.T) {
	in := Insight{
		Key:         "abc",
		RuleID:      RuleCategoryShift,
		Title:       "T",
		Description: "D",
		Priority:    PriorityHigh,
		CreatedAt:   fixedNow,
	}
	out := in.ToFinancialInsight()
	if out.Type != RuleCategoryShift || out.Title != "T" || out.Priority != PriorityHigh {
		t.Errorf("unexpected conversion: %+v", out)
	}
}

// itoa is a tiny helper to keep the test fixtures terse.
func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	var b [20]byte
	bp := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		bp--
		b[bp] = digits[i%10]
		i /= 10
	}
	if neg {
		bp--
		b[bp] = '-'
	}
	return string(b[bp:])
}
