#!/usr/bin/env bash
set -euo pipefail

VOLUMES_FLAG=""
if [ "${1:-}" = "--volumes" ]; then
  VOLUMES_FLAG="-v"
  echo "Tearing down dev stack (including volumes)..."
else
  echo "Tearing down dev stack (preserving volumes)..."
fi

docker compose -f docker-compose.dev.yml down $VOLUMES_FLAG

echo "Done."
