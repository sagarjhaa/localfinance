package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/sagarjhaa/localfinance/internal/prompts"
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
  description     text NOT NULL,   -- merchant name, normalized at parse time ("Amazon", "Safeway"). Use ILIKE for fuzzy match (case-insensitive).
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

// ProgressFunc is the optional callback every step of the chat flow
// invokes to report what it's doing. The frontend uses these to render
// a live status line ("planning queries…", "running 2 queries…").
// nil is acceptable — non-streaming callers pass nil and pay nothing.
type ProgressFunc func(phase, detail string)

// AnswerFinancialQueryStream is the streaming entry point. It runs the
// same chat flow as AnswerFinancialQuery but invokes progress on every
// phase so callers (the SSE handler) can forward live status to the UI.
// The final AIResponse is returned through the normal return; progress
// is purely informational.
func (s *Service) AnswerFinancialQueryStream(ctx context.Context, query FinancialQuery, progress ProgressFunc) (AIResponse, error) {
	userModel := s.GetUserModelPreference(query.UserID)
	if isConversational(query.Question) {
		emit(progress, "compose", "Writing a quick reply…")
		return s.handleConversational(query.Question, userModel)
	}
	if s.db == nil {
		emit(progress, "fallback", "Looking through your transactions…")
		return s.handleFinancialQuery(query, userModel)
	}
	return s.answerWithSQLPlanner(ctx, query, userModel, progress)
}

