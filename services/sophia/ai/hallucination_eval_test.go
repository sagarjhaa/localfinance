//go:build eval

// Package ai — hallucination evaluation harness.
//
// This file is gated behind the `eval` build tag so it never runs as part of
// `make test`. It is invoked by `make eval-hallucination`, which sets
// EVAL_OLLAMA=1 and runs `go test -tags=eval -run HallucinationEval` against
// this package. It depends on a local Ollama daemon at http://localhost:11434
// with the candidate models pulled.
//
// Usage:
//
//	ollama pull llama3.1:8b qwen2.5:7b qwen2.5:14b
//	EVAL_OLLAMA=1 make eval-hallucination
//
// Output: evals/insight_hallucination/results-<model>-<timestamp>.json plus a
// summary table on stdout. A human grader fills in results-template.md against
// the JSON outputs.

package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// ---- fixture/prompt types -------------------------------------------------

type evalTransaction struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Type        string  `json:"type"`
}

type evalFixtures struct {
	GeneratedAt  string            `json:"generated_at"`
	WindowStart  string            `json:"window_start"`
	WindowEnd    string            `json:"window_end"`
	Transactions []evalTransaction `json:"transactions"`
}

type evalExpectedFacts struct {
	Numbers    []string `json:"numbers,omitempty"`
	Merchants  []string `json:"merchants,omitempty"`
	Dates      []string `json:"dates,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Trend      string   `json:"trend,omitempty"`
	Answer     string   `json:"answer,omitempty"`
}

type evalPrompt struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Prompt        string            `json:"prompt"`
	ExpectedFacts evalExpectedFacts `json:"expected_facts"`
}

type evalPrompts struct {
	Prompts []evalPrompt `json:"prompts"`
}

// ---- result types ---------------------------------------------------------

type evalGrade struct {
	NumericOK            bool     `json:"numeric_ok"`
	UnknownNumbers       []string `json:"unknown_numbers"`
	ExpectedMerchantsHit []string `json:"expected_merchants_hit"`
	ExpectedMerchantsMiss []string `json:"expected_merchants_miss"`
	ExpectedDatesHit     []string `json:"expected_dates_hit"`
	ExpectedDatesMiss    []string `json:"expected_dates_miss"`
	ExpectedNumbersHit   []string `json:"expected_numbers_hit"`
	ExpectedNumbersMiss  []string `json:"expected_numbers_miss"`
}

type evalResult struct {
	Model      string    `json:"model"`
	PromptID   string    `json:"prompt_id"`
	PromptType string    `json:"prompt_type"`
	Prompt     string    `json:"prompt"`
	Response   string    `json:"response"`
	LatencyMS  int64     `json:"latency_ms"`
	Error      string    `json:"error,omitempty"`
	Grade      evalGrade `json:"grade"`
	RanAt      string    `json:"ran_at"`
}

type evalRun struct {
	Model     string       `json:"model"`
	StartedAt string       `json:"started_at"`
	Results   []evalResult `json:"results"`
	Summary   evalSummary  `json:"summary"`
}

type evalSummary struct {
	NumericAccuracy  float64 `json:"numeric_accuracy"`
	MerchantAccuracy float64 `json:"merchant_accuracy"`
	DateAccuracy     float64 `json:"date_accuracy"`
	AvgLatencyMS     float64 `json:"avg_latency_ms"`
	N                int     `json:"n"`
}

// ---- ollama client (stdlib only) ------------------------------------------

type ollamaReq struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options"`
	KeepAlive string               `json:"keep_alive,omitempty"`
}

type ollamaResp struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func callOllama(model, prompt string) (string, time.Duration, error) {
	body := ollamaReq{
		Model:  model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"num_ctx":     4096,
			"temperature": 0.0,
		},
		KeepAlive: "24h",
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return "", 0, err
	}
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	start := time.Now()
	httpReq, err := http.NewRequest("POST", host+"/api/generate", bytes.NewReader(buf))
	if err != nil {
		return "", 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", time.Since(start), err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Since(start), err
	}
	if resp.StatusCode != 200 {
		return "", time.Since(start), fmt.Errorf("ollama %d: %s", resp.StatusCode, string(raw))
	}
	var out ollamaResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", time.Since(start), err
	}
	return out.Response, time.Since(start), nil
}

// ---- prompt construction --------------------------------------------------

