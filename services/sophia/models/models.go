package models

import (
	"time"

	"github.com/google/uuid"
)

type FinancialQuery struct {
	UserID         string `json:"user_id" binding:"required"`
	Question       string `json:"question" binding:"required"`
	Context        string `json:"context,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
}

type AIResponse struct {
	Answer         string             `json:"answer"`
	Confidence     float64            `json:"confidence"`
	Sources        []TransactionRef   `json:"sources,omitempty"`
	Insights       []FinancialInsight `json:"insights,omitempty"`
	GeneratedAt    time.Time          `json:"generated_at"`
	ConversationID string             `json:"conversation_id,omitempty"`
	Title          string             `json:"title,omitempty"`
}

type ChatMessage struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

type FinancialContext struct {
	UserID             string             `json:"user_id"`
	RecentTransactions []TransactionRef   `json:"recent_transactions"`
	Summary            TransactionSummary `json:"summary"`
	ContextGeneratedAt time.Time          `json:"context_generated_at"`
}

type TransactionRef struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Type        string  `json:"type"`
}

type TransactionSummary struct {
	TotalTransactions int                `json:"total_transactions"`
	TotalSpent        float64            `json:"total_spent"`
	TotalIncome       float64            `json:"total_income"`
	Categories        map[string]float64 `json:"categories"`
}

type FinancialInsight struct {
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	ActionItem  string    `json:"action_item"`
	CreatedAt   time.Time `json:"created_at"`
}

type CategoryResult struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

type CategorizationRequest struct {
	Description string  `json:"description" binding:"required"`
	Amount      float64 `json:"amount,omitempty"`
	Merchant    string  `json:"merchant,omitempty"`
}

type InsightsRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Period string `json:"period,omitempty"`
}

type SpendingAnalysis struct {
	UserID            string                   `json:"user_id"`
	Period            string                   `json:"period"`
	TotalSpending     float64                  `json:"total_spending"`
	CategoryBreakdown map[string]CategoryStats `json:"category_breakdown"`
	Trends            []SpendingTrend          `json:"trends"`
	Recommendations   []string                 `json:"recommendations"`
}

type CategoryStats struct {
	Amount     float64 `json:"amount"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
	Average    float64 `json:"average"`
	Trend      string  `json:"trend"`
}

type SpendingTrend struct {
	Category    string  `json:"category"`
	Direction   string  `json:"direction"`
	Magnitude   float64 `json:"magnitude"`
	Description string  `json:"description"`
}

// QueryIntent is the structured output from Pass 1 — LLM parses user question into API params
type QueryIntent struct {
	NeedsSummary bool   `json:"needs_summary"`           // call /transactions/summary
	NeedsSearch  bool   `json:"needs_search"`            // call /transactions/search
	Category     string `json:"category,omitempty"`       // filter by category
	Description  string `json:"description,omitempty"`    // ILIKE search on description
	StartDate    string `json:"start_date,omitempty"`     // YYYY-MM-DD
	EndDate      string `json:"end_date,omitempty"`       // YYYY-MM-DD
	MinAmount    float64 `json:"min_amount,omitempty"`
	MaxAmount    float64 `json:"max_amount,omitempty"`
	Limit        int    `json:"limit,omitempty"`          // max results
}

// SpendingSummaryItem matches Thesaurus GET /transactions/summary response
type SpendingSummaryItem struct {
	Category    string  `json:"category"`
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
}
