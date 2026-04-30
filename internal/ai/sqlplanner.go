package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// chatSchemaDoc is the schema description we hand to the LLM in pass 1.
// Hand-written rather than introspected so we can include semantic notes
// (sign convention on amount, allowed category values, the user_id JOIN
// indirection) that a bare CREATE TABLE wouldn't carry.
//
// Keep this in sync with internal/data/models when columns change.
const chatSchemaDoc = `Postgres schema (read-only — only SELECTs are executed):

TABLE accounts (
  id              uuid PRIMARY KEY,
  user_id         uuid NOT NULL,
  name            text,        -- friendly account name, may be empty
  institution     text,        -- "Capital One", "Chase", etc.
  account_number  text         -- masked, e.g. "XXXX1007"
)

TABLE transactions (
  id              uuid PRIMARY KEY,
  account_id      uuid NOT NULL REFERENCES accounts(id),
  date            timestamp NOT NULL,
  description     text NOT NULL,   -- raw bank merchant string ("AMZN MARKETPLACE"); use ILIKE for fuzzy match
  amount          numeric NOT NULL,
  category        text,            -- one of: Food, Transport, Shopping, Entertainment,
                                   --   Utilities, Housing, Income, Transfer, Card Payment,
                                   --   Health, Cash, EMI, Education, Other
  type            text,            -- "debit" or "credit"
  document_id     text             -- which uploaded statement this came from
)

TABLE documents (
  id                 text PRIMARY KEY,
  user_id            uuid NOT NULL,
  original_filename  text,
  status             text,         -- 'processed' | 'error' | 'processing'
  created_at         timestamp
)

SIGN CONVENTION on transactions.amount:
  amount > 0 — money the user spent (charges, purchases)
  amount < 0 — money the user received (refunds, payments-to-card, paychecks)

USER SCOPING:
  transactions has NO user_id column. Always JOIN to accounts and filter
  accounts.user_id = :user_id. The token :user_id will be substituted at
  execution time with the authenticated user's UUID — write it literally
  as ":user_id" in your SQL.

ROW LIMIT:
  All queries are auto-capped at LIMIT 200. You can write your own LIMIT
  but it must be ≤ 200. Use ORDER BY date DESC for "recent" queries.
`

// chatPlanRequest is what pass 1 returns: a small list of named SQL
// queries the LLM thinks will gather enough evidence for the answer.
type chatPlanRequest struct {
	Queries []chatPlanQuery `json:"queries"`
}

type chatPlanQuery struct {
	Name string `json:"name"` // human label, e.g. "amazon_purchases"
	SQL  string `json:"sql"`  // SELECT-only, references :user_id
	Why  string `json:"why"`  // one-line justification (debug only)
}

// chatQueryResult holds the rows returned from one query, formatted for
// the pass-2 prompt and for debugging.
type chatQueryResult struct {
	Name string                   `json:"name"`
	SQL  string                   `json:"sql"`
	Rows []map[string]interface{} `json:"rows"`
	Note string                   `json:"note,omitempty"` // truncation / error notes
}

// answerWithSQLPlanner runs the new chat flow:
//
//  1. Pass 1: LLM, given the schema + question, writes 1-N SELECT queries.
//  2. Each query is validated (SELECT-only, single statement, ≤200 rows,
//     :user_id substituted) and executed against the DB.
//  3. Pass 2: LLM, given the question + each query + its rows, writes a
//     concise natural-language answer.
//
// Falls back to the legacy intent flow if the planner returns nothing
// usable, so the user always gets *some* answer.
func (s *Service) answerWithSQLPlanner(ctx context.Context, query FinancialQuery, userModel string) (AIResponse, error) {
	plan, planErr := s.planChatSQL(ctx, query.Question, userModel)
	if planErr != nil || len(plan.Queries) == 0 {
		log.Printf("chat: SQL planner produced no queries (%v); falling back to legacy intent flow", planErr)
		return s.handleFinancialQuery(query, userModel)
	}

	results := s.executeChatPlan(ctx, plan, query.UserID)

	// If every query failed validation/execution, fall back. Otherwise
	// pass whatever we got (even partial) to the answer phase — the LLM
	// can still produce something useful.
	allEmpty := true
	for _, r := range results {
		if r.Note == "" || len(r.Rows) > 0 {
			allEmpty = false
			break
		}
	}
	if allEmpty {
		log.Printf("chat: every planned query failed; falling back to legacy intent flow")
		return s.handleFinancialQuery(query, userModel)
	}

	answer, srcs, err := s.composeChatAnswer(ctx, query.Question, results, userModel)
	if err != nil {
		return AIResponse{}, err
	}
	return AIResponse{
		Answer:      answer,
		Sources:     srcs,
		Confidence:  0.9,
		GeneratedAt: time.Now(),
	}, nil
}

