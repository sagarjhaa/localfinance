# Hermes — API Gateway (port 3000)

Future routing and aggregation layer. Currently minimal — serves
health endpoint and a status dashboard.

## Build

make build-hermes             # native
make build-arm64-hermes       # cross-compile for Jetson

## Deploy

make deploy-hermes            # build ARM64 + SCP + restart + verify

## Key Files

- main.go — Gin router, health endpoint, HTML status dashboard
