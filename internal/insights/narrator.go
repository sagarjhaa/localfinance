package insights

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sagarjhaa/localfinance/internal/prompts"
)

// Period describes the analysis window the narrator is summarizing. Used only
// for the opening paragraph so the LLM (and the template) can say "the last 90
// days" or "March 2026" naturally.
type Period struct {
	Start time.Time
	End   time.Time
	Label string // optional human label, e.g. "April 2026". If empty, falls back to Start..End.
}

func (p Period) describe() string {
	if p.Label != "" {
		return p.Label
	}
	if p.Start.IsZero() || p.End.IsZero() {
		return "the recent period"
	}
	return fmt.Sprintf("%s to %s", p.Start.Format("Jan 2"), p.End.Format("Jan 2, 2006"))
}

// MonthInReviewNarrative is the public output of the narrator. It carries the
// pre-rendered prose; the handler merges it into the API response alongside the
// structured FinancialInsight list.
type MonthInReviewNarrative struct {
	Overall    string            `json:"overall"`
	PerInsight map[string]string `json:"per_insight"` // keyed by Insight.Key
	Source     string            `json:"source"`      // "template" or "llm"
}

// Narrator turns structured insights into prose. Implementations must never
// invent numbers, merchant names, dates, or categories beyond what the
// underlying Insight already contains.
type Narrator interface {
	Narrate(ctx context.Context, insights []Insight, period Period) (MonthInReviewNarrative, error)
}

// ---------------------------------------------------------------------------
// Phase A — deterministic template renderer (no LLM)
// ---------------------------------------------------------------------------

type templateNarrator struct{}

// NewTemplateNarrator returns a Narrator that only uses Phase A templating.
// Always safe; never calls an LLM.
func NewTemplateNarrator() Narrator { return &templateNarrator{} }

func (n *templateNarrator) Narrate(_ context.Context, insights []Insight, period Period) (MonthInReviewNarrative, error) {
	out := MonthInReviewNarrative{
		PerInsight: make(map[string]string, len(insights)),
		Source:     "template",
	}
	out.Overall = renderOverall(insights, period)
	for _, ins := range insights {
		out.PerInsight[ins.Key] = renderInsight(ins)
	}
	return out, nil
}

// renderOverall produces a 1-2 sentence intro paragraph. Deterministic.
func renderOverall(insights []Insight, period Period) string {
	if len(insights) == 0 {
		return fmt.Sprintf("Nothing notable to flag for %s — your spending looks steady.", period.describe())
	}
	highs, meds, lows := 0, 0, 0
	for _, ins := range insights {
		switch ins.Priority {
		case PriorityHigh:
			highs++
		case PriorityMedium:
			meds++
		default:
			lows++
		}
	}
	parts := []string{}
	if highs > 0 {
		parts = append(parts, fmt.Sprintf("%d high-priority", highs))
	}
	if meds > 0 {
		parts = append(parts, fmt.Sprintf("%d medium-priority", meds))
	}
	if lows > 0 {
		parts = append(parts, fmt.Sprintf("%d low-priority", lows))
	}
	return fmt.Sprintf(
		"Here's what stood out for %s: %d insight(s) — %s.",
		period.describe(), len(insights), strings.Join(parts, ", "),
	)
}

// renderInsight produces a single deterministic sentence per insight, switching
// on RuleID. Each rule's sentence references only fields known to be present in
// that rule's Numbers/Strings map (see rules.go).
func renderInsight(ins Insight) string {
	switch ins.RuleID {
	case RuleMonthSummary:
		count := int(ins.Numbers["txn_count"])
		total := ins.Numbers["total_spend"]
		topCat := ins.Strings["top_category"]
		topCatSpend := ins.Numbers["top_category_spend"]
		topMer := ins.Strings["top_merchant"]
		topMerSpend := ins.Numbers["top_merchant_spend"]
		// All four parts (count, total, top category, top merchant) are
		// always present when this rule fires, so a single template covers
		// every case.
		return fmt.Sprintf(
			"You logged %d transactions totaling $%.2f. Top category was %s ($%.2f); top merchant was %s ($%.2f).",
			count, total, topCat, topCatSpend, topMer, topMerSpend,
		)
	case RuleCategoryShift:
		cat := ins.Strings["category"]
		dir := ins.Strings["direction"]
		cur := ins.Numbers["current"]
		prior := ins.Numbers["prior_avg"]
		delta := ins.Numbers["delta_pct"]
		if delta < 0 {
			delta = -delta
		}
		return fmt.Sprintf(
			"%s spending is %s about %.0f%% in the last 30 days ($%.2f vs a $%.2f prior 30-day average).",
			cat, dir, delta, cur, prior,
		)
	case RuleNewRecurringMerchant:
		merchant := ins.Strings["merchant"]
		count := int(ins.Numbers["count"])
		total := ins.Numbers["total"]
		return fmt.Sprintf(
			"%s is a new recurring charge — %d charges totaling $%.2f in the last 30 days, with no prior history.",
			merchant, count, total,
		)
	case RuleDayOfWeekCluster:
		cat := ins.Strings["category"]
		dow := ins.Strings["dow"]
		share := ins.Numbers["share"]
		dowTotal := ins.Numbers["dow_total"]
		catTotal := ins.Numbers["category_total"]
		return fmt.Sprintf(
			"%.0f%% of your %s spending happens on %ss ($%.2f of $%.2f over the last 90 days).",
			share, cat, dow, dowTotal, catTotal,
		)
	case RulePriceEscalation:
		merchant := ins.Strings["merchant"]
		last := ins.Numbers["last"]
		med := ins.Numbers["median"]
		delta := ins.Numbers["delta_pct"]
		return fmt.Sprintf(
			"%s charges are creeping up — your latest charge of $%.2f is %.0f%% above the $%.2f median of prior charges.",
			merchant, last, delta, med,
		)
	case RuleAnomalyCluster:
		cat := ins.Strings["category"]
		count := int(ins.Numbers["count"])
		total := ins.Numbers["total"]
		mean := ins.Numbers["mean"]
		wStart := ins.Strings["window_start"]
		wEnd := ins.Strings["window_end"]
		return fmt.Sprintf(
			"%d unusually large %s transactions totaling $%.2f hit between %s and %s, well above your category mean of $%.2f.",
			count, cat, total, wStart, wEnd, mean,
		)
	default:
		// Fallback to whatever the rule already produced. Never invent.
		if ins.Description != "" {
			return ins.Description
		}
		return ins.Title
	}
}

