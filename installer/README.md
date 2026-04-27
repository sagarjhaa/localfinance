# LocalFinance macOS Installer

A self-contained `LocalFinance.app` bundle that runs the entire LocalFinance
stack (Postgres + 4 Go services + Iris UI) on a user's Mac. Double-click and go.

## Architecture

```
LocalFinance.app/
├── Contents/
│   ├── Info.plist
│   ├── MacOS/
│   │   └── launcher          # Single Go binary: starts/stops everything
│   └── Resources/
│       ├── bin/
│       │   ├── thesaurus     # Go binary, port 8001
│       │   ├── sophia        # Go binary, port 8002 (talks to Ollama)
│       │   ├── logos         # Go binary, port 8003
│       │   ├── hermes        # Go binary, port 3000
│       │   ├── iris-server.js# Bundled Express server
│       │   ├── node          # Bundled Node 18 binary
│       │   └── postgres/     # initdb, pg_ctl, postgres, psql + libs
│       └── iris-static/      # React build output (served by Iris)
```

The launcher:

1. Probes Ollama at `http://127.0.0.1:11434` — exits with a brew install hint if missing.
2. Verifies the chosen chat model (`--chat-model`, default `llama3.1:8b`) is pulled.
3. Initdbs and starts an embedded Postgres on port `15432` (avoids conflict with `:5432`).
4. Spawns Thesaurus → Sophia → Logos → Hermes → Iris (each waits for the previous to be healthy).
5. Opens `http://localhost:3001` in the default browser.
6. On Ctrl-C / SIGTERM, gracefully stops all children (10s grace, then SIGKILL).

Logs land in `~/Library/Application Support/LocalFinance/logs/{service}.log`.
The Postgres data dir lives at `~/Library/Application Support/LocalFinance/pgdata/`.

## For Developers — Building the Bundle

```bash
make installer            # Builds dist/LocalFinance.app (native arch)
make installer-test       # Runs launcher unit tests
make installer-run        # Builds + runs with --skip-browser (testing)
make installer-clean      # rm -rf dist/LocalFinance.app dist/cache
```

For a universal2 (arm64 + amd64) build:

```bash
UNIVERSAL2=1 bash installer/build.sh
```

Re-running `make installer` after edits is incremental — `make build` and
`make build-iris` skip work that's already up to date.

## For End Users — Installing

1. Install Ollama (one time):
   ```bash
   brew install ollama
   ollama serve &
   ollama pull llama3.1:8b
   ```
2. Drag `LocalFinance.app` into `/Applications`.
3. Double-click. The first launch initializes Postgres (~5s) and pulls down logs.
   Your browser opens to `http://localhost:3001`.
4. To quit, close the launcher window or `kill` the process — children shut down cleanly.

## RAM tier → recommended chat model

| Mac RAM | Recommended `--chat-model`        |
|---------|-----------------------------------|
| 8 GB    | `llama3.2:3b`                     |
| 16 GB   | `llama3.1:8b` (default)           |
| 32 GB+  | `qwen2.5:14b`                     |

Pass via CLI when launching from a terminal:

```bash
/Applications/LocalFinance.app/Contents/MacOS/launcher --chat-model llama3.2:3b
```

## CLI flags

| Flag             | Default                          | Purpose                                      |
|------------------|----------------------------------|----------------------------------------------|
| `--chat-model`   | `llama3.1:8b`                    | Ollama model name passed to Sophia           |
| `--skip-browser` | false                            | Don't auto-open the browser (for testing)    |
| `--data-dir`     | `~/Library/Application Support/LocalFinance` | Override the data + logs location |
| `--ollama-url`   | `http://127.0.0.1:11434`         | Where to find Ollama                         |

## Troubleshooting

* **"Cannot reach Ollama"** — `brew install ollama && ollama serve`.
* **"model X is not pulled"** — `ollama pull <model>`.
* **Port 15432 in use** — kill the stray Postgres: `lsof -i :15432`.
* **Crash logs** — check `~/Library/Application Support/LocalFinance/logs/*.log`.

## Known limitations (Phase 1)

* Crash recovery is best-effort: if a child dies, the launcher kills the rest and exits.
  No restart loop. Tracked for the Phase 1.5 monolith refactor.
* The bundle is **not signed or notarized**. macOS Gatekeeper will block first launch
  until the user right-clicks → Open. Signing is a Phase 1.5 packaging task.
* Ollama is **not bundled**. Detected at startup; user is expected to `brew install` it.
