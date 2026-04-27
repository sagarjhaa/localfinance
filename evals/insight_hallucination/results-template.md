# Insight Hallucination Eval — Manual Grading Template

Fill one row per (prompt_id, model). Use the matching `results-<model>-<timestamp>.json`
output as your source of truth — open it side-by-side with `fixtures.json` and verify
each numeric, merchant, and date claim by hand.

## Success thresholds (gate for Phase 1)

- numeric accuracy >= 95%
- merchant accuracy >= 90%
- date accuracy >= 95%

A model that misses any threshold fails the gate.

## Grading rubric

- **numeric_ok (Y/N)** — Every digit-sequence in the response appears in fixtures
  OR is a correctly computed roll-up (sum, count, average) over fixture rows.
  N if the model invented a number, miscalculated a sum, or hallucinated an amount.
- **merchant_ok (Y/N)** — Every merchant named in the response appears in
  `fixtures.json` descriptions. N if the model invented a merchant or
  mis-attributed a charge.
- **date_ok (Y/N)** — Every date or month/period named in the response is consistent
  with the fixture window (2026-01-26 to 2026-04-26) and matches a fixture row when
  the response cites a specific transaction. N if the model invented a date or
  attributed the wrong month.

## Grading sheet

Run: `<paste timestamp>`
Models tested: llama3.1:8b, qwen2.5:7b, qwen2.5:14b

| prompt_id | model        | numeric_ok | merchant_ok | date_ok | notes |
|-----------|--------------|------------|-------------|---------|-------|
| p01       | llama3.1:8b  |            |             |         |       |
| p01       | qwen2.5:7b   |            |             |         |       |
| p01       | qwen2.5:14b  |            |             |         |       |
| p02       | llama3.1:8b  |            |             |         |       |
| p02       | qwen2.5:7b   |            |             |         |       |
| p02       | qwen2.5:14b  |            |             |         |       |
| p03       | llama3.1:8b  |            |             |         |       |
| p03       | qwen2.5:7b   |            |             |         |       |
| p03       | qwen2.5:14b  |            |             |         |       |
| p04       | llama3.1:8b  |            |             |         |       |
| p04       | qwen2.5:7b   |            |             |         |       |
| p04       | qwen2.5:14b  |            |             |         |       |
| p05       | llama3.1:8b  |            |             |         |       |
| p05       | qwen2.5:7b   |            |             |         |       |
| p05       | qwen2.5:14b  |            |             |         |       |
| p06       | llama3.1:8b  |            |             |         |       |
| p06       | qwen2.5:7b   |            |             |         |       |
| p06       | qwen2.5:14b  |            |             |         |       |
| p07       | llama3.1:8b  |            |             |         |       |
| p07       | qwen2.5:7b   |            |             |         |       |
| p07       | qwen2.5:14b  |            |             |         |       |
| p08       | llama3.1:8b  |            |             |         |       |
| p08       | qwen2.5:7b   |            |             |         |       |
| p08       | qwen2.5:14b  |            |             |         |       |
| p09       | llama3.1:8b  |            |             |         |       |
| p09       | qwen2.5:7b   |            |             |         |       |
| p09       | qwen2.5:14b  |            |             |         |       |
| p10       | llama3.1:8b  |            |             |         |       |
| p10       | qwen2.5:7b   |            |             |         |       |
| p10       | qwen2.5:14b  |            |             |         |       |
| p11       | llama3.1:8b  |            |             |         |       |
| p11       | qwen2.5:7b   |            |             |         |       |
| p11       | qwen2.5:14b  |            |             |         |       |
| p12       | llama3.1:8b  |            |             |         |       |
| p12       | qwen2.5:7b   |            |             |         |       |
| p12       | qwen2.5:14b  |            |             |         |       |
| p13       | llama3.1:8b  |            |             |         |       |
| p13       | qwen2.5:7b   |            |             |         |       |
| p13       | qwen2.5:14b  |            |             |         |       |
| p14       | llama3.1:8b  |            |             |         |       |
| p14       | qwen2.5:7b   |            |             |         |       |
| p14       | qwen2.5:14b  |            |             |         |       |
| p15       | llama3.1:8b  |            |             |         |       |
| p15       | qwen2.5:7b   |            |             |         |       |
| p15       | qwen2.5:14b  |            |             |         |       |

## Aggregate scores (fill after grading)

| model        | numeric % | merchant % | date % | passes gate? |
|--------------|-----------|------------|--------|--------------|
| llama3.1:8b  |           |            |        |              |
| qwen2.5:7b   |           |            |        |              |
| qwen2.5:14b  |           |            |        |              |

## Decision

Winner: `<model>`
Rationale: `<one paragraph>`
