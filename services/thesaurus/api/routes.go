package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/thesaurus/api/handlers"
	"github.com/sagarjhaa/localfinance/services/thesaurus/middleware"
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
