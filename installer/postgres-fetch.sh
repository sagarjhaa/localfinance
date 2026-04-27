#!/usr/bin/env bash
# installer/postgres-fetch.sh
# Acquire a self-contained Postgres distribution and lay it out at $1 such that
# $1/bin/{initdb,pg_ctl,postgres,psql} all exist.
#
# Strategy:
#   1. If a Homebrew Postgres is installed locally (Apple Silicon or Intel),
#      symlink-copy its binaries — fast, license-clean, and stable.
#   2. Otherwise leave $1 absent and exit non-zero. The launcher will fall back
#      to PATH lookup at runtime; the README documents the brew install step.
#
# Why no tarball download? Truly static Postgres tarballs for macOS are scarce.
# embeddedpostgres-binaries from Maven ship dynamically-linked binaries that
# are reliable on Linux but flaky on recent macOS. Documented decision —
# see installer/POSTGRES.md.

set -euo pipefail

DEST="${1:-}"
if [[ -z "${DEST}" ]]; then
  echo "usage: postgres-fetch.sh <dest-dir>" >&2
  exit 2
fi

log() { printf "\033[1;36m[pg-fetch]\033[0m %s\n" "$*"; }

# Bundling decision:
#   The Homebrew Postgres binaries hard-code dylib paths (icu4c, openssl@1.1,
#   krb5) into install_name. These paths often don't exist on a fresh user Mac
#   AND DYLD_FALLBACK_LIBRARY_PATH is stripped by SIP when an unsigned launcher
#   re-execs a binary, so we can't safely paper over it with env vars.
#
#   Robust fix is `install_name_tool -change` on every dependent dylib —
#   tracked for Phase 1.5 packaging task. For Phase 1 we explicitly fall back
#   to "user installs Postgres via brew" (the launcher resolves binaries via
#   PATH at runtime). See installer/POSTGRES.md.
if [[ "${LOCALFINANCE_BUNDLE_POSTGRES:-0}" != "1" ]]; then
  echo "Skipping Postgres bundling (set LOCALFINANCE_BUNDLE_POSTGRES=1 to attempt)." >&2
  echo "  -> launcher will resolve pg_ctl/initdb/postgres from PATH at runtime." >&2
  echo "  -> end users must run: brew install postgresql@15" >&2
  exit 0
fi

# Look for a brew postgres install we can copy from.
BREW_PREFIXES=(
  "/opt/homebrew/opt/postgresql@15"
  "/opt/homebrew/opt/postgresql@16"
  "/opt/homebrew/opt/postgresql@14"
  "/usr/local/opt/postgresql@15"
  "/usr/local/opt/postgresql@16"
  "/usr/local/opt/postgresql@14"
  "/opt/homebrew/opt/postgresql"
  "/usr/local/opt/postgresql"
)

SRC=""
for p in "${BREW_PREFIXES[@]}"; do
  if [[ -x "${p}/bin/initdb" && -x "${p}/bin/postgres" ]]; then
    SRC="${p}"
    break
  fi
done

if [[ -z "${SRC}" ]]; then
  echo "No bundled Postgres source found." >&2
  echo "  -> launcher will use system PATH (user must run: brew install postgresql@15)" >&2
  exit 1
fi

log "Copying Postgres from ${SRC}"
rm -rf "${DEST}"
mkdir -p "${DEST}/bin" "${DEST}/lib" "${DEST}/share"

# Copy just the binaries we actually need. Postgres needs share/postgresql for
# postgres.conf templates and timezone data; lib for libpq + libcrypto.
for tool in initdb pg_ctl postgres psql; do
  if [[ -x "${SRC}/bin/${tool}" ]]; then
    cp -f "${SRC}/bin/${tool}" "${DEST}/bin/${tool}"
  fi
done

# Share: contains the version-specific data dir templates initdb needs.
if [[ -d "${SRC}/share" ]]; then
  ( cd "${SRC}/share" && tar cf - . ) | ( cd "${DEST}/share" && tar xf - )
fi

# Lib: dylibs the binaries link against. We copy the whole lib dir to be safe.
# (Bundle size impact is documented in installer/POSTGRES.md.)
if [[ -d "${SRC}/lib" ]]; then
  ( cd "${SRC}/lib" && tar cf - . ) | ( cd "${DEST}/lib" && tar xf - ) 2>/dev/null || true
fi

log "Postgres bundled at ${DEST}"
du -sh "${DEST}" | awk '{ printf "  size: %s\n", $1 }'