// planChatSQL is pass 1.
func (s *Service) planChatSQL(ctx context.Context, question, userModel string) (chatPlanRequest, error) {
	prompt := fmt.Sprintf(`You are a SQL planner for a personal-finance chat. Given the user's question and the schema below, produce a small set of SQL queries that gather enough evidence to answer.

%s

INSTRUCTIONS:
- Output ONLY a JSON object {"queries": [{"name": ..., "sql": ..., "why": ...}, ...]}.
- Use SELECT only. No INSERT/UPDATE/DELETE/DDL.
- One statement per query (no semicolons in the middle).
- Always JOIN transactions to accounts so you can filter accounts.user_id = :user_id.
- Prefer ILIKE '%%term%%' for merchant searches (descriptions are raw bank strings).
- Cap with LIMIT 200 or smaller.
- If the question is broad (e.g. "how did I spend last month"), include a category-totals query and a top-merchants query.
- Keep the plan small — 1 to 3 queries is usually enough.
- Do not invent column names. Stick to the schema above.

QUESTION: %s`, chatSchemaDoc, question)

	raw, err := s.queryOllamaWithModel(prompt, 0.1, userModel)
	if err != nil {
		return chatPlanRequest{}, err
	}
	jsonStr := extractJSON(raw)
	var plan chatPlanRequest
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return chatPlanRequest{}, fmt.Errorf("parse plan json: %w (raw: %s)", err, truncate(raw, 200))
	}
	return plan, nil
}

// executeChatPlan validates each query, substitutes :user_id, runs it,
// and collects rows. Failures are recorded in r.Note rather than
// short-circuiting — we want the answer phase to see partial data.
func (s *Service) executeChatPlan(ctx context.Context, plan chatPlanRequest, userID string) []chatQueryResult {
	out := make([]chatQueryResult, 0, len(plan.Queries))
	for _, q := range plan.Queries {
		r := chatQueryResult{Name: q.Name, SQL: q.SQL}
		safe, err := validateChatSQL(q.SQL)
		if err != nil {
			r.Note = "rejected: " + err.Error()
			log.Printf("chat.plan rejected query %q: %v\n  sql: %s", q.Name, err, collapseWhitespace(q.SQL))
			out = append(out, r)
			continue
		}
		log.Printf("chat.plan running query %q: %s", q.Name, collapseWhitespace(safe))
		rows, err := s.runChatQuery(ctx, safe, userID)
		if err != nil {
			r.Note = "execution error: " + err.Error()
			log.Printf("chat.plan exec error on %q: %v", q.Name, err)
			out = append(out, r)
			continue
		}
		log.Printf("chat.plan %q → %d row(s)", q.Name, len(rows))
		r.Rows = rows
		out = append(out, r)
	}
	return out
}

