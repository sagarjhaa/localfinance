// Package api wires the unified HTTP surface for the localfinance binary.
// Replaces the per-service Gin routers from the old 5-service stack.
//
// Routes mirror Thesaurus's existing surface 1:1 — same paths, same handlers,
// same auth gating — so existing clients (Iris, Sophia, Logos) keep working
// while the migration completes.
package api

import (
	stdContext "context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/ai"
	"github.com/sagarjhaa/localfinance/internal/ollama"
	"github.com/sagarjhaa/localfinance/internal/api/handlers"
	"github.com/sagarjhaa/localfinance/internal/api/middleware"
	"github.com/sagarjhaa/localfinance/internal/api/setup"
	"github.com/sagarjhaa/localfinance/internal/insights"
	"github.com/sagarjhaa/localfinance/internal/monthreview"
	"github.com/sagarjhaa/localfinance/internal/parse"
	"github.com/sagarjhaa/localfinance/internal/webui"
	sophiaconfig "github.com/sagarjhaa/localfinance/internal/ai/config"
	"gorm.io/gorm"
)

// NewGinRouter builds the consolidated Gin router. db is the shared GORM
// handle; handlers are constructed once at startup so per-request work stays
// allocation-light.
// NewGinRouter constructs the consolidated router with the database handle.
// AI service is built internally from env (OLLAMA_HOST, MODEL_NAME, THESAURUS_URL)
// so existing tests that only need DB-backed routes don't have to wire Ollama.
func NewGinRouter(db *gorm.DB) *gin.Engine {
	cfg, _ := sophiaconfig.Load()
	aiSvc, _ := ai.NewService(cfg.AI)
	thesaurusURL := os.Getenv("THESAURUS_URL")
	if thesaurusURL == "" {
		thesaurusURL = cfg.Thesaurus.BaseURL
	}
	aiSvc.SetThesaurusURL(thesaurusURL)
	return NewGinRouterWithAI(db, aiSvc, cfg.AI.ModelName)
}

