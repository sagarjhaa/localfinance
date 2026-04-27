package insights

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeLLM lets tests inject deterministic LLM responses or errors.
type fakeLLM struct {
	resp string
	err  error
}

func (f *fakeLLM) Generate(_ context.Context, _ string) (string, error) {
	return f.resp, f.err
}

func samplePeriod() Period {
	return Period{
		Start: time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Label: "the last 90 days",
	}
}

func sampleInsights() []Insight {
	return []Insight{
		{
			Key:    "k-cat",
			RuleID: RuleCategoryShift,
			Title:  "Dining spending up 122% vs prior 60d",
			Numbers: map[string]float64{
				"current": 400.00, "prior_avg": 180.00, "delta_pct": 122.22, "prior_total": 360.00,
			},
			Strings:  map[string]string{"category": "Dining", "direction": "up"},
			Priority: PriorityHigh,
		},
		{
			Key:    "k-merchant",
			RuleID: RuleNewRecurringMerchant,
			Title:  "New recurring charge: ChatGPT Plus",
			Numbers: map[string]float64{
				"count": 2, "total": 40.00,
			},
			Strings:  map[string]string{"merchant": "ChatGPT Plus", "merchant_key": "chatgpt plus"},
			Priority: PriorityMedium,
		},
	}
}

// ---------- Phase A determinism ----------

func TestTemplateNarrator_Deterministic(t *testing.T) {
	n := NewTemplateNarrator()
	got1, err := n.Narrate(context.Background(), sampleInsights(), samplePeriod())
	if err != nil {
		t.Fatalf("narrate: %v", err)
	}
	got2, err := n.Narrate(context.Background(), sampleInsights(), samplePeriod())
	if err != nil {
		t.Fatalf("narrate: %v", err)
	}
	if got1.Overall != got2.Overall {
		t.Errorf("Overall not deterministic:\n  %q\n  %q", got1.Overall, got2.Overall)
	}
	for k, v := range got1.PerInsight {
		if got2.PerInsight[k] != v {
			t.Errorf("PerInsight[%q] not deterministic:\n  %q\n  %q", k, v, got2.PerInsight[k])
		}
	}
	if got1.Source != "template" {
		t.Errorf("expected Source=template, got %q", got1.Source)
	}
}

func TestTemplateNarrator_RendersExpectedFacts(t *testing.T) {
	n := NewTemplateNarrator()
	got, _ := n.Narrate(context.Background(), sampleInsights(), samplePeriod())

	cat := got.PerInsight["k-cat"]
	for _, want := range []string{"Dining", "up", "$400.00", "$180.00"} {
		if !strings.Contains(cat, want) {
			t.Errorf("category insight missing %q in %q", want, cat)
		}
	}
	merch := got.PerInsight["k-merchant"]
	for _, want := range []string{"ChatGPT Plus", "$40.00", "2 charges"} {
		if !strings.Contains(merch, want) {
			t.Errorf("merchant insight missing %q in %q", want, merch)
		}
	}
}

func TestTemplateNarrator_EmptyInsights(t *testing.T) {
	n := NewTemplateNarrator()
	got, err := n.Narrate(context.Background(), nil, samplePeriod())
	if err != nil {
		t.Fatalf("narrate: %v", err)
	}
	if got.Overall == "" {
		t.Error("expected non-empty Overall for empty insights")
	}
	if len(got.PerInsight) != 0 {
		t.Errorf("expected empty PerInsight, got %d", len(got.PerInsight))
	}
}

// ---------- Guard tests ----------

func TestValidateLLMOutput_AllowsRephrasing(t *testing.T) {
	n := NewTemplateNarrator()
	phaseA, _ := n.Narrate(context.Background(), sampleInsights(), samplePeriod())
	allowed := extractAllowedTokensFromNarrative(phaseA, sampleInsights())

	// Reorder & reword without introducing new tokens.
	good := "Dining is up 122% in the last 30 days ($400.00 vs $180.00). ChatGPT Plus is a new recurring charge with 2 charges totaling $40.00."
	if err := validateLLMOutput(good, allowed); err != nil {
		t.Errorf("expected good rephrase to pass, got %v", err)
	}
}

