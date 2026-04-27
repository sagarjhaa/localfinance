package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/sophia/ai"
	"github.com/sagarjhaa/localfinance/internal/insights"
	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

// TransactionFetcher is what the handler needs from a Thesaurus client. The
// concrete implementation in production is *ai.Service. Tests inject fakes.
type TransactionFetcher interface {
	FetchTransactionsSince(userID string, start time.Time) ([]models.TransactionRef, error)
}

// InsightsHandler wires together the deterministic insights engine, a narrator,
// a transaction source, and (legacy) the AI service. All dependencies are
// injectable for testing — no globals.
type InsightsHandler struct {
	aiService TransactionFetcher
	engine    *insights.Engine
	narrator  insights.Narrator
	clock     func() time.Time

	// legacy fields for the older endpoints that still use the AI service
	// directly (AnalyzeSpendingPatterns, GetUserInsights pre-engine path).
	legacyAI *ai.Service
}

// NewInsightsHandler builds a handler with default deps wired up. The narrator
// defaults to template-only (Phase A) for safety; pass an *llmNarrator via
// NewLLMNarrator if you want LLM polish (set INSIGHTS_LLM_POLISH=1 in env to
// auto-enable).
func NewInsightsHandler(aiService *ai.Service) *InsightsHandler {
	thesaurusURL := os.Getenv("THESAURUS_URL")
	if thesaurusURL == "" {
		thesaurusURL = "http://localhost:8001"
	}
	dismiss := insights.NewThesaurusDismissalFetcher(thesaurusURL)
	dismiss.Client = &http.Client{Timeout: 5 * time.Second}

	var narrator insights.Narrator = insights.NewTemplateNarrator()
	if os.Getenv("INSIGHTS_LLM_POLISH") == "1" {
		narrator = insights.NewLLMNarrator(aiService)
	}

	return &InsightsHandler{
		aiService: aiService,
		engine:    insights.NewEngine(dismiss),
		narrator:  narrator,
		clock:     func() time.Time { return time.Now().UTC() },
		legacyAI:  aiService,
	}
}

// GenerateInsights is the new engine-backed insights endpoint. Pulls 90 days
// of transactions from Thesaurus, runs the deterministic engine, narrates the
// findings, and returns both the structured FinancialInsight list and the
// narrative.
func (h *InsightsHandler) GenerateInsights(c *gin.Context) {
	var request models.InsightsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := h.clock()
	windowStart := now.AddDate(0, 0, -90)

	txs, err := h.aiService.FetchTransactionsSince(request.UserID, windowStart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch transactions",
			"details": err.Error(),
		})
		return
	}

	rc := insights.RuleContext{
		UserID:       request.UserID,
		Transactions: txs,
		Now:          now,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	engineInsights, err := h.engine.Run(ctx, rc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to run insights engine",
			"details": err.Error(),
		})
		return
	}

	// Narrate. Use a longer timeout because LLM polish can be slow on cold start.
	narrCtx, narrCancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer narrCancel()
	narrative, err := h.narrator.Narrate(narrCtx, engineInsights, insights.Period{
		Start: windowStart,
		End:   now,
		Label: request.Period,
	})
	if err != nil {
		// Narrator must never hard-fail the request; fall back to empty narrative.
		narrative = insights.MonthInReviewNarrative{
			PerInsight: map[string]string{},
			Source:     "template",
		}
	}

	publicInsights := make([]models.FinancialInsight, 0, len(engineInsights))
	for _, ins := range engineInsights {
		fi := ins.ToFinancialInsight()
		// Prefer narrator's per-insight prose (deterministic, factual) over the
		// rule's raw description so the UI gets uniform copy.
		if narrText, ok := narrative.PerInsight[ins.Key]; ok && narrText != "" {
			fi.Description = narrText
		}
		publicInsights = append(publicInsights, fi)
	}

	response := gin.H{
		"user_id":      request.UserID,
		"insights":     publicInsights,
		"period":       request.Period,
		"narrative":    narrative,
		"generated_at": now,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserInsights is the GET variant. It defers to GenerateInsights with a
// synthesized request body for parity.
func (h *InsightsHandler) GetUserInsights(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	now := h.clock()
	windowStart := now.AddDate(0, 0, -90)

	txs, err := h.aiService.FetchTransactionsSince(userID, windowStart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch transactions",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	engineInsights, err := h.engine.Run(ctx, insights.RuleContext{
		UserID: userID, Transactions: txs, Now: now,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to run insights engine",
			"details": err.Error(),
		})
		return
	}

	narrCtx, narrCancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer narrCancel()
	narrative, _ := h.narrator.Narrate(narrCtx, engineInsights, insights.Period{
		Start: windowStart, End: now,
	})

	publicInsights := make([]models.FinancialInsight, 0, len(engineInsights))
	for _, ins := range engineInsights {
		fi := ins.ToFinancialInsight()
		if t, ok := narrative.PerInsight[ins.Key]; ok && t != "" {
			fi.Description = t
		}
		publicInsights = append(publicInsights, fi)
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":   userID,
		"insights":  publicInsights,
		"narrative": narrative,
	})
}

// AnalyzeSpendingPatterns remains a stub that returns a canned analysis shape.
// The new structured insights live on GenerateInsights / GetUserInsights.
func (h *InsightsHandler) AnalyzeSpendingPatterns(c *gin.Context) {
	var request models.InsightsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis := models.SpendingAnalysis{
		UserID:        request.UserID,
		Period:        "monthly",
		TotalSpending: 0.0,
		CategoryBreakdown: map[string]models.CategoryStats{
			"Food": {
				Amount: 250.0, Count: 15, Percentage: 25.0, Average: 16.67, Trend: "stable",
			},
			"Transportation": {
				Amount: 180.0, Count: 8, Percentage: 18.0, Average: 22.50, Trend: "increasing",
			},
		},
		Trends: []models.SpendingTrend{
			{
				Category: "Food", Direction: "stable", Magnitude: 2.5,
				Description: "Food spending has remained consistent over the past month",
			},
		},
		Recommendations: []string{
			"Consider setting a monthly food budget based on your average spending",
			"Transportation costs are trending upward - review your commute options",
		},
	}

	c.JSON(http.StatusOK, analysis)
}