// ---------------------------------------------------------------------------
// Phase B — LLM polish with strict allow-set guard
// ---------------------------------------------------------------------------

// LLMClient is the minimum interface the narrator needs from the AI service.
// Defined locally so tests can pass a fake without depending on the ai package.
type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type llmNarrator struct {
	templateNarrator
	client LLMClient
}

// NewLLMNarrator returns a Narrator that runs Phase A first, then asks the LLM
// to polish the result. If the LLM fails or violates the safety guard, the
// returned narrative falls back to Phase A unchanged.
func NewLLMNarrator(client LLMClient) Narrator {
	return &llmNarrator{client: client}
}

func (n *llmNarrator) Narrate(ctx context.Context, insights []Insight, period Period) (MonthInReviewNarrative, error) {
	phaseA, err := n.templateNarrator.Narrate(ctx, insights, period)
	if err != nil {
		return phaseA, err
	}
	if len(insights) == 0 {
		// Nothing to polish.
		return phaseA, nil
	}

	prompt := buildLLMPrompt(phaseA, period)
	llmOut, err := n.client.Generate(ctx, prompt)
	if err != nil {
		log.Printf("insights narrator: LLM error, falling back to template: %v", err)
		return phaseA, nil
	}

	allowed := extractAllowedTokensFromNarrative(phaseA, insights)
	if err := validateLLMOutput(llmOut, allowed); err != nil {
		log.Printf("insights narrator: LLM output rejected by guard, falling back to template: %v", err)
		return phaseA, nil
	}

	// Accept LLM output as the new Overall paragraph. We deliberately do NOT
	// replace per-insight strings with LLM text — those are factual one-liners
	// and the LLM's job is to add a smooth wrapper, not to rewrite each line.
	polished := MonthInReviewNarrative{
		Overall:    strings.TrimSpace(llmOut),
		PerInsight: phaseA.PerInsight,
		Source:     "llm",
	}
	return polished, nil
}

func buildLLMPrompt(phaseA MonthInReviewNarrative, period Period) string {
	keys := make([]string, 0, len(phaseA.PerInsight))
	for k := range phaseA.PerInsight {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, phaseA.PerInsight[k])
	}
	return prompts.MustRender("insight_narrator", map[string]any{
		"Period":  period.describe(),
		"Overall": phaseA.Overall,
		"Lines":   lines,
	})
}

// ---------------------------------------------------------------------------
// Guard: extract allowed tokens from Phase A text + insight structure
// ---------------------------------------------------------------------------

// AllowedTokens is the set of values the LLM is allowed to mention. Anything
// outside these sets in the LLM's output is treated as hallucination.
type AllowedTokens struct {
	Numbers   map[string]struct{}
	Dates     map[string]struct{}
	Merchants map[string]struct{}
}

// numberRE matches numeric tokens including dollar signs, commas, decimals,
// and percentages. We extract a normalized form (digits + optional decimal).
var numberRE = regexp.MustCompile(`\$?\d{1,3}(?:,\d{3})*(?:\.\d+)?%?|\d+(?:\.\d+)?%?`)

// dateRE matches a few common date forms the templates produce:
//   - 2026-04-22
//   - Apr 2 / Jan 15
//   - 30 days, 60 days, 90 days (numeric+unit, captured via numberRE already)
var dateRE = regexp.MustCompile(`\b(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2}(?:,\s*\d{4})?\b|\b\d{4}-\d{2}-\d{2}\b|\b(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun)(?:day|sday|nesday|rsday|urday)?s?\b`)