func TestValidateLLMOutput_RejectsInventedNumber(t *testing.T) {
	n := NewTemplateNarrator()
	phaseA, _ := n.Narrate(context.Background(), sampleInsights(), samplePeriod())
	allowed := extractAllowedTokensFromNarrative(phaseA, sampleInsights())

	bad := "Dining is up 122% ($400.00) but you also spent $999.99 elsewhere."
	err := validateLLMOutput(bad, allowed)
	if err == nil {
		t.Fatal("expected guard to reject invented number $999.99")
	}
	if !strings.Contains(err.Error(), "999") {
		t.Errorf("expected error to mention 999, got %v", err)
	}
}

func TestValidateLLMOutput_RejectsInventedMerchant(t *testing.T) {
	n := NewTemplateNarrator()
	phaseA, _ := n.Narrate(context.Background(), sampleInsights(), samplePeriod())
	allowed := extractAllowedTokensFromNarrative(phaseA, sampleInsights())

	bad := "Dining is up 122% and you should review your Netflix subscription too."
	err := validateLLMOutput(bad, allowed)
	if err == nil {
		t.Fatal("expected guard to reject invented merchant 'Netflix'")
	}
}

func TestValidateLLMOutput_RejectsInventedDate(t *testing.T) {
	n := NewTemplateNarrator()
	phaseA, _ := n.Narrate(context.Background(), sampleInsights(), samplePeriod())
	allowed := extractAllowedTokensFromNarrative(phaseA, sampleInsights())

	bad := "On Mar 15 you had a big dining bill of $400.00."
	err := validateLLMOutput(bad, allowed)
	if err == nil {
		t.Fatal("expected guard to reject invented date 'Mar 15'")
	}
}

func TestValidateLLMOutput_RejectsEmpty(t *testing.T) {
	if err := validateLLMOutput("   ", AllowedTokens{}); err == nil {
		t.Fatal("expected error for empty LLM output")
	}
}

// ---------- LLM narrator fallback paths ----------

func TestLLMNarrator_FallsBackOnLLMError(t *testing.T) {
	llm := &fakeLLM{err: errors.New("ollama exploded")}
	n := NewLLMNarrator(llm)
	got, err := n.Narrate(context.Background(), sampleInsights(), samplePeriod())
	if err != nil {
		t.Fatalf("narrate: %v", err)
	}
	if got.Source != "template" {
		t.Errorf("expected fallback to template on LLM error, got %q", got.Source)
	}
	if got.Overall == "" {
		t.Error("expected non-empty Overall from fallback")
	}
}

func TestLLMNarrator_FallsBackOnGuardFailure(t *testing.T) {
	// LLM hallucinates a merchant.
	llm := &fakeLLM{resp: "Dining up 122% and your Netflix subscription is $11.99/mo"}
	n := NewLLMNarrator(llm)
	got, err := n.Narrate(context.Background(), sampleInsights(), samplePeriod())
	if err != nil {
		t.Fatalf("narrate: %v", err)
	}
	if got.Source != "template" {
		t.Errorf("expected fallback when guard rejects LLM output, got source=%q overall=%q", got.Source, got.Overall)
	}
}

func TestLLMNarrator_AcceptsCleanLLM(t *testing.T) {
	clean := "Dining is up 122% in the last 30 days ($400.00 vs $180.00 prior). ChatGPT Plus is a new recurring charge with 2 charges totaling $40.00."
	llm := &fakeLLM{resp: clean}
	n := NewLLMNarrator(llm)
	got, err := n.Narrate(context.Background(), sampleInsights(), samplePeriod())
	if err != nil {
		t.Fatalf("narrate: %v", err)
	}
	if got.Source != "llm" {
		t.Errorf("expected source=llm, got %q", got.Source)
	}
	if got.Overall != clean {
		t.Errorf("expected polished overall verbatim, got %q", got.Overall)
	}
	// Per-insight strings must remain template (factual) text.
	if len(got.PerInsight) != 2 {
		t.Errorf("expected 2 per-insight entries, got %d", len(got.PerInsight))
	}
}

func TestLLMNarrator_EmptyInsights_NoLLMCall(t *testing.T) {
	// If the fake's Generate is called, it returns this hallucinated text. We
	// expect it NOT to be called.
	llm := &fakeLLM{resp: "Surprise! $9999.99 detected."}
	n := NewLLMNarrator(llm)
	got, err := n.Narrate(context.Background(), nil, samplePeriod())
	if err != nil {
		t.Fatalf("narrate: %v", err)
	}
	if got.Source != "template" {
		t.Errorf("expected template source for empty insights, got %q", got.Source)
	}
	if strings.Contains(got.Overall, "9999") {
		t.Errorf("LLM was called for empty insights — unwanted: %q", got.Overall)
	}
}
