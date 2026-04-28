// Package insights implements the rules-based financial insights engine for Sophia.
//
// The engine takes a window of a user's transactions and runs deterministic detector
// rules over them. Each rule emits zero or more Insight values. Insights carry a
// stable Key (see keys.go) so dismissals persisted in Thesaurus survive recomputation.
//
// This package contains NO LLM calls. Narration of insights happens in a later phase.
package insights

import (
	"time"

	models "github.com/sagarjhaa/localfinance/internal/ai"
)

// Rule IDs are stable string constants. They are part of the Insight.Key digest, so
// changing them invalidates dismissals — treat as a wire format.
const (
	RuleCategoryShift        = "category_shift"         // category spend ±20% vs prior 90d baseline
	RuleNewRecurringMerchant = "new_recurring_merchant" // merchant with ≥2 charges in last 30d that wasn't in prior 60d
	RuleDayOfWeekCluster     = "day_of_week_cluster"    // category spend concentrated on one DOW > 50%
	RulePriceEscalation      = "price_escalation"       // recurring merchant amount up >10% over last 3 charges
	RuleAnomalyCluster       = "anomaly_cluster"        // ≥3 transactions within 7 days that are 3σ above category mean
)

// Priority levels for Insight.Priority. Stringly typed so the JSON shape stays simple.
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// RuleContext is the input every rule receives. The caller is responsible for
// pre-filtering Transactions to the desired analysis window (default: 90 days).
type RuleContext struct {
	UserID       string
	Transactions []models.TransactionRef
	// Now is injected so tests can pin "current time" without monkey-patching the clock.
	Now time.Time
}

// Insight is the engine's internal representation of a finding. It carries enough
// structured data (Numbers, Strings, EvidenceIDs) for a downstream narrator to
// render rich descriptions without needing to re-analyze the source transactions.
type Insight struct {
	Key         string             `json:"key"`          // deterministic, see keys.go
	RuleID      string             `json:"rule_id"`      // one of the Rule* constants
	Title       string             `json:"title"`        // short factual statement
	Description string             `json:"description"`  // template-rendered, no LLM
	Priority    string             `json:"priority"`     // PriorityHigh|Medium|Low
	EvidenceIDs []string           `json:"evidence_ids"` // transaction IDs supporting the finding
	Numbers     map[string]float64 `json:"numbers"`      // structured data for the narrator
	Strings     map[string]string  `json:"strings"`      // merchant names, categories, etc.
	CreatedAt   time.Time          `json:"created_at"`
}

// ToFinancialInsight converts an internal Insight to the public sophia/models type
// returned by HTTP handlers. EvidenceIDs, Numbers, and Strings are dropped (consumed
// by the narrator before serialization), but Key and RuleID are propagated so the
// UI can call the dismiss endpoint on a stable identifier.
func (i Insight) ToFinancialInsight() models.FinancialInsight {
	return models.FinancialInsight{
		Type:        i.RuleID,
		Title:       i.Title,
		Description: i.Description,
		Priority:    i.Priority,
		ActionItem:  "",
		CreatedAt:   i.CreatedAt,
		Key:         i.Key,
		RuleID:      i.RuleID,
		EvidenceIDs: i.EvidenceIDs,
	}
}

// Rule is the function signature each detector implements. Rules MUST be pure:
// same input -> same output, no side effects, no I/O.
type Rule func(ctx RuleContext) []Insight
