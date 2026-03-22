package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sagarjhaa/localfinance/services/sophia/config"
	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

type Service struct {
	config      config.AIConfig
	httpClient  *http.Client
	thesaurusURL string
}

type OllamaRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Stream      bool    `json:"stream"`
	Temperature float32 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewService(aiConfig config.AIConfig) (*Service, error) {
	return &Service{
		config: aiConfig,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		thesaurusURL: "", // Will be set via dependency injection
	}, nil
}

func (s *Service) SetThesaurusURL(url string) {
	s.thesaurusURL = url
}

func (s *Service) AnswerFinancialQuery(query models.FinancialQuery) (models.AIResponse, error) {
	// 1. Build financial context for the user
	context, err := s.buildFinancialContext(query.UserID)
	if err != nil {
		return models.AIResponse{}, fmt.Errorf("failed to build context: %w", err)
	}

	// 2. Create enhanced prompt with user's financial data
	summaryText := fmt.Sprintf("Total: %d transactions, Spent: $%.2f, Income: $%.2f, Categories: %v", 
		context.Summary.TotalTransactions, context.Summary.TotalSpent, context.Summary.TotalIncome, context.Summary.Categories)
	enhancedPrompt := s.createEnhancedPrompt(query.Question, summaryText)

	// 3. Query Ollama
	response, err := s.queryOllama(enhancedPrompt)
	if err != nil {
		return models.AIResponse{}, fmt.Errorf("failed to query AI: %w", err)
	}

	return models.AIResponse{
		Answer:     response,
		Confidence: 0.85, // TODO: Implement confidence scoring
		Sources:    context.RecentTransactions,
		Insights:   []models.FinancialInsight{}, // TODO: Generate insights
	}, nil
}

func (s *Service) CategorizeTransaction(description string) (models.CategoryResult, error) {
	prompt := fmt.Sprintf(`Categorize this financial transaction into one of these categories:
	
Categories: Food, Transportation, Shopping, Entertainment, Bills, Healthcare, Education, Travel, Income, Other

Transaction: %s

Response format: Category: [category] | Confidence: [0.0-1.0] | Reason: [brief explanation]`, description)

	response, err := s.queryOllama(prompt)
	if err != nil {
		return models.CategoryResult{}, err
	}

	// Parse the response to extract category and confidence
	// For now, return a basic categorization
	return models.CategoryResult{
		Category:   s.extractCategory(response),
		Confidence: 0.8, // TODO: Parse actual confidence from response
		Reasoning:  response,
	}, nil
}

func (s *Service) GenerateInsights(userID string) ([]models.FinancialInsight, error) {
	context, err := s.buildFinancialContext(userID)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Based on this user's financial data, provide 3-5 key insights about their spending patterns:

Recent Transactions: %v

Provide insights in this format:
- [Insight type]: [Description]

Focus on spending trends, unusual patterns, budget recommendations, and savings opportunities.`,
		context.Summary)

	response, err := s.queryOllama(prompt)
	if err != nil {
		return nil, err
	}

	// For now, return a simple insight
	insights := []models.FinancialInsight{
		{
			Type:        "spending_pattern",
			Title:       "Spending Pattern Analysis",
			Description: response,
			Priority:    "medium",
			ActionItem:  "Review monthly spending categories",
		},
	}

	return insights, nil
}

func (s *Service) queryOllama(prompt string) (string, error) {
	fullPrompt := fmt.Sprintf("%s\n\nUser Query: %s", s.config.SystemPrompt, prompt)
	
	requestBody := OllamaRequest{
		Model:       s.config.ModelName,
		Prompt:      fullPrompt,
		Stream:      false,
		Temperature: s.config.Temperature,
		MaxTokens:   s.config.MaxTokens,
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
		return "", fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
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

func (s *Service) buildFinancialContext(userID string) (models.FinancialContext, error) {
	// Call Thesaurus service to get user's transaction data
	transactions, err := s.getUserTransactions(userID, 30) // Last 30 transactions
	if err != nil {
		return models.FinancialContext{}, err
	}

	// Build summary statistics
	summary := s.buildTransactionSummary(transactions)

	return models.FinancialContext{
		UserID:              userID,
		RecentTransactions:  transactions,
		Summary:            summary,
		ContextGeneratedAt: time.Now(),
	}, nil
}

func (s *Service) getUserTransactions(userID string, limit int) ([]models.TransactionRef, error) {
	// TODO: Implement actual HTTP call to Thesaurus service
	// For now, return empty slice
	return []models.TransactionRef{}, nil
}

func (s *Service) buildTransactionSummary(transactions []models.TransactionRef) models.TransactionSummary {
	summary := models.TransactionSummary{
		TotalTransactions: len(transactions),
		Categories:       make(map[string]float64),
	}

	for _, tx := range transactions {
		summary.TotalSpent += tx.Amount
		if tx.Amount < 0 { // Expense
			summary.Categories[tx.Category] += -tx.Amount
		}
	}

	return summary
}

func (s *Service) createEnhancedPrompt(question, context string) string {
	return fmt.Sprintf(`As a financial advisor, answer this question using the user's actual financial data:

Financial Context: %s

User Question: %s

Provide a helpful, specific answer based on their actual spending data. If the data shows concerning patterns, mention them. If there are opportunities for savings, point them out.`,
		context, question)
}

func (s *Service) extractCategory(response string) string {
	// Simple category extraction - in production, this would be more sophisticated
	categories := []string{"Food", "Transportation", "Shopping", "Entertainment", "Bills", "Healthcare", "Education", "Travel", "Income", "Other"}
	
	for _, category := range categories {
		if contains(response, category) {
			return category
		}
	}
	
	return "Other"
}

func contains(text, substr string) bool {
	return len(text) >= len(substr) && 
		(text[:len(substr)] == substr || 
		 text[len(text)-len(substr):] == substr ||
		 containsInMiddle(text, substr))
}

func containsInMiddle(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}