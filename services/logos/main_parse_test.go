package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubSophia returns an httptest.Server that mimics POST /api/v1/parse.
// `txns` is the transactions array the stub returns; `model` populates the
// model field; `status` overrides the response code (defaults to 200).
func stubSophia(t *testing.T, txns []map[string]interface{}, model string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/parse" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		_ = json.Unmarshal(body, &req)
		if req["text"] == "" {
			http.Error(w, "empty text", http.StatusBadRequest)
			return
		}

		if status == 0 {
			status = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"transactions": txns,
			"count":        len(txns),
			"model":        model,
		})
	}))
}

func TestParseViaSophia_HappyPath(t *testing.T) {
	stub := stubSophia(t, []map[string]interface{}{
		{"date": "2026-04-01", "description": "Coffee Shop", "amount": -4.50, "category": "Food", "type": "debit"},
		{"date": "2026-04-02", "description": "Paycheck", "amount": 1200.00, "category": "Income", "type": "credit"},
	}, "llama3.2:1b", 200)
	defer stub.Close()

	txns, model, err := parseViaSophia(stub.URL, "user-123", "any non empty text", "stmt.csv")
	if err != nil {
		t.Fatalf("parseViaSophia: %v", err)
	}
	if model != "llama3.2:1b" {
		t.Errorf("expected model llama3.2:1b, got %q", model)
	}
	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}
	if txns[0].Description != "Coffee Shop" {
		t.Errorf("description not carried through: %q", txns[0].Description)
	}
	if txns[0].Amount != -4.50 {
		t.Errorf("amount not carried through: %v", txns[0].Amount)
	}
	if txns[0].Category != "Food" {
		t.Errorf("category not carried through: %q", txns[0].Category)
	}
	if txns[0].FileSource != "stmt.csv" {
		t.Errorf("file source not stamped: %q", txns[0].FileSource)
	}
	if txns[0].Date.IsZero() {
		t.Errorf("date not parsed")
	}
}

func TestParseViaSophia_EmptyTransactionsIsNotError(t *testing.T) {
	stub := stubSophia(t, []map[string]interface{}{}, "llama3.2:1b", 200)
	defer stub.Close()

	txns, _, err := parseViaSophia(stub.URL, "u", "some text", "f.pdf")
	if err != nil {
		t.Fatalf("expected nil error on empty result, got %v", err)
	}
	if len(txns) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(txns))
	}
}

func TestParseViaSophia_EmptyTextRejected(t *testing.T) {
	_, _, err := parseViaSophia("http://unused", "u", "   ", "f.csv")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestParseViaSophia_NetworkErrorPropagates(t *testing.T) {
	// Point at a server that doesn't exist
	_, _, err := parseViaSophia("http://127.0.0.1:1", "u", "some text", "f.csv")
	if err == nil {
		t.Fatal("expected network error to propagate")
	}
}

func TestParseViaSophia_HTTPErrorPropagates(t *testing.T) {
	stub := stubSophia(t, nil, "", http.StatusInternalServerError)
	defer stub.Close()

	_, _, err := parseViaSophia(stub.URL, "u", "some text", "f.csv")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention 500, got %v", err)
	}
}

func TestParseViaSophia_AmountAsString(t *testing.T) {
	stub := stubSophia(t, []map[string]interface{}{
		{"date": "2026-04-01", "description": "Stringy", "amount": "12.34", "category": "Other", "type": "credit"},
	}, "m", 200)
	defer stub.Close()

	txns, _, err := parseViaSophia(stub.URL, "u", "text", "f.csv")
	if err != nil {
		t.Fatalf("parseViaSophia: %v", err)
	}
	if len(txns) != 1 || txns[0].Amount != 12.34 {
		t.Fatalf("expected amount 12.34 from string, got %+v", txns)
	}
}

func TestExtractTextForAI_CSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny.csv")
	content := "date,description,amount\n2026-04-01,Coffee,-4.50\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := extractTextForAI(path, ".csv")
	if err != nil {
		t.Fatalf("extractTextForAI: %v", err)
	}
	if got != content {
		t.Errorf("CSV text mismatch:\nwant %q\ngot  %q", content, got)
	}
}

func TestExtractTextForAI_XLSXRefused(t *testing.T) {
	_, err := extractTextForAI("anything.xlsx", ".xlsx")
	if err == nil {
		t.Fatal("expected xlsx to be refused")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "csv") {
		t.Errorf("expected error to suggest CSV, got %v", err)
	}
}

func TestEndToEnd_CSVThroughSophiaStub(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny.csv")
	if err := os.WriteFile(path, []byte("Date,Desc,Amount\n2026-04-01,Lunch,-12.00\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stub := stubSophia(t, []map[string]interface{}{
		{"date": "2026-04-01", "description": "Lunch", "amount": -12.00, "category": "Food", "type": "debit"},
	}, "test-model", 200)
	defer stub.Close()

	text, err := extractTextForAI(path, ".csv")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	txns, model, err := parseViaSophia(stub.URL, "user", text, path)
	if err != nil {
		t.Fatalf("parseViaSophia: %v", err)
	}
	if model != "test-model" {
		t.Errorf("model: got %q", model)
	}
	if len(txns) != 1 {
		t.Fatalf("expected 1 txn, got %d", len(txns))
	}
	if txns[0].Description != "Lunch" || txns[0].Amount != -12.00 {
		t.Errorf("unexpected transaction shape: %+v", txns[0])
	}
	if txns[0].FileSource != path {
		t.Errorf("file source not stamped: %q", txns[0].FileSource)
	}
}