// NewGinRouterWithAI builds the router with an explicit AI service. Lets the
// process entrypoint share the AI service across routes and probes.
func NewGinRouterWithAI(db *gorm.DB, aiSvc *ai.Service, modelName string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// Hand the AI service the live DB so the chat path can run the
	// LLM-writes-SQL planner directly instead of round-tripping through
	// the legacy intent + REST flow.
	aiSvc.SetDB(db)

	userHandler := handlers.NewUserHandler(db)
	accountHandler := handlers.NewAccountHandler(db)
	transactionHandler := handlers.NewTransactionHandler(db)
	budgetHandler := handlers.NewBudgetHandler(db)
	authHandler := handlers.NewAuthHandler(db)
	uploadHandler := handlers.NewUploadHandler(db)
	documentHandler := handlers.NewDocumentHandler(db)
	categoryRuleHandler := handlers.NewCategoryRuleHandler(db)
	conversationHandler := handlers.NewConversationHandler(db)
	preferenceHandler := handlers.NewPreferenceHandler(db)
	dismissedInsightsHandler := handlers.NewDismissedInsightsHandler(db)

	// AI-tier handlers (Sophia legacy).
	chatHandler := handlers.NewChatHandler(aiSvc)
	insightsHandler := handlers.NewInsightsHandler(aiSvc)
	categorizeHandler := handlers.NewCategorizeHandler(aiSvc)

	// Wire month-review service. Mirrors services/sophia/api/routes.go.
	// Self-loop default — Thesaurus IS this same process.
	mrThesaurusURL := os.Getenv("THESAURUS_URL")
	if mrThesaurusURL == "" {
		mrPort := os.Getenv("PORT")
		if mrPort == "" {
			mrPort = "3001"
		}
		mrThesaurusURL = "http://localhost:" + mrPort
	}
	mrDismiss := insights.NewThesaurusDismissalFetcher(mrThesaurusURL)
	mrDismiss.Client = &http.Client{Timeout: 5 * time.Second}
	var mrNarrator insights.Narrator = insights.NewTemplateNarrator()
	if os.Getenv("INSIGHTS_LLM_POLISH") == "1" {
		mrNarrator = insights.NewLLMNarrator(aiSvc)
	}
	mrService := monthreview.NewService(insights.NewEngine(mrDismiss), mrNarrator, aiSvc)
	monthReviewHandler := handlers.NewMonthReviewHandler(mrService, db)

	// Parse pipeline replaces the old Logos service. Wired into the upload
	// handler so file uploads kick off in-process AI parsing.
	pipeline := parse.New(db, aiSvc, modelName)
	pipeline.OnPersisted = func(userID, period, documentID string) {
		p, err := monthreview.ParsePeriod(period)
		if err != nil {
			return
		}
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 60*time.Second)
		defer cancel()
		// Fire-and-forget month review generation. Errors are non-fatal.
		_, _ = mrService.Generate(ctx, userID, p)
	}
	uploadHandler.SetPipeline(pipeline)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "localfinance"})
	})

	// First-run setup wizard endpoints. Public — they run before auth and
	// gate access to the rest of the app while Ollama is missing.
	// setup.New gets the Ollama host. Default to 127.0.0.1:11434 when env is
	// unset — important for .app launches via Finder where launchd provides a
	// minimal env. Without this, the wizard would show "install Ollama" even
	// when Ollama is running locally.
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://127.0.0.1:11434"
	}
	setupH := setup.New(ollamaHost)
	setupGroup := router.Group("/api/setup")
	{
		setupGroup.GET("/state", setupH.State)
		setupGroup.GET("/ollama-status", setupH.OllamaStatus)
		setupGroup.GET("/recommended", setupH.Recommended)
		setupGroup.POST("/pull-model", setupH.PullModel)
	}

	v1 := router.Group("/api/v1")
	{
		// Public auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/validate", authHandler.ValidateToken)
		}

		// Protected auth routes
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware(db))
		{
			authProtected.GET("/me", authHandler.GetMe)
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.POST("/refresh", authHandler.RefreshToken)
			authProtected.POST("/change-password", authHandler.ChangePassword)
		}

		// Internal service routes (called by Logos, Sophia — no auth required)
		documents := v1.Group("/documents")
		{
			documents.GET("/", documentHandler.ListUserDocuments)
			documents.GET("/:id", documentHandler.GetDocument)
			documents.PATCH("/:id/status", documentHandler.UpdateDocumentStatus)
		}
		v1.GET("/transactions/by-document", documentHandler.GetTransactionsByDocument)
		v1.POST("/transactions/bulk", transactionHandler.CreateBulkTransactions)

		// Internal query routes for Sophia AI service
		internal := v1.Group("/internal")
		{
			internal.GET("/transactions", transactionHandler.ListTransactions)
			internal.POST("/transactions/search", transactionHandler.SearchTransactions)
			internal.GET("/transactions/summary", transactionHandler.GetSpendingSummary)
			internal.GET("/category-rules/match", categoryRuleHandler.InternalMatchDescription)
			internal.POST("/conversations", conversationHandler.CreateConversation)
			internal.PATCH("/conversations/:id", conversationHandler.UpdateConversation)
			internal.POST("/conversations/:id/messages", conversationHandler.AddMessage)
			internal.GET("/preferences/:user_id", preferenceHandler.InternalGetPreference)

			// Dismissed insights — Sophia persists/un-persists user dismissals here.
			internal.POST("/users/:user_id/dismissed-insights", dismissedInsightsHandler.Create)
			internal.GET("/users/:user_id/dismissed-insights", dismissedInsightsHandler.List)
			internal.DELETE("/users/:user_id/dismissed-insights/:insight_key", dismissedInsightsHandler.Delete)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(db))
		{
			// File upload
			protected.POST("/upload", uploadHandler.UploadDocument)

			// Users
			users := protected.Group("/users")
			{
				users.POST("/", userHandler.CreateUser)
				users.GET("/:id", userHandler.GetUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeleteUser)
				users.GET("/", userHandler.ListUsers)
			}

			// Accounts
			accounts := protected.Group("/accounts")
			{
				accounts.POST("/", accountHandler.CreateAccount)
				accounts.GET("/:id", accountHandler.GetAccount)
				accounts.PUT("/:id", accountHandler.UpdateAccount)
				accounts.DELETE("/:id", accountHandler.DeleteAccount)
				accounts.GET("/", accountHandler.ListAccounts)
				accounts.GET("/user/:userId", accountHandler.GetAccountsByUser)
			}

			// Transactions
			transactions := protected.Group("/transactions")
			{
				transactions.POST("/", transactionHandler.CreateTransaction)
				transactions.GET("/:id", transactionHandler.GetTransaction)
				transactions.PUT("/:id", transactionHandler.UpdateTransaction)
				transactions.DELETE("/:id", transactionHandler.DeleteTransaction)
				transactions.POST("/:id/feedback", transactionHandler.SubmitFeedback)
				transactions.GET("/", transactionHandler.ListTransactions)
				transactions.GET("/account/:accountId", transactionHandler.GetTransactionsByAccount)
				transactions.GET("/category/:category", transactionHandler.GetTransactionsByCategory)
				transactions.POST("/search", transactionHandler.SearchTransactions)
				transactions.POST("/lookup", transactionHandler.LookupTransactions)
				transactions.GET("/summary", transactionHandler.GetSpendingSummary)
			}

			// Category Rules
			categoryRules := protected.Group("/category-rules")
			{
				categoryRules.POST("/", categoryRuleHandler.CreateRule)
				categoryRules.GET("/", categoryRuleHandler.ListRules)
				categoryRules.PUT("/:id", categoryRuleHandler.UpdateRule)
				categoryRules.DELETE("/:id", categoryRuleHandler.DeleteRule)
				categoryRules.GET("/match", categoryRuleHandler.MatchDescription)
			}

			// Conversations
			conversations := protected.Group("/conversations")
			{
				conversations.GET("/", conversationHandler.ListConversations)
				conversations.GET("/:id/messages", conversationHandler.GetMessages)
				conversations.DELETE("/:id", conversationHandler.DeleteConversation)
			}

			// Preferences
			protected.GET("/preferences", preferenceHandler.GetPreferences)
			protected.PUT("/preferences", preferenceHandler.UpdatePreferences)

			// Budgets
			budgets := protected.Group("/budgets")
			{
				budgets.POST("/", budgetHandler.CreateBudget)
				budgets.GET("/:id", budgetHandler.GetBudget)
				budgets.PUT("/:id", budgetHandler.UpdateBudget)
				budgets.DELETE("/:id", budgetHandler.DeleteBudget)
				budgets.GET("/", budgetHandler.ListBudgets)
				budgets.GET("/user/:userId", budgetHandler.GetBudgetsByUser)
			}
		}

		// AI-tier routes (formerly Sophia, port 8002). Now served on the unified
		// router so Iris can call /api/v1/chat/, /insights/, /categorize/, etc.
		chat := v1.Group("/chat")
		{
			chat.POST("/", chatHandler.HandleFinancialQuery)
			// Chat history lives at /api/v1/conversations/* (per-conversation
			// messages with proper persistence). The legacy /chat/history/:userId
			// endpoint was a stub that always returned []; deleted along with
			// its handler.
		}

		insightsGrp := v1.Group("/insights")
		{
			insightsGrp.POST("/", insightsHandler.GenerateInsights)
			insightsGrp.GET("/:userId", insightsHandler.GetUserInsights)
			// Param name is user_id to match dismissed_insights_handler's c.Param("user_id").
			insightsGrp.POST("/:user_id/dismiss", dismissedInsightsHandler.Create)
			insightsGrp.POST("/analyze", insightsHandler.AnalyzeSpendingPatterns)
		}

		categorize := v1.Group("/categorize")
		{
			categorize.POST("/", categorizeHandler.CategorizeTransaction)
			categorize.POST("/batch", categorizeHandler.CategorizeTransactionBatch)
		}

		// Month-in-Review: internal generate (no auth), public get/delete.
		v1.POST("/internal/month-review/generate", monthReviewHandler.Generate)
		v1.GET("/month-review/periods", monthReviewHandler.Periods)
		v1.GET("/month-review/:period", monthReviewHandler.Get)
		v1.DELETE("/month-review/:period", monthReviewHandler.Delete)

		// AI parsing endpoints (consumed by upload pipeline).
		// Parse paths use the fastest sweet-spot model installed (3-14B), NOT
		// the user's chat preference. Parse is latency-sensitive — a 26B model
		// on a Mac takes 5-10 minutes per statement, which is unusable.
		// Chat keeps the user's preference because chat tolerates longer
		// inference for higher-quality answers.
		pickParseModel := func() string {
			installed := ollama.InstalledModels(aiSvc.GetOllamaHost())
			if m := ai.SelectFastestSweetSpotModel(installed); m != "" {
				return m
			}
			return modelName // fallback to runtime default
		}

		v1.POST("/parse", func(c *gin.Context) {
			var req struct {
				Text   string `json:"text" binding:"required"`
				UserID string `json:"user_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			parseModel := pickParseModel()
			transactions, err := aiSvc.ParseTransactions(c.Request.Context(), req.Text, parseModel)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{
				"transactions": transactions,
				"count":        len(transactions),
				"model":        parseModel,
			})
		})

		v1.POST("/parse-images", func(c *gin.Context) {
			var req struct {
				Images []string `json:"images" binding:"required"`
				UserID string   `json:"user_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			if len(req.Images) == 0 {
				c.JSON(400, gin.H{"error": "images array is empty"})
				return
			}
			parseModel := pickParseModel()
			transactions, err := aiSvc.ParseTransactionsFromImages(c.Request.Context(), req.Images, parseModel)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{
				"transactions": transactions,
				"count":        len(transactions),
				"model":        parseModel,
			})
		})

		v1.GET("/models", func(c *gin.Context) {
			resp, err := http.Get(aiSvc.GetOllamaHost() + "/api/tags")
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to connect to Ollama"})
				return
			}
			defer resp.Body.Close()
			var result struct {
				Models []struct {
					Name       string `json:"name"`
					Size       int64  `json:"size"`
					ModifiedAt string `json:"modified_at"`
				} `json:"models"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&result)
			type ModelInfo struct {
				Name string `json:"name"`
				Size string `json:"size"`
			}
			var modelsOut []ModelInfo
			for _, m := range result.Models {
				sizeGB := float64(m.Size) / 1e9
				sizeStr := fmt.Sprintf("%.1f GB", sizeGB)
				if sizeGB < 1 {
					sizeStr = fmt.Sprintf("%.0f MB", float64(m.Size)/1e6)
				}
				modelsOut = append(modelsOut, ModelInfo{Name: m.Name, Size: sizeStr})
			}
			c.JSON(200, modelsOut)
		})

		v1.GET("/status/", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"ai_service": "operational",
				"model":      modelName,
				"endpoints":  []string{"/api/v1/chat", "/api/v1/insights", "/api/v1/categorize"},
			})
		})
	}

	// Static + SPA fallback — must be LAST so /api/* takes precedence.
	router.NoRoute(gin.WrapH(webui.Handler()))

	return router
}
