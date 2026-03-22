package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/thesaurus/api/handlers"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
	// Initialize handlers
	userHandler := handlers.NewUserHandler(db)
	accountHandler := handlers.NewAccountHandler(db)
	transactionHandler := handlers.NewTransactionHandler(db)
	budgetHandler := handlers.NewBudgetHandler(db)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "thesaurus"})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// User routes
		users := v1.Group("/users")
		{
			users.POST("/", userHandler.CreateUser)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
			users.GET("/", userHandler.ListUsers)
		}

		// Account routes
		accounts := v1.Group("/accounts")
		{
			accounts.POST("/", accountHandler.CreateAccount)
			accounts.GET("/:id", accountHandler.GetAccount)
			accounts.PUT("/:id", accountHandler.UpdateAccount)
			accounts.DELETE("/:id", accountHandler.DeleteAccount)
			accounts.GET("/", accountHandler.ListAccounts)
			accounts.GET("/user/:userId", accountHandler.GetAccountsByUser)
		}

		// Transaction routes
		transactions := v1.Group("/transactions")
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

		// Budget routes
		budgets := v1.Group("/budgets")
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