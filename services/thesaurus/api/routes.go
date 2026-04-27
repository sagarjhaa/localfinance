package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/api/handlers"
	"github.com/sagarjhaa/localfinance/internal/api/middleware"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
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

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "thesaurus"})
	})

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
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.POST("/refresh", authHandler.RefreshToken)
			authProtected.POST("/change-password", authHandler.ChangePassword)
		}

		// Internal service routes (called by Logos, Sophia — no auth required)
		documents := v1.Group("/documents")
		{
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
				transactions.GET("/", transactionHandler.ListTransactions)
				transactions.GET("/account/:accountId", transactionHandler.GetTransactionsByAccount)
				transactions.GET("/category/:category", transactionHandler.GetTransactionsByCategory)
				transactions.POST("/search", transactionHandler.SearchTransactions)
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
	}
}
