# Hermes — API Gateway (port 3000)

Future routing and aggregation layer. Currently minimal — serves
health endpoint and a status dashboard.

## Build

make build-hermes             # native

## Key Files

- main.go — Gin router, health endpoint, HTML status dashboard
