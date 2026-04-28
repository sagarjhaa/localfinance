# LocalFinance

**Your money. Your Mac. Your data stays put.**

A privacy-first personal finance app that runs entirely on your Mac.
Upload bank and credit-card statements, chat with your transactions in
plain English, and get rule-based spending insights — all powered by a
local LLM via [Ollama](https://ollama.com). No cloud, no telemetry,
no account-sharing with aggregators.

---

## For end users — install on a fresh Mac

You need:

- macOS 11 (Big Sur) or later
- 16 GB RAM recommended (8 GB works with smaller models)
- ~5 GB free disk space (binary + first model)

### Easiest install

```
curl -fsSL https://raw.githubusercontent.com/sagarjhaa/localfinance/main/scripts/install.sh | sh
```

Installs Ollama (if missing), builds & installs LocalFinance.app, pulls
a model, and launches it. ~5 minutes on a fresh Mac.

### Drag-and-drop install

Download `LocalFinance-0.2.0.dmg` from
[Releases](https://github.com/sagarjhaa/localfinance/releases) (when
published). Open it and drag `LocalFinance` into Applications.

### Manual install

#### 1. Install [Ollama](https://ollama.com)

LocalFinance does AI inference through Ollama. The .app's first-run
wizard walks you through this, but you can also do it ahead of time:

```
brew install ollama
```

Or download the signed installer from [ollama.com/download](https://ollama.com/download).
Start it once so it registers as a launch agent (subsequent boots are automatic).

#### 2. Get LocalFinance.app

**Option A — download the prebuilt .app** (when releases are published):

1. Grab the latest `LocalFinance.app.zip` from [Releases](https://github.com/sagarjhaa/localfinance/releases).
2. Unzip and drag `LocalFinance.app` into `/Applications`.
3. The bundle is currently unsigned — first launch needs a right-click
   to bypass Gatekeeper.

**Option B — build it yourself:**

```
git clone https://github.com/sagarjhaa/localfinance.git
cd localfinance
make installer
open dist/LocalFinance.app
```

#### 3. Launch

Right-click `LocalFinance.app` → **Open** (first time only). Your browser
opens at `http://localhost:3001` and the first-run wizard takes over.

The wizard:
- Detects whether Ollama is running. If not, points you at the installer.
- Recommends a model based on your Mac's RAM (3B / 4B-vision / 7B).
- Pulls the model with a live progress bar (~2-5 GB, takes 3-10 minutes).
- Drops you at the login screen.

#### 4. Sign in

Login is **pre-filled with default credentials** — just click **AUTHENTICATE**:

- Email: `local@localfinance.app`
- Password: `localfinance`

Change them in **Profile → Update Password** once you're in.

#### 5. Upload your first statement

Dashboard → drag a PDF or CSV onto the drop zone. The app:

1. Renders PDF pages to images (or reads CSV text)
2. Sends them to your local Ollama for AI parsing
3. Displays the extracted transactions
4. Generates a Month-in-Review with rule-based insights

A 1-page CSV finishes in seconds; a 10-page PDF can take 1-3 minutes
depending on your model.

### Where your data lives

```
~/Library/Application Support/LocalFinance/
  postgres/      ← embedded DB (when not using brew Postgres)
  uploads/       ← original statement files
  logs/          ← server logs
```

Nothing leaves your Mac. The only network traffic is your browser ↔
`localhost:3001` and `localhost:11434` (Ollama).

### Troubleshooting

| Symptom | Try |
|---|---|
| "App is damaged, can't be opened" | Right-click → Open. Bundle is unsigned. |
| Login screen says wrong password | Default is `localfinance`. If you changed it and forgot, delete `~/Library/Application Support/LocalFinance/` and start fresh. |
| Upload fails with "AI parse failed" | Check Ollama is running: `ollama list` should show installed models. |
| Upload works but says "0 transactions" | Try a different model in Profile → Settings (some 3B models miss small print). |
| Browser shows white page | Tail `~/Library/Application Support/LocalFinance/logs/` and report on GitHub. |

### Update the app

Replace `/Applications/LocalFinance.app` with the new bundle. Your data
in `~/Library/Application Support/LocalFinance/` survives.

---

## For developers

### Run from source

```
brew install ollama
ollama serve &
ollama pull gemma3:4b
make dev   # boots embedded Postgres on first run; ~80MB download, cached after
```

The `make dev` target runs `go run ./cmd/localfinance` against host
Postgres + host Ollama. No Docker required (compose is optional).

By default (no `DB_HOST` env), the binary boots an **embedded Postgres**
under `~/Library/Application Support/LocalFinance/postgres/` (downloads
~80 MB on first run, cached after). To use a host Postgres instead set
`DB_HOST=localhost DB_USER=postgres DB_PASSWORD=devpass DB_NAME=localfinance`.

### Tests

```
make test                # all unit tests
make eval-hallucination  # LLM hallucination eval (live Ollama, EVAL_OLLAMA=1)
```

### Build the .app

```
make installer           # → dist/LocalFinance.app
make installer-clean     # rm -rf dist/LocalFinance.app
make installer-run       # build + open
```

---

## Architecture

Single Go binary (`cmd/localfinance`) serves the React UI, owns all data
and AI logic, and embeds Postgres. Internal packages:

| Package | Responsibility |
|---|---|
| `internal/api` | HTTP routes (Gin) — auth, data, parse, AI, insights, monthreview, setup |
| `internal/auth` | JWT, Argon2id password hashing, default-user seed |
| `internal/data` | GORM models, repositories, embedded Postgres lifecycle |
| `internal/parse` | PDF/CSV ingest pipeline (vision → Ollama) |
| `internal/ai` | Ollama client + two-pass chat + insight narrator |
| `internal/insights` | Deterministic rules engine + narrator |
| `internal/monthreview` | Period summary cache |
| `internal/ollama` | Host Ollama probe + model pull SSE |
| `internal/webui` | `go:embed` of React build |
| `internal/postgres` | Embedded Postgres manager |

The React source lives at `services/iris/client/`. `make webui-build`
emits its `build/` into `internal/webui/dist/` for embedding.

Full design: `docs/superpowers/specs/2026-04-27-service-consolidation-design.md`.

---

## Help a friend test it

You're helping validate this for friends-and-family. The smoothest
demo:

1. Have them install Ollama via `brew install ollama` (one command).
2. Send them `LocalFinance.app` (zip the bundle, AirDrop or share via Drive).
3. They right-click → Open. Wizard does the rest.

If they hit Gatekeeper hard ("can't be opened because Apple cannot
check it for malicious software"), have them go to System Settings →
Privacy & Security → scroll down → click **Open Anyway** next to
LocalFinance.

What to ask them after a week:
- Did at least one insight tell you something you didn't already know?
- Did the app save you money (canceled a sub, caught a charge, etc.)?
- What broke / what felt wrong?

The validation gate (per `docs/designs/product-direction.md`) is **3-5
users × 4 weeks × ≥1 saved-money moment per user per month**. If we hit
it, we ship Phase 2 (Subscription Auditor). If we don't, we re-evaluate.

---

## License

MIT — see [LICENSE](LICENSE).

## Building from source

```
git clone https://github.com/sagarjhaa/localfinance
cd localfinance
make build         # → dist/localfinance (Go binary with embedded React UI)
make installer-run # → builds + opens dist/LocalFinance.app
```

Requires Go 1.21+, Node 18+, and `pdftotext` from poppler (`brew install poppler` on macOS).
A local Ollama install is needed at runtime — the app guides first-run install via
its setup wizard if not present.
