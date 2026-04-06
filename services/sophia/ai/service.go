package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/sagarjhaa/localfinance/services/sophia/config"
	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

type Service struct {
	config       config.AIConfig
	httpClient   *http.Client
	thesaurusURL string
}

type OllamaRequest struct {
	Model       string            `json:"model"`
	Prompt      string            `json:"prompt"`
	Stream      bool              `json:"stream"`
	Temperature float32           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewService(aiConfig config.AIConfig) (*Service, error) {
	return &Service{
		config: aiConfig,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
		thesaurusURL: "",
	}, nil
}

func (s *Service) SetThesaurusURL(url string) {
	s.thesaurusURL = url
}

// GetOllamaHost returns the configured Ollama host URL.
func (s *Service) GetOllamaHost() string {
	return s.config.OllamaHost
}

// GetUserModelPreference fetches the user's preferred chat model from Thesaurus.
// Falls back to the default config model if the preference is not set or on error.
func (s *Service) GetUserModelPreference(userID string) string {
	if s.thesaurusURL == "" {
		return s.config.ModelName
	}

	resp, err := s.httpClient.Get(fmt.Sprintf("%s/api/v1/internal/preferences/%s", s.thesaurusURL, userID))
	if err != nil {
		return s.config.ModelName
	}
	defer resp.Body.Close()

	var pref struct {
		ChatModel string `json:"chat_model"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pref); err != nil || pref.ChatModel == "" {
		return s.config.ModelName
	}

	return pref.ChatModel
}

// isConversational checks if a question is a greeting or general chat, not a financial query
func isConversational(question string) bool {
	lower := strings.ToLower(strings.TrimSpace(question))

	// Short messages (1-3 words) that don't contain financial terms are likely conversational
	words := strings.Fields(lower)
	if len(words) <= 3 {
		financialTerms := []string{"spend", "spent", "expense", "transaction", "budget", "category",
			"food", "shopping", "transport", "income", "payment", "balance", "total",
			"how much", "show me", "list", "what did"}
		for _, term := range financialTerms {
			if strings.Contains(lower, term) {
				return false
			}
		}
		return true
	}

	// Explicit greetings
	greetings := []string{"hello", "hi", "hey", "good morning", "good afternoon", "good evening",
		"what can you do", "who are you", "help", "what are you", "how are you", "thanks", "thank you"}
	for _, g := range greetings {
		if strings.Contains(lower, g) {
			return true
		}
	}

	return false
}

func (s *Service) handleConversational(question string, userModel string) (models.AIResponse, error) {
	prompt := fmt.Sprintf(`You are a friendly financial assistant for LocalFinance, a privacy-first personal finance app.
The user said: "%s"

Respond naturally and briefly. If it's a greeting, greet back and mention what you can help with (analyzing spending, categorizing transactions, answering questions about their finances). Keep it to 2-3 sentences max. Be warm but concise.`, question)

	response, err := s.queryOllamaWithModel(prompt, 0.7, userModel)
	if err != nil {
		return models.AIResponse{}, err
	}

	return models.AIResponse{
		Answer:     response,
		Confidence: 1.0,
	}, nil
}

// AnswerFinancialQuery orchestrates conversation persistence and query handling.
// It creates/reuses a conversation, saves messages, and generates titles for new conversations.
func (s *Service) AnswerFinancialQuery(query models.FinancialQuery) (models.AIResponse, error) {
	// Determine conversation ID — reuse existing or create new
	conversationID := query.ConversationID
	isNewConversation := false
	if conversationID == "" && s.thesaurusURL != "" {
		var err error
		conversationID, err = s.createConversation(query.UserID)
		if err != nil {
			log.Printf("Failed to create conversation: %v (continuing without persistence)", err)
		} else {
			isNewConversation = true
		}
	}

	// Save user message
	if conversationID != "" && s.thesaurusURL != "" {
		if err := s.saveMessage(conversationID, "user", query.Question, nil); err != nil {
			log.Printf("Failed to save user message: %v", err)
		}
	}

	// Get user's preferred model
	userModel := s.GetUserModelPreference(query.UserID)

	// Route to appropriate handler
	var response models.AIResponse
	var err error
	if isConversational(query.Question) {
		response, err = s.handleConversational(query.Question, userModel)
	} else {
		response, err = s.handleFinancialQuery(query, userModel)
	}
	if err != nil {
		return models.AIResponse{}, err
	}

	response.Model = userModel

	// Save assistant message
	if conversationID != "" && s.thesaurusURL != "" {
		confidence := response.Confidence
		if err := s.saveMessage(conversationID, "assistant", response.Answer, &confidence); err != nil {
			log.Printf("Failed to save assistant message: %v", err)
		}
	}

	// Generate title async for new conversations
	if isNewConversation && conversationID != "" {
		go s.updateConversationTitle(conversationID, query.Question)
	}

	response.ConversationID = conversationID
	return response, nil
}

// handleFinancialQuery implements the two-pass approach:
// Pass 1: LLM parses user question → structured query intent (API params)
// Execute: Call Thesaurus REST API with those params → real transaction data
// Pass 2: Real data + original question → LLM generates natural language answer
func (s *Service) handleFinancialQuery(query models.FinancialQuery, userModel string) (models.AIResponse, error) {
	// Pass 1: Parse user question into structured query intent
	intent, err := s.parseQueryIntent(query.Question, userModel)
	if err != nil {
		log.Printf("Pass 1 failed, falling back to generic: %v", err)
		// Fallback: just get recent transactions
		intent = &models.QueryIntent{NeedsSearch: true, Limit: 50}
	}

	log.Printf("Query intent: %+v", intent)

	// Execute: Fetch real data from Thesaurus based on intent
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

	// If we got no data at all, fetch recent transactions as fallback
	if len(transactions) == 0 && len(summaryItems) == 0 {
		transactions, _ = s.getUserTransactions(query.UserID, 50)
	}

	// Build data context string for Pass 2
	dataContext := s.buildDataContext(transactions, summaryItems)

	// Pass 2: Generate natural language answer from real data
	answer, err := s.generateAnswer(query.Question, dataContext, userModel)
	if err != nil {
		return models.AIResponse{}, fmt.Errorf("failed to generate answer: %w", err)
	}

	return models.AIResponse{
		Answer:     answer,
		Confidence: s.calculateConfidence(transactions, summaryItems),
		Sources:    transactions,
	}, nil
}

// createConversation creates a new conversation in Thesaurus and returns the ID.
func (s *Service) createConversation(userID string) (string, error) {
	payload, _ := json.Marshal(map[string]string{
		"user_id": userID,
		"title":   "New conversation",
	})

	resp, err := s.httpClient.Post(
		s.thesaurusURL+"/api/v1/internal/conversations",
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create conversation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create conversation returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse conversation response: %w", err)
	}

	return result.ID, nil
}

// saveMessage saves a chat message to a conversation in Thesaurus.
func (s *Service) saveMessage(conversationID, role, content string, confidence *float64) error {
	payload := map[string]interface{}{
		"role":    role,
		"content": content,
	}
	if confidence != nil {
		payload["confidence"] = *confidence
	}

	payloadJSON, _ := json.Marshal(payload)

	resp, err := s.httpClient.Post(
		fmt.Sprintf("%s/api/v1/internal/conversations/%s/messages", s.thesaurusURL, conversationID),
		"application/json",
		bytes.NewBuffer(payloadJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("save message returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// updateConversationTitle generates a short title via Ollama and PATCHes the conversation.
// Intended to run in a goroutine (async, non-blocking).
func (s *Service) updateConversationTitle(conversationID, question string) {
	prompt := fmt.Sprintf(`Generate a very short title (3-5 words) for a finance chat that starts with this question: "%s"

Rules:
- Exactly 3-5 words
- No quotes, no punctuation
- Descriptive of the topic
- Example: "Monthly Food Spending" or "Recent Amazon Purchases"

Title:`, question)

	title, err := s.queryOllamaRaw(prompt, 0.3)
	if err != nil {
		log.Printf("Failed to generate conversation title: %v", err)
		return
	}

	// Clean up the title — take first line, trim whitespace and quotes
	title = strings.TrimSpace(title)
	if idx := strings.IndexAny(title, "\n\r"); idx >= 0 {
		title = title[:idx]
	}
	title = strings.Trim(title, "\"'`")
	title = strings.TrimSpace(title)

	// Truncate if too long
	if len(title) > 100 {
		title = title[:100]
	}
	if title == "" {
		title = "Finance Chat"
	}

	// PATCH the conversation title in Thesaurus
	payload, _ := json.Marshal(map[string]string{"title": title})
	req, err := http.NewRequest(http.MethodPatch,
		fmt.Sprintf("%s/api/v1/internal/conversations/%s", s.thesaurusURL, conversationID),
		bytes.NewBuffer(payload),
	)
	if err != nil {
		log.Printf("Failed to build PATCH request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to update conversation title: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Update title returned %d: %s", resp.StatusCode, string(body))
	}
}

// Pass 1: Ask LLM to parse the user's question into structured query parameters
func (s *Service) parseQueryIntent(question string, userModel string) (*models.QueryIntent, error) {
	today := time.Now().Format("2006-01-02")

	prompt := fmt.Sprintf(`You are a query parser. Convert this financial question into a JSON query.
Today's date is %s.

Available categories: Food, Transport, Shopping, Entertainment, Utilities, Housing, Income, Transfer, Health, Cash, EMI, Education, Other

Rules:
- Set needs_summary=true when user asks about spending by category, totals, or breakdowns
- Set needs_search=true when user asks about specific transactions or merchants
- Use start_date/end_date in YYYY-MM-DD format for time ranges
- "last month" means the previous calendar month, "last 14 days" means 14 days before today
- Use description for merchant/store name searches (e.g., "Amazon", "Starbucks")
- Use category for category-level queries (e.g., "food", "shopping")
- Default limit to 50 if not specified

Respond with ONLY valid JSON, no other text:

Question: %s`, today, question)

	response, err := s.queryOllamaWithModel(prompt, 0.1, userModel)
	if err != nil {
		return nil, err
	}

	// Extract JSON from the response (LLM might wrap it in markdown)
	jsonStr := extractJSON(response)

	var intent models.QueryIntent
	if err := json.Unmarshal([]byte(jsonStr), &intent); err != nil {
		log.Printf("Failed to parse intent JSON: %s (raw: %s)", err, jsonStr)
		// Fallback: if the question mentions categories/spending, assume summary
		lower := strings.ToLower(question)
		intent = models.QueryIntent{Limit: 50}
		if containsAny(lower, []string{"category", "categories", "spending", "spent", "breakdown", "summary", "total"}) {
			intent.NeedsSummary = true
			// Default to last 30 days
			intent.StartDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
			intent.EndDate = today
		} else {
			intent.NeedsSearch = true
		}
	}

	// Ensure at least one query type
	if !intent.NeedsSummary && !intent.NeedsSearch {
		intent.NeedsSearch = true
	}
	if intent.Limit == 0 {
		intent.Limit = 50
	}

	return &intent, nil
}

// Execute: Call Thesaurus /transactions/summary endpoint
func (s *Service) fetchSpendingSummary(userID, startDate, endDate string) ([]models.SpendingSummaryItem, error) {
	if s.thesaurusURL == "" {
		return nil, fmt.Errorf("thesaurus URL not configured")
	}

	// Default date range if not specified
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	u := fmt.Sprintf("%s/api/v1/internal/transactions/summary?user_id=%s&start_date=%s&end_date=%s",
		s.thesaurusURL, userID, startDate, endDate)

	resp, err := s.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch summary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("summary API returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var items []models.SpendingSummaryItem
	if err := json.Unmarshal(body, &items); err != nil {
		// Try wrapped response
		var wrapped struct {
			Summary []models.SpendingSummaryItem `json:"summary"`
		}
		if err2 := json.Unmarshal(body, &wrapped); err2 == nil {
			return wrapped.Summary, nil
		}
		return nil, fmt.Errorf("failed to parse summary: %w (body: %s)", err, string(body))
	}

	return items, nil
}

// Execute: Call Thesaurus /transactions/search endpoint
func (s *Service) searchTransactions(userID string, intent *models.QueryIntent) ([]models.TransactionRef, error) {
	if s.thesaurusURL == "" {
		return nil, fmt.Errorf("thesaurus URL not configured")
	}

	// Build search filter
	filter := map[string]interface{}{
		"user_id": userID,
	}
	if intent.Category != "" {
		filter["category"] = intent.Category
	}
	if intent.Description != "" {
		filter["description"] = intent.Description
	}
	if intent.StartDate != "" {
		filter["start_date"] = intent.StartDate + "T00:00:00Z"
	}
	if intent.EndDate != "" {
		filter["end_date"] = intent.EndDate + "T23:59:59Z"
	}
	if intent.MinAmount != 0 {
		filter["min_amount"] = intent.MinAmount
	}
	if intent.MaxAmount != 0 {
		filter["max_amount"] = intent.MaxAmount
	}
	if intent.Limit > 0 {
		filter["limit"] = intent.Limit
	}

	filterJSON, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Post(
		s.thesaurusURL+"/api/v1/internal/transactions/search",
		"application/json",
		bytes.NewBuffer(filterJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search transactions: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// Fallback to list endpoint
		return s.getUserTransactionsFiltered(userID, intent)
	}

	var transactions []models.TransactionRef
	if err := json.Unmarshal(body, &transactions); err != nil {
		// Try wrapped response
		var wrapped struct {
			Data         []models.TransactionRef `json:"data"`
			Transactions []models.TransactionRef `json:"transactions"`
		}
		if err2 := json.Unmarshal(body, &wrapped); err2 == nil {
			if len(wrapped.Data) > 0 {
				return wrapped.Data, nil
			}
			return wrapped.Transactions, nil
		}
		return nil, err
	}

	return transactions, nil
}

// Fallback: use list endpoint with query params
func (s *Service) getUserTransactionsFiltered(userID string, intent *models.QueryIntent) ([]models.TransactionRef, error) {
	params := url.Values{}
	params.Set("user_id", userID)
	if intent.Category != "" {
		params.Set("category", intent.Category)
	}
	if intent.StartDate != "" {
		params.Set("start_date", intent.StartDate)
	}
	if intent.EndDate != "" {
		params.Set("end_date", intent.EndDate)
	}
	if intent.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", intent.Limit))
	}

	u := fmt.Sprintf("%s/api/v1/internal/transactions?%s", s.thesaurusURL, params.Encode())
	return s.fetchTransactions(u)
}

func (s *Service) getUserTransactions(userID string, limit int) ([]models.TransactionRef, error) {
	if s.thesaurusURL == "" {
		return []models.TransactionRef{}, nil
	}
	u := fmt.Sprintf("%s/api/v1/internal/transactions?user_id=%s&limit=%d", s.thesaurusURL, userID, limit)
	return s.fetchTransactions(u)
}

func (s *Service) fetchTransactions(u string) ([]models.TransactionRef, error) {
	resp, err := s.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []models.TransactionRef{}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try multiple response shapes
	var direct []models.TransactionRef
	if err := json.Unmarshal(body, &direct); err == nil {
		return direct, nil
	}

	var wrapped struct {
		Data         []models.TransactionRef `json:"data"`
		Transactions []models.TransactionRef `json:"transactions"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		if len(wrapped.Data) > 0 {
			return wrapped.Data, nil
		}
		return wrapped.Transactions, nil
	}

	return []models.TransactionRef{}, nil
}

// Build data context string from real query results for Pass 2
func (s *Service) buildDataContext(transactions []models.TransactionRef, summary []models.SpendingSummaryItem) string {
	var sb strings.Builder

	if len(summary) > 0 {
		sb.WriteString("SPENDING SUMMARY BY CATEGORY:\n")
		totalSpend := 0.0
		for _, item := range summary {
			totalSpend += item.TotalAmount
		}
		sb.WriteString(fmt.Sprintf("Total spending: $%.2f\n", totalSpend))
		for _, item := range summary {
			sb.WriteString(fmt.Sprintf("- %s: $%.2f (%d transactions, %.1f%%)\n",
				item.Category, item.TotalAmount, item.Count, item.Percentage))
		}
		sb.WriteString("\n")
	}

	if len(transactions) > 0 {
		sb.WriteString(fmt.Sprintf("TRANSACTIONS (%d results):\n", len(transactions)))
		for i, tx := range transactions {
			if i >= 30 {
				sb.WriteString(fmt.Sprintf("... and %d more transactions\n", len(transactions)-30))
				break
			}
			cleanDesc := cleanMerchantName(tx.Description)
			dateStr := tx.Date
			if len(dateStr) > 10 {
				dateStr = dateStr[:10] // Just YYYY-MM-DD
			}
			sb.WriteString(fmt.Sprintf("- %s | %s | $%.2f | %s\n",
				dateStr, cleanDesc, tx.Amount, tx.Category))
		}
	}

	if sb.Len() == 0 {
		sb.WriteString("NO TRANSACTION DATA FOUND for this query. The user may not have uploaded any statements yet.")
	}

	return sb.String()
}

// cleanMerchantName does basic cleanup of raw bank descriptions
// No hardcoded merchant names or locations — keeps it universal
func cleanMerchantName(raw string) string {
	s := raw

	// Remove trailing digits (phone numbers, transaction IDs, zip codes)
	for len(s) > 3 && s[len(s)-1] >= '0' && s[len(s)-1] <= '9' {
		s = s[:len(s)-1]
	}
	s = strings.TrimRight(s, "-. #*_")

	// Remove common payment prefixes
	for _, prefix := range []string{"TST*", "SQ *", "SQ*", "PP*"} {
		s = strings.TrimPrefix(s, prefix)
	}

	// Truncate to first 30 chars if still long (city/state/phone likely after)
	if len(s) > 30 {
		s = s[:30]
		// Don't cut in middle of a word — find last clean break
		s = strings.TrimRight(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		if len(s) < 5 {
			s = raw[:30] // give up on smart trim
		}
	}

	s = strings.TrimRight(s, "-. #*_,")
	if len(s) < 2 {
		return raw
	}

	// Title case if all uppercase
	if s == strings.ToUpper(s) && len(s) > 2 {
		s = strings.Title(strings.ToLower(s))
	}

	return s
}

// Pass 2: Generate natural language answer from real data
func (s *Service) generateAnswer(question, dataContext string, userModel string) (string, error) {
	prompt := fmt.Sprintf(`You are a concise financial assistant. Answer using ONLY the real data below.

RULES:
- Use actual numbers from the data. Never invent amounts.
- Be concise: 2-4 sentences for the main answer.
- Format currency as $X,XXX.XX
- NEVER include raw transaction descriptions or codes. Use clean merchant names only (e.g., "Amazon" not "AMAZON.COM*EY8608493").
- For category breakdowns, list each category with its total and percentage on a clean line.
- Highlight the biggest spending category and any notable patterns.
- If no data, say so honestly.
- Do not give generic advice. Only answer what was asked.

REAL DATA:
%s

USER QUESTION: %s

Answer:`, dataContext, question)

	return s.queryOllamaWithModel(prompt, 0.2, userModel)
}

// queryOllamaRaw queries Ollama without prepending the system prompt
func (s *Service) queryOllamaRaw(prompt string, temperature float32) (string, error) {
	if temperature == 0 {
		temperature = s.config.Temperature
	}

	requestBody := OllamaRequest{
		Model:       s.config.ModelName,
		Prompt:      prompt,
		Stream:      false,
		Temperature: temperature,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.httpClient.Post(
		s.config.OllamaHost+"/api/generate",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return ollamaResp.Response, nil
}

// queryOllamaWithModel queries Ollama with an explicit model override
func (s *Service) queryOllamaWithModel(prompt string, temperature float32, model string) (string, error) {
	if temperature == 0 {
		temperature = s.config.Temperature
	}

	reqBody := OllamaRequest{
		Model:       model,
		Prompt:      prompt,
		Stream:      false,
		Temperature: temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.httpClient.Post(
		s.config.OllamaHost+"/api/generate",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return strings.TrimSpace(ollamaResp.Response), nil
}

// queryOllama queries with the system prompt prepended (for categorization etc.)
func (s *Service) queryOllama(prompt string) (string, error) {
	fullPrompt := fmt.Sprintf("%s\n\nUser Query: %s", s.config.SystemPrompt, prompt)
	return s.queryOllamaRaw(fullPrompt, s.config.Temperature)
}

func (s *Service) calculateConfidence(txns []models.TransactionRef, summary []models.SpendingSummaryItem) float64 {
	if len(txns) == 0 && len(summary) == 0 {
		return 0.3 // No data — low confidence
	}
	if len(txns) > 10 || len(summary) > 3 {
		return 0.95 // Lots of data
	}
	return 0.75
}

// CategorizeTransaction uses AI to categorize a transaction
func (s *Service) CategorizeTransaction(description string) (models.CategoryResult, error) {
	prompt := fmt.Sprintf(`Categorize this financial transaction into one of these categories:

Categories: Food, Transportation, Shopping, Entertainment, Bills, Healthcare, Education, Travel, Income, Other

Transaction: %s

Response format: Category: [category] | Confidence: [0.0-1.0] | Reason: [brief explanation]`, description)

	response, err := s.queryOllama(prompt)
	if err != nil {
		return models.CategoryResult{}, err
	}

	return models.CategoryResult{
		Category:   s.extractCategory(response),
		Confidence: 0.8,
		Reasoning:  response,
	}, nil
}

// GenerateInsights generates financial insights for a user
func (s *Service) GenerateInsights(userID string) ([]models.FinancialInsight, error) {
	// Get spending summary for last 30 days
	startDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	summary, err := s.fetchSpendingSummary(userID, startDate, endDate)
	if err != nil || len(summary) == 0 {
		return []models.FinancialInsight{{
			Type:        "no_data",
			Title:       "No Transaction Data",
			Description: "Upload bank statements to get personalized insights.",
			Priority:    "high",
			ActionItem:  "Upload a CSV or PDF bank statement",
		}}, nil
	}

	dataContext := s.buildDataContext(nil, summary)
	prompt := fmt.Sprintf(`Based on this spending data, provide exactly 3 insights. Each on its own line, format: TITLE: description

%s

Insights:`, dataContext)

	response, err := s.queryOllamaRaw(prompt, 0.3)
	if err != nil {
		return nil, err
	}

	// Parse insights from response
	lines := strings.Split(response, "\n")
	var insights []models.FinancialInsight
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") {
			line = strings.TrimPrefix(line, "- ")
			line = strings.TrimPrefix(line, "* ")
		}
		if len(line) < 10 {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		title := strings.TrimSpace(parts[0])
		desc := line
		if len(parts) == 2 {
			desc = strings.TrimSpace(parts[1])
		}
		insights = append(insights, models.FinancialInsight{
			Type:        "spending_insight",
			Title:       title,
			Description: desc,
			Priority:    "medium",
			CreatedAt:   time.Now(),
		})
		if len(insights) >= 3 {
			break
		}
	}

	if len(insights) == 0 {
		insights = []models.FinancialInsight{{
			Type:        "spending_pattern",
			Title:       "Spending Analysis",
			Description: response,
			Priority:    "medium",
			CreatedAt:   time.Now(),
		}}
	}

	return insights, nil
}

func (s *Service) extractCategory(response string) string {
	categories := []string{"Food", "Transportation", "Shopping", "Entertainment", "Bills", "Healthcare", "Education", "Travel", "Income", "Other"}
	lower := strings.ToLower(response)
	for _, category := range categories {
		if strings.Contains(lower, strings.ToLower(category)) {
			return category
		}
	}
	return "Other"
}

// ParseTransactions uses the LLM to extract transactions from raw statement text
func (s *Service) ParseTransactions(text string, userModel string) ([]map[string]interface{}, error) {
	// Extract the transaction section — skip headers/summaries at the top
	// Look for the first line starting with a date pattern (MM/DD) which indicates transactions
	text = extractTransactionSection(text, 2000)

	prompt := fmt.Sprintf(`Extract transactions as JSON.
Fields: date (YYYY-MM-DD), description (Clean Name), amount (Positive=Charge, Negative=Payment), category (Food, Transport, Shopping, Entertainment, Utilities, Housing, Income, Transfer, Health, Cash, EMI, Education, Other).

STRICT RULES:
1. Only extract transactions from the activity table. Do not extract account headers, reward balances, or summary totals.
2. The date must be the one listed on the transaction line. Do not use the statement's overall date.
3. If a transaction doesn't fit a category, use "Other". NEVER create new categories.
4. No raw codes/cities/states in description.
5. If year unknown, use 2026.
6. Output ONLY the JSON array inside <JSON> tags.

Examples:
Input: 03/12 AMZN Mktp US*Amzn.com/bill WA $22.50
Output: {"date": "2026-03-12", "description": "Amazon", "amount": 22.50, "category": "Shopping"}

Input: 02/18 SAFEWAY #1196 SUNNYVALE CA 28.33
Output: {"date": "2026-02-18", "description": "Safeway", "amount": 28.33, "category": "Food"}

Input: 03/12 AUTOMATIC PAYMENT - THANK YOU -1323.73
Output: {"date": "2026-03-12", "description": "Payment Received", "amount": -1323.73, "category": "Income"}

Start your response exactly with "<JSON>[" and end with "]</JSON>".

Statement:
%s`, text)

	response, err := s.queryOllamaWithModel(prompt, 0.1, userModel)
	if err != nil {
		return nil, err
	}

	// Extract JSON from <JSON> anchor tags first, then fallback to general extraction
	jsonStr := ""
	if start := strings.Index(response, "<JSON>"); start >= 0 {
		start += 6
		if end := strings.Index(response[start:], "</JSON>"); end >= 0 {
			jsonStr = strings.TrimSpace(response[start : start+end])
		}
	}
	if jsonStr == "" {
		jsonStr = extractJSON(response)
	}

	// Remove literal newlines inside JSON strings (invalid in JSON)
	// Replace newlines that are between quotes with spaces
	// Simple approach: normalize all newlines between [ ] to spaces, then let JSON parser handle structure
	jsonStr = strings.ReplaceAll(jsonStr, "\r\n", " ")
	jsonStr = strings.ReplaceAll(jsonStr, "\n", " ")
	jsonStr = strings.ReplaceAll(jsonStr, "\r", " ")
	// Collapse multiple spaces
	for strings.Contains(jsonStr, "  ") {
		jsonStr = strings.ReplaceAll(jsonStr, "  ", " ")
	}

	// Repair common LLM JSON issues
	jsonStr = repairJSON(jsonStr)

	var transactions []map[string]interface{}

	// Try parsing as array of objects
	if err := json.Unmarshal([]byte(jsonStr), &transactions); err != nil {
		// Try wrapping in array if it's comma-separated objects
		if err2 := json.Unmarshal([]byte("["+jsonStr+"]"), &transactions); err2 != nil {
			// Try parsing as array of strings and convert to transaction objects
			var strArray []string
			if err3 := json.Unmarshal([]byte(jsonStr), &strArray); err3 == nil {
				for _, s := range strArray {
					transactions = append(transactions, map[string]interface{}{
						"description": s,
						"date":        "",
						"amount":      0,
						"category":    "Other",
					})
				}
			} else {
				log.Printf("AI parse JSON error. Raw response (first 500 chars): %s", response[:min(500, len(response))])
				return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
			}
		}
	}

	return transactions, nil
}

// extractTransactionSection finds the transaction lines in statement text, skipping headers
func extractTransactionSection(text string, maxLen int) string {
	lines := strings.Split(text, "\n")
	datePattern := regexp.MustCompile(`^\s*\d{2}/\d{2}`)

	// Find the first line that looks like a transaction (starts with MM/DD)
	startIdx := -1
	for i, line := range lines {
		if datePattern.MatchString(line) {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		// No date patterns found — just truncate from start
		if len(text) > maxLen {
			return text[:maxLen]
		}
		return text
	}

	// Build text from the transaction section
	var result strings.Builder
	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		if result.Len()+len(line)+1 > maxLen {
			break
		}
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

// repairJSON fixes common LLM JSON output issues
func repairJSON(s string) string {
	// Remove thinking tags and their content
	s = regexp.MustCompile(`(?s)<think>.*?</think>`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?s)<thinking>.*?</thinking>`).ReplaceAllString(s, "")

	// Remove escaped backslashes that aren't part of valid JSON escapes
	s = strings.ReplaceAll(s, `\\n`, " ")
	s = strings.ReplaceAll(s, `\\t`, " ")
	s = strings.ReplaceAll(s, `\\"`, `"`)
	s = strings.ReplaceAll(s, `\\\\`, `\`)

	// Fix unquoted keys: {date: "val"} → {"date": "val"}
	re := regexp.MustCompile(`([{,]\s*)([a-zA-Z_][a-zA-Z0-9_]*)\s*:`)
	s = re.ReplaceAllString(s, `$1"$2":`)

	// Fix single quotes: {'key': 'val'} → {"key": "val"}
	s = strings.ReplaceAll(s, `'`, `"`)

	// Fix trailing commas before ] or }
	s = regexp.MustCompile(`,\s*([}\]])`).ReplaceAllString(s, "$1")

	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper: extract JSON from LLM response that might have markdown wrapping
func extractJSON(response string) string {
	response = strings.TrimSpace(response)

	// Try to find JSON between ```json and ```
	if idx := strings.Index(response, "```json"); idx >= 0 {
		start := idx + 7
		if end := strings.Index(response[start:], "```"); end >= 0 {
			return strings.TrimSpace(response[start : start+end])
		}
	}
	// Try to find JSON between ``` and ```
	if idx := strings.Index(response, "```"); idx >= 0 {
		start := idx + 3
		if end := strings.Index(response[start:], "```"); end >= 0 {
			candidate := strings.TrimSpace(response[start : start+end])
			if strings.HasPrefix(candidate, "{") {
				return candidate
			}
		}
	}

	// Try to find raw JSON object
	if start := strings.Index(response, "{"); start >= 0 {
		if end := strings.LastIndex(response, "}"); end > start {
			return response[start : end+1]
		}
	}

	// Try to find raw JSON array
	if start := strings.Index(response, "["); start >= 0 {
		if end := strings.LastIndex(response, "]"); end > start {
			return response[start : end+1]
		}
	}

	return response
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
