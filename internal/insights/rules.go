package insights

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

// txDateFormats are the layouts we accept on TransactionRef.Date. Sophia gets dates
// from Thesaurus as strings; in practice we see either bare YYYY-MM-DD or full RFC3339.
// We try both before giving up on a transaction.
var txDateFormats = []string{
	"2006-01-02",
	time.RFC3339,
	time.RFC3339Nano,
}

// parseTxDate returns the parsed timestamp and ok=true when the date is well-formed.
// Bad rows are skipped silently — a single corrupt input shouldn't poison a whole rule.
func parseTxDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range txDateFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// isExpense returns true for transactions that count as outgoing money. We treat
// missing Type as expense (legacy data) but explicit "income"/"credit" as income.
func isExpense(t models.TransactionRef) bool {
	switch strings.ToLower(strings.TrimSpace(t.Type)) {
	case "income", "credit", "deposit", "refund":
		return false
	}
	return true
}

// absAmount normalizes amounts to a positive magnitude so rules don't have to care
// whether the data store uses signed or unsigned conventions.
func absAmount(t models.TransactionRef) float64 {
	if t.Amount < 0 {
		return -t.Amount
	}
	return t.Amount
}

// normalizeMerchant produces a stable key for grouping repeat charges from the same
// merchant. We lowercase and strip common noise (trailing digits often differ per
// charge, e.g. "STARBUCKS #1234"). This is best-effort; perfect merchant resolution
// is out of scope here.
func normalizeMerchant(desc string) string {
	s := strings.ToLower(strings.TrimSpace(desc))
	if s == "" {
		return ""
	}
	// Collapse runs of whitespace
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// AllRules returns every detector registered with the engine. Order is stable so
// engine output is deterministic for tests.
func AllRules() []Rule {
	return []Rule{
		detectCategoryShift,
		detectNewRecurringMerchant,
		detectDayOfWeekCluster,
		detectPriceEscalation,
		detectAnomalyCluster,
	}
}

// ---------------------------------------------------------------------------
// Rule 1: detectCategoryShift
//
// Compare last-30d spending per category vs prior-60d AVERAGE 30d window.
// If shift >20% AND absolute amount >$50, emit. Skip categories with <3
// transactions in either window (too noisy).
// ---------------------------------------------------------------------------

func detectCategoryShift(ctx RuleContext) []Insight {
	now := ctx.Now
	last30Start := now.AddDate(0, 0, -30)
	prior60Start := now.AddDate(0, 0, -90)
	prior60End := last30Start

	type bucket struct {
		total float64
		count int
		ids   []string
	}
	last := map[string]*bucket{}
	prior := map[string]*bucket{}

	for _, t := range ctx.Transactions {
		if !isExpense(t) || t.Category == "" {
			continue
		}
		d, ok := parseTxDate(t.Date)
		if !ok {
			continue
		}
		amt := absAmount(t)
		if d.After(last30Start) || d.Equal(last30Start) {
			if !d.After(now) {
				b := last[t.Category]
				if b == nil {
					b = &bucket{}
					last[t.Category] = b
				}
				b.total += amt
				b.count++
				b.ids = append(b.ids, t.ID)
			}
		} else if (d.After(prior60Start) || d.Equal(prior60Start)) && d.Before(prior60End) {
			b := prior[t.Category]
			if b == nil {
				b = &bucket{}
				prior[t.Category] = b
			}
			b.total += amt
			b.count++
			b.ids = append(b.ids, t.ID)
		}
	}

	var out []Insight
	cats := make([]string, 0, len(last))
	for c := range last {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	for _, cat := range cats {
		l := last[cat]
		p := prior[cat]
		if l == nil || p == nil || l.count < 3 || p.count < 3 {
			continue
		}
		// Prior is 60d, normalize to 30d-equivalent average.
		priorAvg30 := p.total / 2.0
		if priorAvg30 <= 0 {
			continue
		}
		deltaPct := ((l.total - priorAvg30) / priorAvg30) * 100.0
		if math.Abs(deltaPct) < 20.0 {
			continue
		}
		if l.total < 50.0 && priorAvg30 < 50.0 {
			continue
		}
		direction := "up"
		priority := PriorityMedium
		if deltaPct < 0 {
			direction = "down"
			priority = PriorityLow
		} else if deltaPct >= 50 {
			priority = PriorityHigh
		}
		key := MakeKey(RuleCategoryShift, last30Start, now, cat)
		evidence := append([]string{}, l.ids...)
		sort.Strings(evidence)
		title := fmt.Sprintf("%s spending %s %.0f%% vs prior 60d", cat, direction, math.Abs(deltaPct))
		desc := fmt.Sprintf(
			"You spent $%.2f on %s in the last 30 days, vs $%.2f average over the prior 60 days.",
			l.total, cat, priorAvg30,
		)
		out = append(out, Insight{
			Key:         key,
			RuleID:      RuleCategoryShift,
			Title:       title,
			Description: desc,
			Priority:    priority,
			EvidenceIDs: evidence,
			Numbers: map[string]float64{
				"current":    round2(l.total),
				"prior_avg":  round2(priorAvg30),
				"delta_pct":  round2(deltaPct),
				"prior_total": round2(p.total),
			},
			Strings: map[string]string{
				"category":  cat,
				"direction": direction,
			},
			CreatedAt: now,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Rule 2: detectNewRecurringMerchant
//
// Merchant with ≥2 transactions in last 30d, ZERO in prior 60d. One insight
// per merchant. We use normalized description as the merchant key.
// ---------------------------------------------------------------------------

func detectNewRecurringMerchant(ctx RuleContext) []Insight {
	now := ctx.Now
	last30Start := now.AddDate(0, 0, -30)
	prior60Start := now.AddDate(0, 0, -90)

	type m struct {
		display string
		ids     []string
		total   float64
	}
	recent := map[string]*m{}
	priorSeen := map[string]bool{}

	for _, t := range ctx.Transactions {
		if !isExpense(t) {
			continue
		}
		d, ok := parseTxDate(t.Date)
		if !ok {
			continue
		}
		key := normalizeMerchant(t.Description)
		if key == "" {
			continue
		}
		if d.Before(last30Start) {
			if !d.Before(prior60Start) {
				priorSeen[key] = true
			}
			continue
		}
		if d.After(now) {
			continue
		}
		bucket := recent[key]
		if bucket == nil {
			bucket = &m{display: t.Description}
			recent[key] = bucket
		}
		bucket.ids = append(bucket.ids, t.ID)
		bucket.total += absAmount(t)
	}

	var out []Insight
	keys := make([]string, 0, len(recent))
	for k := range recent {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b := recent[k]
		if len(b.ids) < 2 {
			continue
		}
		if priorSeen[k] {
			continue
		}
		evidence := append([]string{}, b.ids...)
		sort.Strings(evidence)
		key := MakeKey(RuleNewRecurringMerchant, last30Start, now, k)
		title := fmt.Sprintf("New recurring charge: %s", b.display)
		desc := fmt.Sprintf(
			"%s charged you %d times totaling $%.2f in the last 30 days, with nothing in the 60 days before that.",
			b.display, len(b.ids), b.total,
		)
		out = append(out, Insight{
			Key:         key,
			RuleID:      RuleNewRecurringMerchant,
			Title:       title,
			Description: desc,
			Priority:    PriorityMedium,
			EvidenceIDs: evidence,
			Numbers: map[string]float64{
				"count": float64(len(b.ids)),
				"total": round2(b.total),
			},
			Strings: map[string]string{
				"merchant":     b.display,
				"merchant_key": k,
			},
			CreatedAt: now,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Rule 3: detectDayOfWeekCluster
//
// For each category with ≥10 transactions in the 90d window, compute spending
// share by day-of-week. If any DOW has >50% of category total AND >$100, emit.
// ---------------------------------------------------------------------------

func detectDayOfWeekCluster(ctx RuleContext) []Insight {
	now := ctx.Now
	periodStart := now.AddDate(0, 0, -90)

	type catData struct {
		total   float64
		count   int
		byDOW   [7]float64
		idsByDOW [7][]string
	}
	cats := map[string]*catData{}
	for _, t := range ctx.Transactions {
		if !isExpense(t) || t.Category == "" {
			continue
		}
		d, ok := parseTxDate(t.Date)
		if !ok || d.Before(periodStart) || d.After(now) {
			continue
		}
		c := cats[t.Category]
		if c == nil {
			c = &catData{}
			cats[t.Category] = c
		}
		amt := absAmount(t)
		c.total += amt
		c.count++
		dow := int(d.Weekday())
		c.byDOW[dow] += amt
		c.idsByDOW[dow] = append(c.idsByDOW[dow], t.ID)
	}

	var out []Insight
	names := make([]string, 0, len(cats))
	for n := range cats {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		c := cats[name]
		if c.count < 10 || c.total <= 0 {
			continue
		}
		for dow := 0; dow < 7; dow++ {
			amt := c.byDOW[dow]
			share := amt / c.total
			if share > 0.50 && amt > 100.0 {
				dowName := time.Weekday(dow).String()
				key := MakeKey(RuleDayOfWeekCluster, periodStart, now, name, dowName)
				evidence := append([]string{}, c.idsByDOW[dow]...)
				sort.Strings(evidence)
				title := fmt.Sprintf("%s spending clusters on %s", name, dowName)
				desc := fmt.Sprintf(
					"%.0f%% of your %s spending ($%.2f of $%.2f) over the last 90 days happened on %ss.",
					share*100, name, amt, c.total, dowName,
				)
				out = append(out, Insight{
					Key:         key,
					RuleID:      RuleDayOfWeekCluster,
					Title:       title,
					Description: desc,
					Priority:    PriorityLow,
					EvidenceIDs: evidence,
					Numbers: map[string]float64{
						"share":          round2(share * 100),
						"dow_total":      round2(amt),
						"category_total": round2(c.total),
					},
					Strings: map[string]string{
						"category": name,
						"dow":      dowName,
					},
					CreatedAt: now,
				})
				// One DOW can dominate; once we found it for a category, move on.
				break
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Rule 4: detectPriceEscalation
//
// Find merchants with ≥3 transactions in the 90d window. If the slope of amount
// over time is positive AND the last charge is >10% higher than the median of
// prior charges, emit.
// ---------------------------------------------------------------------------

func detectPriceEscalation(ctx RuleContext) []Insight {
	now := ctx.Now
	periodStart := now.AddDate(0, 0, -90)

	type charge struct {
		t   time.Time
		amt float64
		id  string
	}
	type group struct {
		display string
		charges []charge
	}
	groups := map[string]*group{}
	for _, t := range ctx.Transactions {
		if !isExpense(t) {
			continue
		}
		d, ok := parseTxDate(t.Date)
		if !ok || d.Before(periodStart) || d.After(now) {
			continue
		}
		k := normalizeMerchant(t.Description)
		if k == "" {
			continue
		}
		g := groups[k]
		if g == nil {
			g = &group{display: t.Description}
			groups[k] = g
		}
		g.charges = append(g.charges, charge{t: d, amt: absAmount(t), id: t.ID})
	}

	var out []Insight
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		g := groups[k]
		if len(g.charges) < 3 {
			continue
		}
		sort.Slice(g.charges, func(i, j int) bool { return g.charges[i].t.Before(g.charges[j].t) })

		// Linear regression slope on (time-as-day-offset, amount). Positive slope is required.
		n := float64(len(g.charges))
		var sumX, sumY, sumXY, sumXX float64
		base := g.charges[0].t
		for _, c := range g.charges {
			x := c.t.Sub(base).Hours() / 24.0
			y := c.amt
			sumX += x
			sumY += y
			sumXY += x * y
			sumXX += x * x
		}
		denom := n*sumXX - sumX*sumX
		if denom == 0 {
			continue
		}
		slope := (n*sumXY - sumX*sumY) / denom
		if slope <= 0 {
			continue
		}

		// Compare last charge to median of priors.
		last := g.charges[len(g.charges)-1]
		prior := make([]float64, len(g.charges)-1)
		for i := 0; i < len(g.charges)-1; i++ {
			prior[i] = g.charges[i].amt
		}
		med := median(prior)
		if med <= 0 {
			continue
		}
		deltaPct := ((last.amt - med) / med) * 100.0
		if deltaPct <= 10.0 {
			continue
		}

		evidence := make([]string, 0, len(g.charges))
		for _, c := range g.charges {
			evidence = append(evidence, c.id)
		}
		sort.Strings(evidence)
		key := MakeKey(RulePriceEscalation, periodStart, now, k)
		title := fmt.Sprintf("%s price up %.0f%%", g.display, deltaPct)
		desc := fmt.Sprintf(
			"Your most recent %s charge was $%.2f, up %.0f%% from the $%.2f median of your prior %d charges.",
			g.display, last.amt, deltaPct, med, len(prior),
		)
		priority := PriorityMedium
		if deltaPct >= 25 {
			priority = PriorityHigh
		}
		out = append(out, Insight{
			Key:         key,
			RuleID:      RulePriceEscalation,
			Title:       title,
			Description: desc,
			Priority:    priority,
			EvidenceIDs: evidence,
			Numbers: map[string]float64{
				"last":       round2(last.amt),
				"median":     round2(med),
				"delta_pct":  round2(deltaPct),
				"slope":      round2(slope),
				"sample_size": float64(len(g.charges)),
			},
			Strings: map[string]string{
				"merchant":     g.display,
				"merchant_key": k,
			},
			CreatedAt: now,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Rule 5: detectAnomalyCluster
//
// Per category with ≥10 transactions, find any 7-day window with ≥3 transactions
// that are anomalous. We define anomalous robustly as |amt - median| > 3*MAD
// (median absolute deviation) — using mean+stddev fails on heavy-tailed data
// because the anomalies contaminate the very statistic we're measuring against.
// We still report the mean and std-dev style fields so the narrator can speak
// in user-friendly "X above your category mean" language.
// ---------------------------------------------------------------------------

func detectAnomalyCluster(ctx RuleContext) []Insight {
	now := ctx.Now
	periodStart := now.AddDate(0, 0, -90)

	type tx struct {
		t   time.Time
		amt float64
		id  string
	}
	cats := map[string][]tx{}
	for _, t := range ctx.Transactions {
		if !isExpense(t) || t.Category == "" {
			continue
		}
		d, ok := parseTxDate(t.Date)
		if !ok || d.Before(periodStart) || d.After(now) {
			continue
		}
		cats[t.Category] = append(cats[t.Category], tx{t: d, amt: absAmount(t), id: t.ID})
	}

	var out []Insight
	names := make([]string, 0, len(cats))
	for n := range cats {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		txs := cats[name]
		if len(txs) < 10 {
			continue
		}
		amts := make([]float64, len(txs))
		for i, t := range txs {
			amts[i] = t.amt
		}
		mean, sd := meanStddev(amts)
		med := median(amts)
		mad := medianAbsoluteDeviation(amts, med)
		if mad <= 0 {
			// Degenerate case (>50% of charges identical). Fall back to a simple
			// "10x median" rule so we still surface egregious spikes.
			if med <= 0 {
				continue
			}
			mad = med // makes threshold = 4*median below
		}
		threshold := med + 3*mad
		// Tag anomalies and sort chronologically.
		sort.Slice(txs, func(i, j int) bool { return txs[i].t.Before(txs[j].t) })
		anomalies := make([]tx, 0)
		for _, t := range txs {
			if t.amt > threshold {
				anomalies = append(anomalies, t)
			}
		}
		if len(anomalies) < 3 {
			continue
		}
		// Sliding 7-day window over anomalies.
		for i := 0; i < len(anomalies); i++ {
			j := i
			for j < len(anomalies) && anomalies[j].t.Sub(anomalies[i].t) <= 7*24*time.Hour {
				j++
			}
			if j-i >= 3 {
				cluster := anomalies[i:j]
				ids := make([]string, 0, len(cluster))
				var total float64
				for _, c := range cluster {
					ids = append(ids, c.id)
					total += c.amt
				}
				sort.Strings(ids)
				wStart := QuantizeDay(cluster[0].t)
				wEnd := QuantizeDay(cluster[len(cluster)-1].t)
				key := MakeKey(RuleAnomalyCluster, wStart, wEnd, name)
				title := fmt.Sprintf("Unusual %s spending cluster", name)
				desc := fmt.Sprintf(
					"%d %s transactions totaling $%.2f happened between %s and %s — each more than 3σ above your category mean of $%.2f.",
					len(cluster), name, total,
					wStart.Format("Jan 2"), wEnd.Format("Jan 2"), mean,
				)
				out = append(out, Insight{
					Key:         key,
					RuleID:      RuleAnomalyCluster,
					Title:       title,
					Description: desc,
					Priority:    PriorityHigh,
					EvidenceIDs: ids,
					Numbers: map[string]float64{
						"count":      float64(len(cluster)),
						"total":      round2(total),
						"mean":       round2(mean),
						"stddev":     round2(sd),
						"threshold":  round2(threshold),
					},
					Strings: map[string]string{
						"category":     name,
						"window_start": wStart.Format("2006-01-02"),
						"window_end":   wEnd.Format("2006-01-02"),
					},
					CreatedAt: now,
				})
				break // one cluster per category
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func meanStddev(xs []float64) (float64, float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(len(xs))
	var sq float64
	for _, x := range xs {
		d := x - mean
		sq += d * d
	}
	variance := sq / float64(len(xs))
	return mean, math.Sqrt(variance)
}

// medianAbsoluteDeviation returns MAD = median(|x_i - med|).
func medianAbsoluteDeviation(xs []float64, med float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	devs := make([]float64, len(xs))
	for i, x := range xs {
		d := x - med
		if d < 0 {
			d = -d
		}
		devs[i] = d
	}
	return median(devs)
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := make([]float64, len(xs))
	copy(cp, xs)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2.0
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