func emit(p ProgressFunc, phase, detail string) {
	if p != nil {
		p(phase, detail)
	}
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
func (s *Service) answerWithSQLPlanner(ctx context.Context, query FinancialQuery, userModel string, progress ProgressFunc) (AIResponse, error) {
	emit(progress, "plan", "Asking AI to plan queries…")
	plan, planErr := s.planChatSQL(ctx, query.Question, userModel)
	if planErr != nil || len(plan.Queries) == 0 {
		log.Printf("chat: SQL planner produced no queries (%v); falling back to legacy intent flow", planErr)
		emit(progress, "fallback", "Falling back to simpler search…")
		return s.handleFinancialQuery(query, userModel)
	}

	emit(progress, "execute", fmt.Sprintf("Running %d quer%s…", len(plan.Queries), pluralY(len(plan.Queries))))
	results := s.executeChatPlanWithProgress(ctx, plan, query.UserID, progress)

	// One-shot self-correct: if every query was rejected/errored, send
	// the rejection reasons back to the LLM and ask for a fix. This
	// turns the most common LLM mistake (wrong user-scope shape) into
	// a single retry instead of a silent fallback.
	if everyQueryFailed(results) {
		log.Printf("chat: every planned query failed first pass; asking LLM to fix")
		emit(progress, "replan", "First attempt didn't run. Asking AI to fix…")
		fixed, fixErr := s.replanChatSQL(ctx, query.Question, plan, results, userModel)
		if fixErr != nil {
			log.Printf("chat.replan: LLM returned unparseable retry plan: %v", fixErr)
		} else if len(fixed.Queries) == 0 {
			log.Printf("chat.replan: LLM returned empty retry plan")
		} else {
			log.Printf("chat.replan: retry produced %d queries; running them now", len(fixed.Queries))
			emit(progress, "execute", fmt.Sprintf("Re-running %d quer%s…", len(fixed.Queries), pluralY(len(fixed.Queries))))
			results = s.executeChatPlanWithProgress(ctx, fixed, query.UserID, progress)
		}
	}

	if everyQueryFailed(results) {
		// Both passes failed. Don't silently fall back to the legacy flow
		// — it's the one that hallucinates totals from a free-form prompt
		// over raw transaction lists. Better to admit the system bailed
		// than to confidently print wrong numbers.
		log.Printf("chat: both plan attempts failed; returning a clear bail-out instead of legacy flow")
		var failures []string
		for _, r := range results {
			failures = append(failures, fmt.Sprintf("%s — %s", r.Name, r.Note))
		}
		return AIResponse{
			Answer: "I couldn't put together a query for that question. " +
				"Try rephrasing — for example, ‘How much did I spend on Amazon last month?’ " +
				"or ‘List my biggest 5 charges this year.’",
			Confidence:  0.1,
			GeneratedAt: time.Now(),
		}, nil
	}

	emit(progress, "compose", "Writing the answer…")
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

// pluralY returns "y" when n == 1 and "ies" otherwise. Used for
// "1 query" vs "3 queries" in progress strings.
func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// executeChatPlanWithProgress is executeChatPlan but emits per-query
// status updates. Kept separate from executeChatPlan so the latter
// stays a no-callback helper for non-streaming callers.
func (s *Service) executeChatPlanWithProgress(ctx context.Context, plan chatPlanRequest, userID string, progress ProgressFunc) []chatQueryResult {
	out := make([]chatQueryResult, 0, len(plan.Queries))
	for i, q := range plan.Queries {
		emit(progress, "query", fmt.Sprintf("Query %d/%d: %s", i+1, len(plan.Queries), q.Name))
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

// planChatSQL is pass 1.
func (s *Service) planChatSQL(ctx context.Context, question, userModel string) (chatPlanRequest, error) {
	prompt := prompts.MustRender("chat_plan", map[string]any{
		"Schema":   chatSchemaDoc,
		"Question": question,
	})
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
	// Render each query's rows as flat strings up-front so the template
	// just iterates structured data.
	type queryView struct {
		Name string
		SQL  string
		Note string
		Rows []string
	}
	views := make([]queryView, 0, len(results))
	for _, r := range results {
		v := queryView{Name: r.Name, SQL: collapseWhitespace(r.SQL), Note: r.Note}
		if r.Note == "" {
			for j, row := range r.Rows {
				if j >= 50 {
					v.Rows = append(v.Rows, fmt.Sprintf("... (%d more rows omitted)", len(r.Rows)-50))
					break
				}
				v.Rows = append(v.Rows, formatRowForPrompt(row))
			}
		}
		views = append(views, v)
	}
	prompt := prompts.MustRender("chat_answer", map[string]any{
		"Question": question,
		"Results":  views,
	})
	answer, err := s.queryOllamaWithModel(prompt, 0.2, userModel)
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

// everyQueryFailed returns true when every result has a Note (i.e.
// validation rejection or DB error) and zero rows. A query that
// legitimately returns 0 rows with no Note is success — the user's
// answer might just be "no match", which is honest.
func everyQueryFailed(results []chatQueryResult) bool {
	if len(results) == 0 {
		return true
	}
	for _, r := range results {
		if r.Note == "" {
			return false
		}
	}
	return true
}

// replanChatSQL asks the LLM to rewrite a plan that wholly failed.
// We feed back the original question, the queries it wrote, and the
// per-query rejection reason so it has concrete grounds to correct.
// Best-effort — caller should still gracefully degrade if this also
// produces nothing usable.
func (s *Service) replanChatSQL(ctx context.Context, question string, prev chatPlanRequest, results []chatQueryResult, userModel string) (chatPlanRequest, error) {
	type failedRow struct{ Name, SQL, Reason string }
	failed := make([]failedRow, 0, len(prev.Queries))
	for i, q := range prev.Queries {
		row := failedRow{Name: q.Name, SQL: collapseWhitespace(q.SQL)}
		if i < len(results) {
			row.Reason = results[i].Note
		}
		failed = append(failed, row)
	}
	prompt := prompts.MustRender("chat_replan", map[string]any{
		"Schema":   chatSchemaDoc,
		"Failed":   failed,
		"Question": question,
	})
	raw, err := s.queryOllamaWithModel(prompt, 0.1, userModel)
	if err != nil {
		return chatPlanRequest{}, err
	}
	jsonStr := extractJSON(raw)
	var plan chatPlanRequest
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return chatPlanRequest{}, fmt.Errorf("parse retry plan json: %w (raw: %s)", err, truncate(raw, 200))
	}
	return plan, nil
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
	// Strip accidental quotes around the placeholder. LLMs sometimes
	// emit `WHERE user_id = ':user_id'` which becomes a literal string
	// after substitution and Postgres rejects it as a non-UUID.
	stripped = strings.ReplaceAll(stripped, "':user_id'", ":user_id")
	stripped = strings.ReplaceAll(stripped, "\":user_id\"", ":user_id")

	// Coerce common alternate placeholders to :user_id.
	if !strings.Contains(stripped, ":user_id") {
		alts := []string{"$1", "?"}
		for _, alt := range alts {
			pat := regexp.MustCompile(`(?i)\.user_id\s*=\s*` + regexp.QuoteMeta(alt))
			stripped = pat.ReplaceAllString(stripped, ".user_id = :user_id")
		}
		// Hardcoded UUID literal? Replace too — never trust an LLM-
		// generated UUID, always substitute the authed user's.
		uuidLiteralRE := regexp.MustCompile(`(?i)\.user_id\s*=\s*'[0-9a-f-]{8,}'`)
		stripped = uuidLiteralRE.ReplaceAllString(stripped, ".user_id = :user_id")
	}

	// CRITICAL: the user-scope filter must reference the *user_id column*
	// of the accounts table — never `account_id` (which is the FK to
	// accounts.id and would make us scope by some random account UUID).
	// Match qualified forms: accounts.user_id, a.user_id, etc. Naked
	// "user_id =" without a table prefix is also accepted because the
	// accounts table is the only one with that column.
	scopeRE := regexp.MustCompile(`(?i)(?:\baccounts\.|\b[a-z_]\w{0,30}\.)?user_id\s*=\s*:user_id`)
	if !scopeRE.MatchString(stripped) {
		return "", fmt.Errorf("query must filter by accounts.user_id = :user_id (saw something else, possibly account_id)")
	}
	// Also: forbid `account_id = :user_id` outright. That's the most
	// common mistake — the column name shares a substring with user_id
	// but holds account UUIDs, so the substitution silently returns
	// zero rows.
	if regexp.MustCompile(`(?i)\baccount_id\s*=\s*:user_id`).MatchString(stripped) {
		return "", fmt.Errorf("account_id is not user_id — use accounts.user_id and JOIN through accounts")
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
