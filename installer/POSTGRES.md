# Postgres in the LocalFinance bundle

## Decision: do NOT embed Postgres in Phase 1

The launcher resolves `initdb`, `pg_ctl`, `postgres`, and `psql` from the
runtime user's `PATH`. End users are expected to install Postgres themselves
once via Homebrew:

```bash
brew install postgresql@15
brew services start postgresql@15  # optional — launcher starts its own instance
```

The launcher then runs Postgres on **port 15432** (avoiding the default 5432
that brew may already have bound) with its own data dir under
`~/Library/Application Support/LocalFinance/pgdata/`.

## Why we didn't embed it

Three options were evaluated; all three failed acceptance criteria for Phase 1:

1. **Copy Homebrew Postgres binaries verbatim** — the binaries hard-code
   absolute install paths to dependent dylibs:
     ```
     /usr/local/opt/icu4c/lib/libicui18n.67.dylib
     /usr/local/opt/openssl@1.1/lib/libssl.1.1.dylib
     /usr/local/opt/krb5/lib/libgssapi_krb5.2.2.dylib
     ```
   On any Mac without those exact versions installed, the binary fails to
   load. Patching with `DYLD_FALLBACK_LIBRARY_PATH` doesn't work because SIP
   strips `DYLD_*` env vars from child processes when the parent binary is
   unsigned (and ours isn't, in Phase 1).

2. **`embeddedpostgres-binaries` from Maven** — these are dynamically linked
   too and exhibit intermittent loader issues on macOS 14+. Bundle size
   advantage is real, but the fragility is worse than the brew fallback.

3. **Build Postgres from source statically** — 2-3h initial build, fragile
   across Xcode SDK versions, and Apple disallows fully-static binaries on
   macOS. Out of Phase 1 budget.

The proper fix is `install_name_tool -change` on every dependent dylib in the
bundled Postgres binaries plus copying the matching dylib versions into
`Resources/bin/postgres/lib/`. That's tracked as a Phase 1.5 packaging task —
not worth blowing the Phase 1 budget on.

## Build-time opt-in (experimental)

If you want to attempt bundling anyway (e.g., for testing the install_name
work mentioned above), set the env var when building:

```bash
LOCALFINANCE_BUNDLE_POSTGRES=1 make installer
```

`installer/postgres-fetch.sh` will copy the local Homebrew Postgres into
`Resources/bin/postgres/`. The bundle will be ~110 MB larger and **will fail
to start on any Mac whose Homebrew prefix doesn't match exactly**. Use at
your own risk.

## Runtime layout (when `PATH` mode is used)

```
LocalFinance.app/Contents/Resources/bin/
├── thesaurus, sophia, logos, hermes  # Go binaries
├── iris-server.js, node              # Iris bundle + Node 18
└── (no postgres/ dir — launcher uses PATH)
```

## Bundle size impact

| Mode                          | Approx size |
|-------------------------------|-------------|
| Default (PATH-resolved PG)    | ~165 MB     |
| `LOCALFINANCE_BUNDLE_POSTGRES=1` | ~275 MB  |

## Known limitation surfaced in installer/README.md

* End user runs `brew install postgresql@15` once before first launch. The
  launcher prints a clear error pointing at that command if the binaries
  aren't on PATH.
