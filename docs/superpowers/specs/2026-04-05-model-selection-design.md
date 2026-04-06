# User-Configurable Chat Model Design Spec

**Date:** 2026-04-05
**Status:** Approved
**Goal:** Let users choose which Ollama model to use for chat, saved as a preference, shown in the UI.

---

## Data Model

### user_preferences table

| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | BeforeCreate hook |
| user_id | UUID (FK users, unique) | One preference row per user |
| chat_model | string | Default: "llama3.2:1b" |
| created_at | timestamp | Auto |
| updated_at | timestamp | Auto |

---

## API Endpoints

### Thesaurus — User-facing (auth required)

| Method | Path | Purpose |
|--------|------|---------|
| GET | /api/v1/preferences | Get user's preferences |
| PUT | /api/v1/preferences | Create or update preferences (upsert) |

### Thesaurus — Internal (no auth, service-to-service)

| Method | Path | Purpose |
|--------|------|---------|
| GET | /internal/preferences/:user_id | Get user's chat_model (Sophia calls this) |

### Sophia — User-facing (proxied via Iris)

| Method | Path | Purpose |
|--------|------|---------|
| GET | /api/v1/models | List installed Ollama models (calls Ollama /api/tags) |

---

## Flows

### List Available Models

1. Settings page mounts → GET /api/proxy/sophia/api/v1/models via Iris
2. Sophia calls Ollama GET /api/tags
3. Returns: `[{name: "llama3.2:1b", size: "1.3 GB"}, {name: "gemma3:1b", size: "815 MB"}, ...]`

### Get Current Preference

1. Settings page mounts → GET /api/proxy/thesaurus/api/v1/preferences via Iris
2. Thesaurus returns: `{chat_model: "llama3.2:1b"}` (or default if no row exists)

### Save Preference

1. User selects model from dropdown, clicks Save
2. PUT /api/proxy/thesaurus/api/v1/preferences via Iris
3. Body: `{chat_model: "gemma3:1b"}`
4. Thesaurus upserts the preference row

### Chat Uses Preference

1. User sends chat message → Sophia
2. Sophia calls Thesaurus GET /internal/preferences/:user_id
3. Gets `{chat_model: "gemma3:1b"}`
4. Sophia passes model name to all Ollama API calls (overrides config default)
5. Response includes `{model: "gemma3:1b"}` so UI can display it

### Chat UI Shows Model

1. Chat response includes `model` field
2. Status bar shows: "OLLAMA GEMMA3:1B ACTIVE" (from response, not health endpoint)
3. Loading indicator shows: "Querying Local LLM (gemma3:1b)..."

---

## Service Responsibilities

| Service | Role |
|---------|------|
| **Thesaurus** | Stores user_preferences, CRUD endpoints, internal lookup |
| **Sophia** | Lists installed models from Ollama, reads user preference per chat request, passes model to Ollama |
| **Iris** | Settings dropdown for model selection, Chat shows model from response |

---

## Settings UI

Add "AI Model" section to existing Settings page (above or below Category Rules):

- Section header: "Chat Model"
- Dropdown: populated from Sophia /api/v1/models
- Each option shows: model name + size (e.g., "llama3.2:1b (1.3 GB)")
- Current selection: from user preference (or default)
- Save button: persists to Thesaurus
- Success feedback: "Model saved" toast/message

---

## What This Does NOT Include

- Auto-downloading new models from Ollama registry
- Model benchmarking or comparison UI
- Per-conversation model selection (always uses user's saved preference)
- Model parameter tuning (temperature, context length)