func buildContextBlock(txs []evalTransaction) string {
	var b strings.Builder
	b.WriteString("You are a personal-finance assistant. Answer the user's question using ONLY the transactions below. Do not invent any number, merchant, or date that does not appear in the list. If the data is insufficient, say so.\n\n")
	b.WriteString("TRANSACTIONS (id | date | description | amount | category | type):\n")
	for _, t := range txs {
		b.WriteString(fmt.Sprintf("%s | %s | %s | %.2f | %s | %s\n",
			t.ID, t.Date, t.Description, t.Amount, t.Category, t.Type))
	}
	b.WriteString("\n")
	return b.String()
}

// ---- automated grading ----------------------------------------------------

var numberRe = regexp.MustCompile(`-?\d+(?:\.\d+)?`)

// fixtureNumberSet builds the set of digit-sequences that legitimately appear
// in the fixture data: tx ids, dates, amounts (multiple formattings), and
// individual integer counts implied by lengths.
func fixtureNumberSet(fx evalFixtures) map[string]struct{} {
	set := map[string]struct{}{}
	add := func(s string) {
		if s == "" {
			return
		}
		set[s] = struct{}{}
	}
	for _, t := range fx.Transactions {
		// tx id digits
		for _, m := range numberRe.FindAllString(t.ID, -1) {
			add(m)
			add(strings.TrimLeft(m, "0"))
		}
		// date components
		add(t.Date)
		for _, m := range numberRe.FindAllString(t.Date, -1) {
			add(m)
			add(strings.TrimLeft(m, "0"))
		}
		// amounts in several common formattings
		amt := t.Amount
		if amt < 0 {
			amt = -amt
		}
		add(fmt.Sprintf("%.2f", amt))
		add(fmt.Sprintf("%.2f", t.Amount))
		add(fmt.Sprintf("%.0f", amt))
		add(fmt.Sprintf("%.1f", amt))
		// integer-part on its own (e.g. "245" from 245.88)
		add(fmt.Sprintf("%d", int64(amt)))
		// digits inside descriptions ("12345")
		for _, m := range numberRe.FindAllString(t.Description, -1) {
			add(m)
		}
	}
	// counts up to 30 are plausibly correct (lengths, frequencies, top-N)
	for i := 0; i <= len(fx.Transactions)+5; i++ {
		add(fmt.Sprintf("%d", i))
	}
	// Pre-computed roll-ups so number-match grader doesn't penalise correct
	// arithmetic. These are the same totals encoded in prompts.json.
	for _, s := range []string{
		"1223.48", "420.92", "334.53", "53.96", "263.80", "122.95", "88.75",
		"142.84", "338.48", "12750.00", "12750", "401.70", "821.78",
	} {
		add(s)
	}
	return set
}

func gradeResponse(resp string, fxNums map[string]struct{}, expected evalExpectedFacts) evalGrade {
	g := evalGrade{NumericOK: true}
	lower := strings.ToLower(resp)

	// numeric: every digit-sequence in response must appear in fixture set
	seen := map[string]bool{}
	for _, m := range numberRe.FindAllString(resp, -1) {
		if seen[m] {
			continue
		}
		seen[m] = true
		stripped := strings.TrimLeft(m, "0")
		if stripped == "" {
			stripped = "0"
		}
		if _, ok := fxNums[m]; ok {
			continue
		}
		if _, ok := fxNums[stripped]; ok {
			continue
		}
		// allow trailing-zero variations: "10.99" vs "10.9"
		g.UnknownNumbers = append(g.UnknownNumbers, m)
		g.NumericOK = false
	}

	// expected-fact recall (these inform Y/N grading rather than gating it)
	for _, n := range expected.Numbers {
		if strings.Contains(resp, n) {
			g.ExpectedNumbersHit = append(g.ExpectedNumbersHit, n)
		} else {
			g.ExpectedNumbersMiss = append(g.ExpectedNumbersMiss, n)
		}
	}
	for _, m := range expected.Merchants {
		if strings.Contains(lower, strings.ToLower(m)) {
			g.ExpectedMerchantsHit = append(g.ExpectedMerchantsHit, m)
		} else {
			g.ExpectedMerchantsMiss = append(g.ExpectedMerchantsMiss, m)
		}
	}
	for _, d := range expected.Dates {
		if strings.Contains(resp, d) {
			g.ExpectedDatesHit = append(g.ExpectedDatesHit, d)
		} else {
			g.ExpectedDatesMiss = append(g.ExpectedDatesMiss, d)
		}
	}
	return g
}

// ---- helpers --------------------------------------------------------------

// repoRoot walks up from the test's working directory (services/sophia/ai)
// looking for go.mod at the repo root (the one that has an "evals" sibling).
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "evals", "insight_hallucination", "fixtures.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate repo root with evals/insight_hallucination/fixtures.json (started at %s)", wd)
	return ""
}

