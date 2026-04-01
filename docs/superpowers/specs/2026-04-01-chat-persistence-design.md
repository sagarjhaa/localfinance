# Chat Persistence Design Spec

**Date:** 2026-04-01
**Status:** Approved
**Goal:** Persist chat conversations so users can see history, resume past chats, and start new ones.

---

## Data Model

### conversations table

| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | BeforeCreate hook |
| user_id | UUID (FK users) | Scoped to user |
| title | string | AI-generated after first message |
| created_at | timestamp | Auto |
| updated_at | timestamp | Auto |

### chat_messages table

| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | BeforeCreate hook |
| conversation_id | UUID (FK conversations) | |
| role | string | "user" or "assistant" |
| content | text | Message body |
| confidence | float (nullable) | Only for assistant messages |
| created_at | timestamp | Auto |

---

## API Endpoints

### Thesaurus — Internal (no auth, service-to-service)

| Method | Path | Purpose |
|--------|------|---------|
| POST | /internal/conversations | Create conversation (user_id, title) |
| PATCH | /internal/conversations/:id | Update title |
| POST | /internal/conversations/:id/messages | Add message (role, content, confidence) |

### Thesaurus — User-facing (auth required)

| Method | Path | Purpose |
|--------|------|---------|
| GET | /api/v1/conversations | List user's conversations (id, title, updated_at, last message preview) |
| GET | /api/v1/conversations/:id/messages | Get all messages for a conversation |
| DELETE | /api/v1/conversations/:id | Delete a conversation and its messages |

---

## Flow

### New Conversation

1. User types message in Chat UI (no active conversation)
2. Frontend sends POST to Sophia: `{question, user_id}`
3. Sophia answers the question (two-pass or conversational)
4. Sophia creates conversation via Thesaurus: `POST /internal/conversations`
5. Sophia saves both messages (user + assistant) via Thesaurus: `POST /internal/conversations/:id/messages`
6. Sophia generates a title via Ollama (short, 3-5 words from the question context)
7. Sophia updates conversation title via Thesaurus: `PATCH /internal/conversations/:id`
8. Sophia returns: `{answer, confidence, conversation_id, title}`

### Continue Conversation

1. User types message with active `conversation_id`
2. Frontend sends POST to Sophia: `{question, user_id, conversation_id}`
3. Sophia answers the question
4. Sophia saves both messages to Thesaurus
5. Returns: `{answer, confidence, conversation_id}`

### Load Conversation List

1. Chat.jsx mounts
2. GET `/api/v1/conversations` via Iris proxy → Thesaurus
3. Returns: `[{id, title, updated_at, preview}]` ordered by updated_at desc
4. Sidebar renders conversation list

### Resume Conversation

1. User clicks conversation in sidebar
2. GET `/api/v1/conversations/:id/messages` via Iris proxy → Thesaurus
3. Returns: `[{id, role, content, confidence, created_at}]` ordered by created_at asc
4. Chat feed populates with history
5. User can continue chatting (sends with conversation_id)

### Title Generation

After first exchange, Sophia makes a lightweight Ollama call:
```
Prompt: "Generate a short title (3-5 words) for this conversation. 
The user asked: '{question}'. 
Respond with ONLY the title, no quotes or punctuation."
```
This runs async — doesn't block the response. Title updates in the sidebar on next load.

---

## Service Responsibilities

| Service | Role |
|---------|------|
| **Thesaurus** | CRUD for conversations and messages tables. Owns the data. |
| **Sophia** | Orchestrates: answers question, then saves conversation + messages to Thesaurus. Generates titles. |
| **Iris** | Displays conversation list in sidebar, loads messages, sends with conversation_id. Zero logic. |

---

## Chat UI Changes

### Sidebar (left panel)
- "New Chat" button at top of chat area
- Conversation list below: title + relative date ("2 hours ago", "Yesterday")
- Active conversation highlighted
- Click to load, shows messages in feed

### Chat Feed
- On mount: if conversations exist, load the most recent one
- If no conversations: show empty state with "Ask me anything"
- After first message in new chat: conversation appears in sidebar
- Messages persist across page refreshes and sessions

### Data Sent to Sophia
- First message: `{question, user_id}` — no conversation_id
- Subsequent messages: `{question, user_id, conversation_id}`
- Sophia returns: `{answer, confidence, conversation_id, title?, sources?}`

---

## What This Does NOT Include

- Message editing or deletion (individual messages)
- Chat search
- Conversation sharing or export
- Message reactions or ratings
- Chat context window (sending past messages to Ollama for context) — future enhancement
