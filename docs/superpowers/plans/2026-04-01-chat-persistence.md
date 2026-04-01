# Chat Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist chat conversations in PostgreSQL so users can see history, resume past chats, and start new ones.

**Architecture:** Thesaurus gets two new tables (conversations, chat_messages) with CRUD endpoints. Sophia orchestrates saving after each exchange and generates titles. Chat.jsx shows conversation list in sidebar and loads history on click.

**Tech Stack:** Go/Gin/GORM (Thesaurus, Sophia), React (Iris), PostgreSQL

**Spec:** `docs/superpowers/specs/2026-04-01-chat-persistence-design.md`

---

## File Map

### New Files
| File | Responsibility |
|---|---|
| `services/thesaurus/models/conversation.go` | Conversation + ChatMessage GORM models |
| `services/thesaurus/api/handlers/conversation_handler.go` | CRUD handlers for conversations and messages |

### Modified Files
| File | Change |
|---|---|
| `services/thesaurus/database/database.go` | Add Conversation, ChatMessage to AutoMigrate |
| `services/thesaurus/api/routes.go` | Register conversation endpoints (internal + protected) |
| `services/sophia/models/models.go` | Add ConversationID to FinancialQuery and AIResponse |
| `services/sophia/ai/service.go` | Save conversations/messages to Thesaurus after each exchange, generate titles |
| `services/sophia/api/handlers/chat_handler.go` | Pass conversation_id through, return it in response |
| `services/iris/client/src/pages/Chat.jsx` | Conversation list in sidebar, load history, send conversation_id |

---

## Task 1: Thesaurus — Conversation and ChatMessage Models

**Files:**
- Create: `services/thesaurus/models/conversation.go`
- Modify: `services/thesaurus/database/database.go`

- [ ] **Step 1: Create conversation models**

Create `services/thesaurus/models/conversation.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Conversation struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Title     string         `gorm:"default:'New Chat'" json:"title"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Messages  []ChatMessage  `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type ChatMessage struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ConversationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"conversation_id"`
	Role           string     `gorm:"not null" json:"role"`
	Content        string     `gorm:"type:text;not null" json:"content"`
	Confidence     *float64   `json:"confidence,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
```

- [ ] **Step 2: Add to AutoMigrate**

In `services/thesaurus/database/database.go`, add to the AutoMigrate call (after `&models.CategoryRule{}`):

```go
		&models.Conversation{},
		&models.ChatMessage{},
```

- [ ] **Step 3: Verify build**

Run: `cd services/thesaurus && go build ./...`
Expected: Clean compile, no errors

- [ ] **Step 4: Commit**

```bash
git add services/thesaurus/models/conversation.go services/thesaurus/database/database.go
git commit -m "feat(thesaurus): add Conversation and ChatMessage models with auto-migration"
```

---

## Task 2: Thesaurus — Conversation Handler and Routes

**Files:**
- Create: `services/thesaurus/api/handlers/conversation_handler.go`
- Modify: `services/thesaurus/api/routes.go`

- [ ] **Step 1: Create conversation handler**

Create `services/thesaurus/api/handlers/conversation_handler.go`:

```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/gorm"
)

type ConversationHandler struct {
	db *gorm.DB
}

func NewConversationHandler(db *gorm.DB) *ConversationHandler {
	return &ConversationHandler{db: db}
}

// POST /internal/conversations — create a new conversation (called by Sophia)
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Title  string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	title := req.Title
	if title == "" {
		title = "New Chat"
	}

	conv := models.Conversation{
		UserID: userID,
		Title:  title,
	}
	if err := h.db.Create(&conv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, conv)
}

// PATCH /internal/conversations/:id — update conversation title (called by Sophia)
func (h *ConversationHandler) UpdateConversation(c *gin.Context) {
	id := c.Param("id")
	convID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.db.Model(&models.Conversation{}).Where("id = ?", convID).Update("title", req.Title)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": convID, "title": req.Title})
}

// POST /internal/conversations/:id/messages — add a message (called by Sophia)
func (h *ConversationHandler) AddMessage(c *gin.Context) {
	id := c.Param("id")
	convID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	var req struct {
		Role       string   `json:"role" binding:"required"`
		Content    string   `json:"content" binding:"required"`
		Confidence *float64 `json:"confidence,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := models.ChatMessage{
		ConversationID: convID,
		Role:           req.Role,
		Content:        req.Content,
		Confidence:     req.Confidence,
	}
	if err := h.db.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Touch conversation updated_at
	h.db.Model(&models.Conversation{}).Where("id = ?", convID).Update("updated_at", msg.CreatedAt)

	c.JSON(http.StatusCreated, msg)
}

