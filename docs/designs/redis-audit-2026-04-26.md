# Redis Audit — 2026-04-26

**Trigger:** Issue P3 from eng review. Decide whether the macOS installer needs to ship Redis.

## Findings

- `services/thesaurus/config/config.go` defines `RedisConfig` (host/port/password/db) and reads `REDIS_*` env vars.
- **No Go file imports `go-redis`, `redis`, or `redigo`.** No service connects to Redis.
- `docker-compose.dev.yml` runs `redis:7-alpine` and other services declare `depends_on: redis`, but nothing actually talks to it.
- `go.mod` files have no Redis client dependency.

**Conclusion:** Redis is dead infrastructure. Currently unused.

## Decision

- **Phase 1 installer:** Do NOT ship Redis. The `.app` bundle skips it entirely.
- **Local dev:** Remove `redis` service from `docker-compose.dev.yml` and drop `depends_on: redis` from other services. Keep `RedisConfig` struct removal as a Phase 1.5 cleanup (low priority — it's harmless).

## Action items

1. Remove `redis` service block from `docker-compose.dev.yml` and any `depends_on` references — small focused commit.
2. When the installer launcher (Task 6.3) starts services, no Redis startup needed.
3. Phase 1.5 monolith refactor: delete `RedisConfig` struct + env-var defaults from `services/thesaurus/config/config.go`.
