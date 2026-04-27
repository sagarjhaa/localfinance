# Insight Hallucination Eval (Phase 0 Gate)

This harness empirically measures whether a candidate local Ollama model can
summarize transactions without hallucinating numbers, merchant names, or dates.
Phase 1 features depend on a passing run.

## What it does

1. Loads 30 synthetic transactions from `fixtures.json`.
2. Loads 15 prompts (10 analytical + 5 month-in-review) from `prompts.json`.
3. For each candidate model, sends each prompt with the transactions as context
   to a local Ollama (`http://localhost:11434/api/generate`,
   `OLLAMA_KEEP_ALIVE=24h`, `num_ctx=4096`, `temperature=0`).
4. Auto-grades responses (every digit-sequence must appear in fixtures or be a
   pre-computed roll-up; tracks expected merchant/date recall).
5. Writes one `results-<model>-<timestamp>.json` per model and prints a summary
   table.

The auto-grader is conservative — it flags suspicious numbers but does NOT
replace human grading. Use the JSON output to fill in `results-template.md`.

## Models under test

- `llama3.1:8b`
- `qwen2.5:7b`
- `qwen2.5:14b`

Override with `EVAL_MODELS=foo:8b,bar:14b` if needed.

## Run it

Prerequisites: Ollama running locally.

```bash
ollama pull llama3.1:8b
ollama pull qwen2.5:7b
ollama pull qwen2.5:14b

EVAL_OLLAMA=1 make eval-hallucination
```

Expected runtime: ~5-10 minutes total (45 calls; the 14b model dominates).

The test is gated behind:

- the `eval` Go build tag (so `make test` ignores it)
- `EVAL_OLLAMA=1` (so it skips cleanly if env is not set)

To run directly without `make`:

```bash
EVAL_OLLAMA=1 go test -tags=eval -run HallucinationEval -v -timeout 30m \
  ./services/sophia/ai/...
```

## After the run

1. Open each `results-<model>-<timestamp>.json` next to `fixtures.json`.
2. For every (prompt, model) pair, fill one row in
   [`results-template.md`](./results-template.md) with `numeric_ok`,
   `merchant_ok`, `date_ok` (Y/N) plus notes.
3. Compute aggregate percentages per model.
4. Compare against the gate.

## Gate (success thresholds)

A model **passes** if and only if:

- numeric accuracy ≥ **95%**
- merchant accuracy ≥ **90%**
- date accuracy ≥ **95%**

If no candidate passes, Phase 1 insight features are blocked until either the
prompts/context are improved or a different model is chosen.

## Files

- `fixtures.json` — 30 transactions over a 90-day window ending 2026-04-26.
- `prompts.json` — 15 prompts with `expected_facts` for grading.
- `results-template.md` — manual grading sheet.
- `results-<model>-<timestamp>.json` — per-model run output (gitignored, write
  on demand).
- `../services/sophia/ai/hallucination_eval_test.go` — the harness.
