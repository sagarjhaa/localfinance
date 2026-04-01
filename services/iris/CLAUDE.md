# Iris — Frontend (port 3001)

React UI + Express proxy. ZERO business logic.

## NEVER do these in Iris:
- Parse files (CSV, PDF, Excel)
- Transform or process data
- Run business logic or categorization
- Store state beyond auth tokens
- Direct database access

## Build

make build-iris               # builds React client + bundles Node server

## Test

make test-iris                # unit tests (auth routes)
make test-e2e                 # integration tests

## Deploy

make deploy-iris              # build React + bundle + SCP + restart + verify

## Adding a Page

1. Create component in client/src/pages/NewPage.jsx
2. Add route in client/src/App.jsx
3. Add nav link in sidebar (copy from Dashboard.jsx sidebar)
4. API calls go through client/src/api/client.js

## Adding an API Proxy Route

1. All backend calls proxy through server/routes/proxy.js
2. Proxy strips /api/proxy/{service}/ prefix, forwards rest to backend
3. Never add business logic in proxy — just forward request/response

## Key Files

- server/index.js — Express entry point (Helmet, CORS, static files)
- server/routes/auth.js — proxies auth to Thesaurus
- server/routes/upload.js — proxies file uploads to Thesaurus
- server/routes/proxy.js — generic proxy to any backend service
- server/middleware/auth.js — validates JWT via Thesaurus
- client/src/App.jsx — React router
- client/src/api/client.js — Axios HTTP client with interceptors
- client/src/pages/ — one file per page (Dashboard, Chat, Settings, etc.)