// GET /api/v1/conversations — list user's conversations (auth required)
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	type ConversationWithPreview struct {
		models.Conversation
		Preview string `json:"preview"`
	}

	var conversations []models.Conversation
	if err := h.db.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&conversations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get last message preview for each conversation
	var results []ConversationWithPreview
	for _, conv := range conversations {
		var lastMsg models.ChatMessage
		h.db.Where("conversation_id = ? AND role = ?", conv.ID, "assistant").
			Order("created_at DESC").
			First(&lastMsg)

		preview := ""
		if lastMsg.Content != "" {
			preview = lastMsg.Content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
		}

		results = append(results, ConversationWithPreview{
			Conversation: conv,
			Preview:      preview,
		})
	}

	c.JSON(http.StatusOK, results)
}

// GET /api/v1/conversations/:id/messages — get all messages for a conversation (auth required)
func (h *ConversationHandler) GetMessages(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	convID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	// Verify conversation belongs to user
	var conv models.Conversation
	if err := h.db.Where("id = ? AND user_id = ?", convID, userID).First(&conv).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	var messages []models.ChatMessage
	if err := h.db.Where("conversation_id = ?", convID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// DELETE /api/v1/conversations/:id — delete a conversation (auth required)
func (h *ConversationHandler) DeleteConversation(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	convID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	// Verify ownership and soft-delete
	result := h.db.Where("id = ? AND user_id = ?", convID, userID).Delete(&models.Conversation{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	// Delete messages too
	h.db.Where("conversation_id = ?", convID).Delete(&models.ChatMessage{})

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
```

- [ ] **Step 2: Register routes**

In `services/thesaurus/api/routes.go`, add after the existing handler initializations (line 19):

```go
	conversationHandler := handlers.NewConversationHandler(db)
```

Add internal routes inside the `internal` group (after line 60):

```go
				internal.POST("/conversations", conversationHandler.CreateConversation)
				internal.PATCH("/conversations/:id", conversationHandler.UpdateConversation)
				internal.POST("/conversations/:id/messages", conversationHandler.AddMessage)
```

Add protected routes inside the `protected` group (after the budgets block, before the closing `}`):

```go
			// Conversations
			conversations := protected.Group("/conversations")
			{
				conversations.GET("/", conversationHandler.ListConversations)
				conversations.GET("/:id/messages", conversationHandler.GetMessages)
				conversations.DELETE("/:id", conversationHandler.DeleteConversation)
			}
```

- [ ] **Step 3: Verify build**

Run: `cd services/thesaurus && go build ./...`
Expected: Clean compile

- [ ] **Step 4: Commit**

```bash
git add services/thesaurus/api/handlers/conversation_handler.go services/thesaurus/api/routes.go
git commit -m "feat(thesaurus): add conversation CRUD endpoints for chat persistence"
```

---

## Task 3: Sophia — Save Conversations and Generate Titles

**Files:**
- Modify: `services/sophia/models/models.go`
- Modify: `services/sophia/ai/service.go`
- Modify: `services/sophia/api/handlers/chat_handler.go`

- [ ] **Step 1: Update Sophia models**

In `services/sophia/models/models.go`, add `ConversationID` to `FinancialQuery`:

```go
type FinancialQuery struct {
	UserID         string `json:"user_id" binding:"required"`
	Question       string `json:"question" binding:"required"`
	ConversationID string `json:"conversation_id,omitempty"`
	Context        string `json:"context,omitempty"`
}
```

Add `ConversationID` and `Title` to `AIResponse`:

```go
type AIResponse struct {
	Answer         string             `json:"answer"`
	Confidence     float64            `json:"confidence"`
	ConversationID string             `json:"conversation_id,omitempty"`
	Title          string             `json:"title,omitempty"`
	Sources        []TransactionRef   `json:"sources,omitempty"`
	Insights       []FinancialInsight `json:"insights,omitempty"`
	GeneratedAt    time.Time          `json:"generated_at"`
}
```

- [ ] **Step 2: Add conversation persistence methods to ai/service.go**

Add these methods to `services/sophia/ai/service.go` (at the end of the file, before any helper functions):

```go
// createConversation creates a new conversation in Thesaurus
func (s *Service) createConversation(userID string) (string, error) {
	body, _ := json.Marshal(map[string]string{"user_id": userID, "title": "New Chat"})
	resp, err := s.httpClient.Post(
		s.thesaurusURL+"/api/v1/internal/conversations",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}

// saveMessage saves a chat message to Thesaurus
func (s *Service) saveMessage(conversationID, role, content string, confidence *float64) error {
	payload := map[string]interface{}{
		"role":    role,
		"content": content,
	}
	if confidence != nil {
		payload["confidence"] = *confidence
	}
	body, _ := json.Marshal(payload)

	resp, err := s.httpClient.Post(
		fmt.Sprintf("%s/api/v1/internal/conversations/%s/messages", s.thesaurusURL, conversationID),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// updateConversationTitle generates and saves a title for the conversation
func (s *Service) updateConversationTitle(conversationID, question string) {
	prompt := fmt.Sprintf(`Generate a short title (3-5 words) for a conversation that started with this question: "%s"
Respond with ONLY the title, no quotes, no punctuation, no explanation.`, question)

	title, err := s.queryOllamaRaw(prompt, 0.7)
	if err != nil {
		log.Printf("Failed to generate title: %v", err)
		return
	}

	title = strings.TrimSpace(title)
	if len(title) > 100 {
		title = title[:100]
	}

	body, _ := json.Marshal(map[string]string{"title": title})
	req, _ := http.NewRequest("PATCH",
		fmt.Sprintf("%s/api/v1/internal/conversations/%s", s.thesaurusURL, conversationID),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.httpClient.Do(req)
}
```

- [ ] **Step 3: Update AnswerFinancialQuery to save conversations**

In `services/sophia/ai/service.go`, modify `AnswerFinancialQuery` to accept and return conversation_id. Replace the existing function (the one that starts with the `isConversational` check) with:

```go
func (s *Service) AnswerFinancialQuery(query models.FinancialQuery) (models.AIResponse, error) {
	// Determine or create conversation
	conversationID := query.ConversationID
	isNewConversation := conversationID == ""

	if isNewConversation && s.thesaurusURL != "" {
		var err error
		conversationID, err = s.createConversation(query.UserID)
		if err != nil {
			log.Printf("Failed to create conversation: %v", err)
			// Continue without persistence — don't block the response
		}
	}

	// Save user message
	if conversationID != "" && s.thesaurusURL != "" {
		s.saveMessage(conversationID, "user", query.Question, nil)
	}

	// Check if this is a conversational message (greeting, general question)
	var response models.AIResponse
	var err error

	if isConversational(query.Question) {
		response, err = s.handleConversational(query.Question)
	} else {
		// Two-pass financial query flow
		response, err = s.handleFinancialQuery(query)
	}

	if err != nil {
		return models.AIResponse{}, err
	}

	// Save assistant message
	if conversationID != "" && s.thesaurusURL != "" {
		s.saveMessage(conversationID, "assistant", response.Answer, &response.Confidence)
	}

	// Generate title async for new conversations
	if isNewConversation && conversationID != "" && s.thesaurusURL != "" {
		go s.updateConversationTitle(conversationID, query.Question)
	}

	response.ConversationID = conversationID
	return response, nil
}
```

Then rename the old financial query logic to `handleFinancialQuery`. Find the code that was previously in AnswerFinancialQuery after the conversational check (the Pass 1/Pass 2 flow) and wrap it in a new method:

```go
func (s *Service) handleFinancialQuery(query models.FinancialQuery) (models.AIResponse, error) {
	// Pass 1: Parse user question into structured query intent
	intent, err := s.parseQueryIntent(query.Question)
	if err != nil {
		log.Printf("Pass 1 failed, falling back to generic: %v", err)
		intent = &models.QueryIntent{NeedsSearch: true, Limit: 50}
	}

	log.Printf("Query intent: %+v", intent)

	var transactions []models.TransactionRef
	var summaryItems []models.SpendingSummaryItem

	if intent.NeedsSummary {
		summaryItems, err = s.fetchSpendingSummary(query.UserID, intent.StartDate, intent.EndDate)
		if err != nil {
			log.Printf("Summary fetch failed: %v", err)
		}
	}

	if intent.NeedsSearch || !intent.NeedsSummary {
		transactions, err = s.searchTransactions(query.UserID, intent)
		if err != nil {
			log.Printf("Search fetch failed: %v", err)
		}
	}

	if len(transactions) == 0 && len(summaryItems) == 0 {
		transactions, _ = s.getUserTransactions(query.UserID, 50)
	}

	dataContext := s.buildDataContext(transactions, summaryItems)

	answer, err := s.generateAnswer(query.Question, dataContext)
	if err != nil {
		return models.AIResponse{}, fmt.Errorf("failed to generate answer: %w", err)
	}

	return models.AIResponse{
		Answer:     answer,
		Confidence: s.calculateConfidence(transactions, summaryItems),
		Sources:    transactions,
	}, nil
}
```

- [ ] **Step 4: Verify build**

Run: `cd services/sophia && go build ./...`
Expected: Clean compile

- [ ] **Step 5: Commit**

```bash
git add services/sophia/models/models.go services/sophia/ai/service.go services/sophia/api/handlers/chat_handler.go
git commit -m "feat(sophia): persist conversations and generate titles via Thesaurus"
```

---

## Task 4: Iris — Chat UI with Conversation List and History

**Files:**
- Modify: `services/iris/client/src/pages/Chat.jsx`

- [ ] **Step 1: Update Chat.jsx with conversation persistence**

This is the largest change. The Chat component needs:
- State for `conversations` list and `activeConversationId`
- Fetch conversation list on mount
- Load messages when clicking a conversation
- Send `conversation_id` with messages
- "New Chat" button
- Conversation list in sidebar

Read the current `services/iris/client/src/pages/Chat.jsx` file first, then modify it:

**New state variables** (add after existing useState declarations):

```jsx
const [conversations, setConversations] = useState([]);
const [activeConversationId, setActiveConversationId] = useState(null);
```

**New useEffect to fetch conversations on mount** (add after the existing health check useEffect):

```jsx
// Fetch conversation list on mount
useEffect(() => {
  proxyAPI.thesaurus
    .get('/api/v1/conversations/')
    .then((res) => {
      const convs = res.data || [];
      setConversations(convs);
      // Load most recent conversation if exists
      if (convs.length > 0) {
        loadConversation(convs[0].id);
      }
    })
    .catch(() => {});
}, []);
```

**New function to load a conversation's messages:**

```jsx
const loadConversation = async (convId) => {
  try {
    const res = await proxyAPI.thesaurus.get(`/api/v1/conversations/${convId}/messages`);
    const msgs = (res.data || []).map((m) => ({
      role: m.role,
      content: m.content,
      confidence: m.confidence,
      timestamp: new Date(m.created_at),
    }));
    setMessages(msgs);
    setActiveConversationId(convId);
  } catch (err) {
    console.error('Failed to load conversation:', err);
  }
};
```

**New function to start a new chat:**

```jsx
const startNewChat = () => {
  setMessages([]);
  setActiveConversationId(null);
  setInput('');
};
```

**Modify sendMessage** to include conversation_id and update conversations list:

In the existing `sendMessage` function, update the `proxyAPI.sophia.post` call to include `conversation_id`, and update the conversation list after getting the response:

```jsx
const sendMessage = async (text) => {
  if (!text.trim() || loading) return;

  const userMsg = { role: 'user', content: text.trim(), timestamp: new Date() };
  setMessages((prev) => [...prev, userMsg]);
  setInput('');
  setLoading(true);

  try {
    const res = await proxyAPI.sophia.post('/api/v1/chat/', {
      user_id: String(user.id),
      question: text.trim(),
      conversation_id: activeConversationId || undefined,
    });
    const aiMsg = {
      role: 'assistant',
      content: res.data.answer,
      confidence: res.data.confidence,
      sources: res.data.sources || [],
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, aiMsg]);
    setOllamaStatus('active');

    // Update conversation tracking
    if (res.data.conversation_id) {
      setActiveConversationId(res.data.conversation_id);
      // Refresh conversation list
      proxyAPI.thesaurus
        .get('/api/v1/conversations/')
        .then((r) => setConversations(r.data || []))
        .catch(() => {});
    }
  } catch (err) {
    const errMsg = {
      role: 'assistant',
      content: "I couldn't process that right now. Please check that Ollama is running on the Jetson.",
      error: true,
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, errMsg]);
    setOllamaStatus('offline');
  } finally {
    setLoading(false);
  }
};
```

**Add conversation list to the sidebar.** In the sidebar JSX (the left panel with nav links), add above the nav links:

```jsx
{/* New Chat button */}
<div style={{ padding: '16px 24px', borderBottom: '1px solid #e5e5e5' }}>
  <button
    onClick={startNewChat}
    style={{
      width: '100%',
      padding: '10px 16px',
      background: '#1A1A1A',
      color: '#fff',
      border: 'none',
      borderRadius: 8,
      cursor: 'pointer',
      fontFamily: FONTS.body,
      fontSize: 13,
      fontWeight: 600,
    }}
  >
    + New Chat
  </button>
</div>

{/* Conversation history */}
{conversations.length > 0 && (
  <div style={{ padding: '8px 0', borderBottom: '1px solid #e5e5e5', maxHeight: 300, overflowY: 'auto' }}>
    <div style={{ padding: '4px 24px', fontSize: 10, fontWeight: 700, textTransform: 'uppercase', color: COLORS.stone500, letterSpacing: 1 }}>
      Recent Chats
    </div>
    {conversations.map((conv) => (
      <div
        key={conv.id}
        onClick={() => loadConversation(conv.id)}
        style={{
          padding: '8px 24px',
          cursor: 'pointer',
          background: conv.id === activeConversationId ? '#f5f5f4' : 'transparent',
          borderLeft: conv.id === activeConversationId ? '3px solid #1A1A1A' : '3px solid transparent',
          fontFamily: FONTS.body,
          fontSize: 13,
          color: COLORS.stone700,
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {conv.title || 'New Chat'}
        <div style={{ fontSize: 10, color: COLORS.stone500, marginTop: 2 }}>
          {new Date(conv.updated_at).toLocaleDateString()}
        </div>
      </div>
    ))}
  </div>
)}
```

- [ ] **Step 2: Verify Iris builds**

Run: `make build-iris`
Expected: React builds successfully, esbuild bundles server

- [ ] **Step 3: Commit**

```bash
git add services/iris/client/src/pages/Chat.jsx
git commit -m "feat(iris): add conversation list sidebar and chat history persistence"
```

---

## Task 5: Build, Deploy, and Verify

- [ ] **Step 1: Build all changed services**

```bash
make build-arm64-thesaurus
make build-arm64-sophia
make build-iris
```

- [ ] **Step 2: Deploy to Jetson**

```bash
# Stop services
sshpass -p jetson ssh -o StrictHostKeyChecking=no sagar@10.0.0.16 'echo jetson | sudo -S systemctl stop localfinance-thesaurus localfinance-sophia localfinance-iris'
sleep 2

# Deploy binaries
sshpass -p jetson scp -o StrictHostKeyChecking=no dist/thesaurus sagar@10.0.0.16:/home/sagar/localfinance/bin/thesaurus
sshpass -p jetson scp -o StrictHostKeyChecking=no dist/sophia sagar@10.0.0.16:/home/sagar/localfinance/bin/sophia
sshpass -p jetson scp -o StrictHostKeyChecking=no dist/iris-server.js sagar@10.0.0.16:/home/sagar/localfinance/iris/server.js
sshpass -p jetson scp -o StrictHostKeyChecking=no -r services/iris/client/build sagar@10.0.0.16:/home/sagar/localfinance/iris/client/

# Start services (thesaurus first — it runs migrations)
sshpass -p jetson ssh -o StrictHostKeyChecking=no sagar@10.0.0.16 'echo jetson | sudo -S systemctl start localfinance-thesaurus'
sleep 3
sshpass -p jetson ssh -o StrictHostKeyChecking=no sagar@10.0.0.16 'echo jetson | sudo -S systemctl start localfinance-sophia localfinance-iris'
sleep 3
```

- [ ] **Step 3: Verify all services healthy**

```bash
make verify
```
Expected: All 5 services healthy

- [ ] **Step 4: Test end-to-end flow**

```bash
# Register/login to get token
TOKEN=$(curl -s -X POST http://10.0.0.16:3001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"sagarjhaa@gmail.com","password":"test123"}' | jq -r '.token')

# Send a chat message (no conversation_id — creates new)
curl -s --max-time 120 -X POST http://10.0.0.16:8002/api/v1/chat/ \
  -H "Content-Type: application/json" \
  -d '{"question":"hi","user_id":"3e196552-5413-417d-9202-d1d238e07a40"}' | jq '.conversation_id, .answer'

# List conversations
curl -s http://10.0.0.16:8001/api/v1/conversations/ \
  -H "Authorization: Bearer $TOKEN" | jq '.[0].title, .[0].id'
```

- [ ] **Step 5: Commit any fixes**

If any issues found, fix and commit with descriptive messages.
