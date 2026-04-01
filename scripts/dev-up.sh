#!/usr/bin/env bash
set -euo pipefail

echo "Starting LocalFinance dev stack..."
docker compose -f docker-compose.dev.yml up -d --build

echo "Waiting for services to be healthy..."

SERVICES="hermes:3000 thesaurus:8001 sophia:8002 logos:8003 iris:3001"
MAX_WAIT=120
ELAPSED=0

for svc_port in $SERVICES; do
  name="${svc_port%%:*}"
  port="${svc_port##*:}"
  echo -n "  Waiting for $name (port $port)..."

  while true; do
    if [ $ELAPSED -ge $MAX_WAIT ]; then
      echo " TIMEOUT after ${MAX_WAIT}s"
      echo "Check logs: docker compose -f docker-compose.dev.yml logs $name"
      exit 1
    fi

    health_path="/health"
    if [ "$name" = "iris" ]; then
      health_path="/api/health"
    fi

    if curl -sf "http://localhost:${port}${health_path}" > /dev/null 2>&1; then
      echo " ready"
      break
    fi

    sleep 2
    ELAPSED=$((ELAPSED + 2))
  done
done

echo ""
echo "All services healthy:"
echo "  Iris:      http://localhost:3001"
echo "  Hermes:    http://localhost:3000"
echo "  Thesaurus: http://localhost:8001"
echo "  Sophia:    http://localhost:8002"
echo "  Logos:     http://localhost:8003"