// composeChatAnswer is pass 2 — feed the question + each query and its
// (truncated) rows back to the LLM and ask for a natural answer.
func (s *Service) composeChatAnswer(ctx context.Context, question string, results []chatQueryResult, userModel string) (string, []TransactionRef, error) {
	var b strings.Builder
	b.WriteString("You are a concise personal-finance assistant. The data below was fetched by running the SQL queries shown. Answer the user's question using ONLY the row data.\n\n")
	b.WriteString("RULES:\n")
	b.WriteString("- Use real numbers from the rows. Never invent amounts, dates, or merchants.\n")
	b.WriteString("- 2-4 sentences. Format currency as $X,XXX.XX.\n")
	b.WriteString("- Use clean merchant names (e.g. \"Amazon\" not \"AMZN MARKETPLACE\").\n")
	b.WriteString("- If a query returned no rows, say what was searched and that there were no matches.\n")
	b.WriteString("- Do not give generic advice; answer only what was asked.\n\n")
	b.WriteString("QUESTION: " + question + "\n\n")
	b.WriteString("EVIDENCE:\n")
	for i, r := range results {
		b.WriteString(fmt.Sprintf("--- Query %d: %s ---\n", i+1, r.Name))
		b.WriteString("SQL: " + collapseWhitespace(r.SQL) + "\n")
		if r.Note != "" {
			b.WriteString("Note: " + r.Note + "\n\n")
			continue
		}
		if len(r.Rows) == 0 {
			b.WriteString("Result: 0 rows.\n\n")
			continue
		}
		b.WriteString(fmt.Sprintf("Result: %d row(s)\n", len(r.Rows)))
		for j, row := range r.Rows {
			if j >= 50 {
				b.WriteString(fmt.Sprintf("... (%d more rows omitted)\n", len(r.Rows)-50))
				break
			}
			b.WriteString("  " + formatRowForPrompt(row) + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Now write the answer.")

	answer, err := s.queryOllamaWithModel(b.String(), 0.2, userModel)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(answer), collectTxnSources(results), nil
}

// collectTxnSources walks the result set and pulls out any rows that
// look like transaction rows (have id+date+description+amount). Used
// to populate AIResponse.Sources so the UI can show "Which
// transactions?" provenance.
func collectTxnSources(results []chatQueryResult) []TransactionRef {
	seen := make(map[string]bool)
	out := []TransactionRef{}
	for _, r := range results {
		for _, row := range r.Rows {
			ref, ok := rowAsTransactionRef(row)
			if !ok {
				continue
			}
			if ref.ID == "" || seen[ref.ID] {
				continue
			}
			seen[ref.ID] = true
			out = append(out, ref)
		}
	}
	return out
}

func rowAsTransactionRef(row map[string]interface{}) (TransactionRef, bool) {
	id, hasID := row["id"]
	if !hasID {
		return TransactionRef{}, false
	}
	idStr, _ := id.(string)
	desc, _ := row["description"].(string)
	cat, _ := row["category"].(string)
	tType, _ := row["type"].(string)
	dateStr := ""
	switch d := row["date"].(type) {
	case time.Time:
		dateStr = d.Format("2006-01-02")
	case string:
		dateStr = d
	}
	amt := 0.0
	switch a := row["amount"].(type) {
	case float64:
		amt = a
	case int64:
		amt = float64(a)
	}
	if dateStr == "" && desc == "" && amt == 0 {
		return TransactionRef{}, false
	}
	return TransactionRef{
		ID:          idStr,
		Date:        dateStr,
		Description: desc,
		Amount:      amt,
		Category:    cat,
		Type:        tType,
	}, true
}

// chatSQLForbidden flags any keyword that would mutate the database or
// reach beyond the user's data. Matched as a whole-word substring on
// the lower-cased SQL after comments are stripped.
var chatSQLForbidden = []string{
	"insert", "update", "delete", "drop", "truncate", "alter",
	"create", "grant", "revoke", "vacuum", "analyze", "copy",
	"do ", "call ",
}

var commentRE = regexp.MustCompile(`(?s)/\*.*?\*/|--[^\n]*`)
var limitRE = regexp.MustCompile(`(?i)\blimit\s+(\d+)\b`)

// validateChatSQL enforces the read-only + single-statement + bounded
// guarantees we promised the LLM in the plan prompt. Returns the
// cleaned SQL ready for execution, or an error describing why it was
// rejected. The :user_id placeholder is preserved; runChatQuery does
// the substitution.
func validateChatSQL(sql string) (string, error) {
	if strings.TrimSpace(sql) == "" {
		return "", fmt.Errorf("empty sql")
	}
	stripped := commentRE.ReplaceAllString(sql, " ")
	lower := strings.ToLower(stripped)
	trimmed := strings.TrimSpace(lower)
	if !strings.HasPrefix(trimmed, "select ") && !strings.HasPrefix(trimmed, "with ") {
		return "", fmt.Errorf("must start with SELECT or WITH")
	}
	for _, kw := range chatSQLForbidden {
		if strings.Contains(lower, " "+kw) || strings.HasPrefix(lower, kw) {
			return "", fmt.Errorf("forbidden keyword: %s", strings.TrimSpace(kw))
		}
	}
	// Reject anything after the first statement. Trailing ; is fine.
	endIdx := strings.Index(stripped, ";")
	if endIdx >= 0 {
		rest := strings.TrimSpace(stripped[endIdx+1:])
		if rest != "" {
			return "", fmt.Errorf("multiple statements not allowed")
		}
		stripped = stripped[:endIdx]
	}
	stripped = strings.TrimSpace(stripped)

	// Cap LIMIT at 200. If the LLM specified a higher one, clamp it.
	// If absent, append.
	if m := limitRE.FindStringSubmatchIndex(stripped); m != nil {
		// rebuild with clamped limit
		full := stripped[m[0]:m[1]]
		nStart, nEnd := m[2], m[3]
		nStr := stripped[nStart:nEnd]
		var n int
		fmt.Sscanf(nStr, "%d", &n)
		if n > 200 {
			n = 200
		}
		_ = full
		stripped = stripped[:nStart] + fmt.Sprintf("%d", n) + stripped[nEnd:]
	} else {
		stripped = stripped + " LIMIT 200"
	}
	// Must reference accounts.user_id somewhere — if the LLM forgot to
	// scope, refuse, otherwise we'd leak across users. The placeholder
	// for the actual UUID is :user_id; if the LLM wrote $1 / ? / a
	// hardcoded UUID we coerce to :user_id here so runChatQuery's
	// substitution still works.
	hasUserScope := strings.Contains(stripped, "user_id")
	if !hasUserScope {
		return "", fmt.Errorf("query must filter by accounts.user_id")
	}
	if !strings.Contains(stripped, ":user_id") {
		// Replace the most common alternates with our placeholder. Order
		// matters — match longest first.
		replacements := []string{"$1", "?"}
		for _, alt := range replacements {
			if strings.Contains(stripped, "user_id = "+alt) {
				stripped = strings.Replace(stripped, "user_id = "+alt, "user_id = :user_id", 1)
				break
			}
			if strings.Contains(stripped, "user_id="+alt) {
				stripped = strings.Replace(stripped, "user_id="+alt, "user_id=:user_id", 1)
				break
			}
		}
		// Also catch a hardcoded UUID literal — if the LLM put one in,
		// replace it with the real placeholder so we can substitute the
		// authed user's ID and not whatever the LLM dreamed up.
		uuidLiteralRE := regexp.MustCompile(`(?i)user_id\s*=\s*'[0-9a-f-]{8,}'`)
		stripped = uuidLiteralRE.ReplaceAllString(stripped, "user_id = :user_id")
	}
	if !strings.Contains(stripped, ":user_id") {
		return "", fmt.Errorf("could not normalize user_id filter")
	}
	return stripped, nil
}

// runChatQuery substitutes the :user_id placeholder with a parameterized
// $1 binding and executes the query, returning rows as a slice of
// column-name → value maps.
func (s *Service) runChatQuery(ctx context.Context, sql, userID string) ([]map[string]interface{}, error) {
	if s.db == nil {
		return nil, fmt.Errorf("db not configured")
	}
	parameterized := strings.ReplaceAll(sql, ":user_id", "?")
	tx := s.db.WithContext(ctx)
	rows, err := tx.Raw(parameterized, userID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		holders := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range holders {
			ptrs[i] = &holders[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return out, err
		}
		row := make(map[string]interface{}, len(cols))
		for i, c := range cols {
			row[c] = normalizeRowValue(holders[i])
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// normalizeRowValue converts the driver's raw types into ones the JSON
// encoder and the prompt formatter handle cleanly. Bytes → string,
// time.Time stays a time.Time, everything else passes through.
func normalizeRowValue(v interface{}) interface{} {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}

func formatRowForPrompt(row map[string]interface{}) string {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	// Stable order — Sort then move common keys to front for readability.
	pinned := []string{"date", "description", "amount", "category", "account", "name", "total"}
	in := func(k string) bool {
		for _, p := range pinned {
			if k == p {
				return true
			}
		}
		return false
	}
	rest := make([]string, 0)
	for _, k := range keys {
		if !in(k) {
			rest = append(rest, k)
		}
	}
	ordered := append([]string{}, pinned...)
	ordered = append(ordered, rest...)

	parts := make([]string, 0, len(row))
	for _, k := range ordered {
		v, ok := row[k]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, " | ")
}

// avoid extra dependency on strings.ReplaceAllN; collapse runs of
// whitespace to single spaces so the SQL renders cleanly in prompts.
var wsRE = regexp.MustCompile(`\s+`)

func collapseWhitespace(s string) string {
	return strings.TrimSpace(wsRE.ReplaceAllString(s, " "))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Compile-time assertion: gorm import is used.
var _ = (*gorm.DB)(nil)
