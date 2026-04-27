# LocalFinance macOS Installer

Builds `dist/LocalFinance.app` — a self-contained macOS bundle wrapping the
single LocalFinance binary. The binary serves the React UI, owns all data
and AI logic, and embeds Postgres for first-launch use.

## Build

```
make installer
```

This runs `make build` (which also rebuilds the React UI) and lays out:

```
dist/LocalFinance.app/
  Contents/
    Info.plist
    MacOS/LocalFinance        # the single Go binary
    Resources/AppIcon.icns    # optional, only if installer/AppIcon.icns exists
```

## Run

```
open dist/LocalFinance.app
```

The first launch:
- Starts the embedded Postgres under `~/Library/Application Support/LocalFinance/postgres/`
- Probes for Ollama on the host. If not installed, the in-app wizard walks
  the user through `brew install ollama` and pulls the default model.
- Opens http://localhost:3001 in the user's browser.

## Data location

```
~/Library/Application Support/LocalFinance/
  postgres/                   # embedded DB data dir
  logs/                       # server + ollama logs
```

## Known limitations

- The `.app` is **unsigned**. On first launch macOS will refuse to open it
  with a Gatekeeper warning. Right-click → Open (and confirm) once and the
  warning never returns.
- No code signing or notarization yet — this is single-developer pre-release.
