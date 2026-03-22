package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/thesaurus/api/handlers"
	"github.com/sagarjhaa/localfinance/services/thesaurus/middleware"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
	// Initialize handlers
	userHandler := handlers.NewUserHandler(db)
	accountHandler := handlers.NewAccountHandler(db)
	transactionHandler := handlers.NewTransactionHandler(db)
	budgetHandler := handlers.NewBudgetHandler(db)
	authHandler := handlers.NewAuthHandler(db)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "thesaurus"})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public auth routes (no authentication required)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/validate", authHandler.ValidateToken)
		}

		// Protected auth routes (authentication required)
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware(db))
		{
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.POST("/refresh", authHandler.RefreshToken)
			authProtected.POST("/change-password", authHandler.ChangePassword)
		}

		// Protected user routes
		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(db))
		{
			users.POST("/", userHandler.CreateUser)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
			users.GET("/", userHandler.ListUsers)
		}

		// Protected account routes
		accounts := v1.Group("/accounts")
		accounts.Use(middleware.AuthMiddleware(db))
		{
			accounts.POST("/", accountHandler.CreateAccount)
			accounts.GET("/:id", accountHandler.GetAccount)
			accounts.PUT("/:id", accountHandler.UpdateAccount)
			accounts.DELETE("/:id", accountHandler.DeleteAccount)
			accounts.GET("/", accountHandler.ListAccounts)
			accounts.GET("/user/:userId", accountHandler.GetAccountsByUser)
		}

		// Protected transaction routes
		transactions := v1.Group("/transactions")
		transactions.Use(middleware.AuthMiddleware(db))
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

		// Protected budget routes
		budgets := v1.Group("/budgets")
		budgets.Use(middleware.AuthMiddleware(db))
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