// normalizeNumber strips $, commas, and trailing % so "$1,234.50" and "1234.5"
// compare equal. Trailing zero in the decimal is also normalized.
func normalizeNumber(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	s = strings.TrimSuffix(s, "%")
	s = strings.ReplaceAll(s, ",", "")
	// Trim trailing ".0" / ".00" so "30" and "30.0" match.
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

func extractNumbers(text string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, m := range numberRE.FindAllString(text, -1) {
		n := normalizeNumber(m)
		if n != "" {
			out[n] = struct{}{}
		}
	}
	return out
}

func extractDates(text string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, m := range dateRE.FindAllString(text, -1) {
		out[strings.ToLower(strings.TrimSpace(m))] = struct{}{}
	}
	return out
}

// extractAllowedTokens returns the sets of tokens that the LLM may legitimately
// reference. It scans the Phase A text only — extractAllowedTokensFromNarrative
// adds in merchants/categories from the structured insights themselves.
func extractAllowedTokens(phaseAText string) (numbers, dates, merchants map[string]struct{}) {
	return extractNumbers(phaseAText), extractDates(phaseAText), map[string]struct{}{}
}

// extractAllowedTokensFromNarrative builds the full AllowedTokens set, drawing
// numbers and dates from the Phase A prose AND merchant/category names from the
// raw Insight.Strings (which are guaranteed factual).
func extractAllowedTokensFromNarrative(narr MonthInReviewNarrative, insights []Insight) AllowedTokens {
	all := narr.Overall + "\n" + strings.Join(mapValues(narr.PerInsight), "\n")
	at := AllowedTokens{
		Numbers:   extractNumbers(all),
		Dates:     extractDates(all),
		Merchants: map[string]struct{}{},
	}
	// Add structured merchant/category names from insights so case-only changes
	// don't trip the guard.
	for _, ins := range insights {
		for _, k := range []string{"merchant", "category", "dow"} {
			if v := strings.TrimSpace(ins.Strings[k]); v != "" {
				at.Merchants[strings.ToLower(v)] = struct{}{}
			}
		}
	}
	// Also harvest capitalized words from Phase A as merchant/category-ish tokens.
	for _, w := range capitalizedWordsRE.FindAllString(all, -1) {
		at.Merchants[strings.ToLower(w)] = struct{}{}
	}
	return at
}

var capitalizedWordsRE = regexp.MustCompile(`\b[A-Z][a-zA-Z]{2,}(?:\s+[A-Z][a-zA-Z]+)*\b`)

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

// validateLLMOutput rejects LLM text that mentions any number, date, or
// capitalized merchant/category not in the allowed set. Common English words
// (high-priority, the, etc.) are not in the merchant set but they're not
// extracted by the merchant regex either — we only flag capitalized multi-letter
// tokens that don't appear in Phase A.
func validateLLMOutput(llmText string, allowed AllowedTokens) error {
	llmText = strings.TrimSpace(llmText)
	if llmText == "" {
		return fmt.Errorf("LLM output empty")
	}

	// Numbers
	for _, m := range numberRE.FindAllString(llmText, -1) {
		n := normalizeNumber(m)
		if n == "" {
			continue
		}
		if _, ok := allowed.Numbers[n]; !ok {
			return fmt.Errorf("LLM introduced unauthorized number: %q (normalized %q)", m, n)
		}
	}

	// Dates
	for _, m := range dateRE.FindAllString(llmText, -1) {
		key := strings.ToLower(strings.TrimSpace(m))
		if _, ok := allowed.Dates[key]; !ok {
			return fmt.Errorf("LLM introduced unauthorized date/day token: %q", m)
		}
	}

	// Merchants/categories: any capitalized multi-letter token in the LLM
	// output must appear (case-insensitively) somewhere in the allowed merchant
	// set OR as a capitalized word in Phase A. We pre-baked Phase A's
	// capitalized words into Merchants in extractAllowedTokensFromNarrative.
	for _, w := range capitalizedWordsRE.FindAllString(llmText, -1) {
		// Skip first-word-of-sentence false positives by checking common
		// English starters that legitimately get capitalized.
		lower := strings.ToLower(w)
		if isCommonSentenceStarter(lower) {
			continue
		}
		if _, ok := allowed.Merchants[lower]; !ok {
			return fmt.Errorf("LLM introduced unauthorized merchant/category-like token: %q", w)
		}
	}

	return nil
}

// isCommonSentenceStarter returns true for words that frequently begin a
// sentence and are not real merchants/categories. This keeps the guard from
// false-positiving on innocuous sentence-initial capitalization.
func isCommonSentenceStarter(lower string) bool {
	switch lower {
	case "the", "your", "you", "this", "that", "these", "those",
		"here", "there", "it", "its", "in", "on", "at", "with",
		"and", "but", "or", "for", "to", "from", "of", "a", "an",
		"meanwhile", "overall", "additionally", "also", "however",
		"finally", "lastly", "first", "second", "third":
		return true
	}
	return false
}
