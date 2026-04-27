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

## Distribution

To package the bundle as a drag-to-Applications disk image:

```
make dmg
```

This runs `make installer` to refresh `dist/LocalFinance.app`, stages
it in a temp dir alongside a symlink to `/Applications`, and uses
`hdiutil create -format UDZO` to produce
`dist/LocalFinance-<version>.dmg`. The version comes from
`CFBundleVersion` in the bundle's `Info.plist` (read via PlistBuddy).

The resulting `.dmg` is **unsigned** — same Gatekeeper caveats as the
raw `.app`. It contains:

- `LocalFinance.app` — the bundle
- `Applications` — symlink, so the volume window shows a drag target

Open it with `open dist/LocalFinance-<version>.dmg` to verify the
window mounts and shows both icons.

A v2 polish pass will add a background image + AppleScript-driven
window layout (icon size and positions) so the volume opens looking
like a proper installer. For now drag-and-drop works without it.

## Known limitations

- The `.app` is **unsigned**. On first launch macOS will refuse to open it
  with a Gatekeeper warning. Right-click → Open (and confirm) once and the
  warning never returns.
- No code signing or notarization yet — this is single-developer pre-release.
