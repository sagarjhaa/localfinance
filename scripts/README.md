# LocalFinance scripts

## `install.sh` — one-line installer for end users

```
curl -fsSL https://raw.githubusercontent.com/sagarjhaa/localfinance/main/scripts/install.sh | sh
```

Takes a fresh Mac to a running LocalFinance in roughly five minutes. The
script:

1. Confirms you're on macOS.
2. Installs Ollama if missing (`brew install ollama` when Homebrew is
   available, otherwise the official `ollama.com/install.sh`) and starts
   `ollama serve` if it isn't already running.
3. Builds `dist/LocalFinance.app` via `make installer`. If the script is
   piped from `curl` (i.e. not run from inside a clone) it first clones
   the repo into `~/.localfinance/src`. Once GitHub Releases are
   published this step will switch to a direct `.app` download.
4. Installs the bundle into `/Applications` (falling back to
   `~/Applications` if `/Applications` isn't writable). Re-installs are
   clean — the existing bundle is removed first.
5. Picks a recommended model from your Mac's RAM
   (`llama3.2:3b` / `gemma3:4b` / `qwen2.5:7b`) and starts an
   `ollama pull` in the background so the first-run wizard finds it
   ready.
6. Opens `LocalFinance.app`.

The script is idempotent — re-running it skips anything that's already
installed and re-installs the .app cleanly.

## Other scripts

- `dev-up.sh` / `dev-down.sh` — local Docker stack helpers used by
  `make dev-up` / `make dev-down`.
- `verify.sh` — health check used by `make verify`.