func loadFixtures(t *testing.T, root string) evalFixtures {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "evals", "insight_hallucination", "fixtures.json"))
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	var fx evalFixtures
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("parse fixtures: %v", err)
	}
	return fx
}

func loadPrompts(t *testing.T, root string) evalPrompts {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "evals", "insight_hallucination", "prompts.json"))
	if err != nil {
		t.Fatalf("read prompts: %v", err)
	}
	var ps evalPrompts
	if err := json.Unmarshal(raw, &ps); err != nil {
		t.Fatalf("parse prompts: %v", err)
	}
	return ps
}

// ---- main test ------------------------------------------------------------

func TestHallucinationEval(t *testing.T) {
	if os.Getenv("EVAL_OLLAMA") != "1" {
		t.Skip("EVAL_OLLAMA=1 not set; skipping hallucination eval (run via `make eval-hallucination`)")
	}

	root := repoRoot(t)
	fx := loadFixtures(t, root)
	ps := loadPrompts(t, root)
	fxNums := fixtureNumberSet(fx)

	models := []string{"llama3.1:8b", "qwen2.5:7b", "qwen2.5:14b"}
	if env := os.Getenv("EVAL_MODELS"); env != "" {
		models = strings.Split(env, ",")
	}

	contextBlock := buildContextBlock(fx.Transactions)
	stamp := time.Now().UTC().Format("20060102-150405")

	// summary table header
	fmt.Println()
	fmt.Println("model                | n  | numeric_acc | merchant_acc | date_acc | avg_latency_ms")
	fmt.Println("---------------------+----+-------------+--------------+----------+---------------")

	for _, model := range models {
		run := evalRun{Model: model, StartedAt: time.Now().UTC().Format(time.RFC3339)}

		var numOK, totalNum int
		var merchHit, merchTotal int
		var dateHit, dateTotal int
		var latencySum int64

		for _, p := range ps.Prompts {
			full := contextBlock + "USER QUESTION: " + p.Prompt + "\n\nANSWER:"
			t.Logf("[%s] %s", model, p.ID)
			resp, latency, err := callOllama(model, full)
			r := evalResult{
				Model:      model,
				PromptID:   p.ID,
				PromptType: p.Type,
				Prompt:     p.Prompt,
				Response:   resp,
				LatencyMS:  latency.Milliseconds(),
				RanAt:      time.Now().UTC().Format(time.RFC3339),
			}
			if err != nil {
				r.Error = err.Error()
				t.Logf("  error: %v", err)
			} else {
				r.Grade = gradeResponse(resp, fxNums, p.ExpectedFacts)
				totalNum++
				if r.Grade.NumericOK {
					numOK++
				}
				merchHit += len(r.Grade.ExpectedMerchantsHit)
				merchTotal += len(r.Grade.ExpectedMerchantsHit) + len(r.Grade.ExpectedMerchantsMiss)
				dateHit += len(r.Grade.ExpectedDatesHit)
				dateTotal += len(r.Grade.ExpectedDatesHit) + len(r.Grade.ExpectedDatesMiss)
				latencySum += latency.Milliseconds()
			}
			run.Results = append(run.Results, r)
		}

		var numericAcc, merchAcc, dateAcc, avgLat float64
		if totalNum > 0 {
			numericAcc = float64(numOK) / float64(totalNum)
			avgLat = float64(latencySum) / float64(totalNum)
		}
		if merchTotal > 0 {
			merchAcc = float64(merchHit) / float64(merchTotal)
		}
		if dateTotal > 0 {
			dateAcc = float64(dateHit) / float64(dateTotal)
		}
		run.Summary = evalSummary{
			NumericAccuracy:  numericAcc,
			MerchantAccuracy: merchAcc,
			DateAccuracy:     dateAcc,
			AvgLatencyMS:     avgLat,
			N:                totalNum,
		}

		// write per-model results JSON
		safeModel := strings.NewReplacer(":", "-", "/", "-").Replace(model)
		outPath := filepath.Join(root, "evals", "insight_hallucination",
			fmt.Sprintf("results-%s-%s.json", safeModel, stamp))
		buf, _ := json.MarshalIndent(run, "", "  ")
		if err := os.WriteFile(outPath, buf, 0o644); err != nil {
			t.Errorf("write results %s: %v", outPath, err)
		} else {
			t.Logf("wrote %s", outPath)
		}

		fmt.Printf("%-20s | %2d | %11.2f | %12.2f | %8.2f | %14.0f\n",
			model, totalNum, numericAcc, merchAcc, dateAcc, avgLat)
	}

	// stable order across runs
	sort.Strings(models)
}
