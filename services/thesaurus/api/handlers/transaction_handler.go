package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/gorm"
)

type TransactionHandler struct {
	db *gorm.DB
}

func NewTransactionHandler(db *gorm.DB) *TransactionHandler {
	return &TransactionHandler{db: db}
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	var transaction models.Transaction
	if err := c.ShouldBindJSON(&transaction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Create(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	var transaction models.Transaction
	if err := h.db.Preload("Account").First(&transaction, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction"})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	var transaction models.Transaction
	if err := h.db.First(&transaction, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find transaction"})
		return
	}

	var updateData models.Transaction
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Model(&transaction).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	if err := h.db.Delete(&models.Transaction{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction deleted successfully"})
}

func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	var filter models.TransactionFilter
	
	// Parse query parameters
	if userID := c.Query("user_id"); userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			filter.UserID = &uid
		}
	}
	
	if accountID := c.Query("account_id"); accountID != "" {
		if aid, err := uuid.Parse(accountID); err == nil {
			filter.AccountID = &aid
		}
	}
	
	if category := c.Query("category"); category != "" {
		filter.Category = &category
	}
	
	if startDate := c.Query("start_date"); startDate != "" {
		if sd, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartDate = &sd
		}
	}
	
	if endDate := c.Query("end_date"); endDate != "" {
		if ed, err := time.Parse("2006-01-02", endDate); err == nil {
			filter.EndDate = &ed
		}
	}
	
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	} else {
		filter.Limit = 100 // Default limit
	}
	
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Offset = o
		}
	}

	transactions, err := h.getTransactionsWithFilter(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   transactions,
		"filter": filter,
	})
}

func (h *TransactionHandler) GetTransactionsByAccount(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID"})
		return
	}

	var transactions []models.Transaction
	query := h.db.Where("account_id = ?", accountID).Order("date DESC")
	
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			query = query.Limit(l)
		}
	}

	if err := query.Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

func (h *TransactionHandler) GetTransactionsByCategory(c *gin.Context) {
	category := c.Param("category")
	
	var transactions []models.Transaction
	if err := h.db.Where("category = ?", category).Order("date DESC").Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

func (h *TransactionHandler) SearchTransactions(c *gin.Context) {
	var filter models.TransactionFilter
	if err := c.ShouldBindJSON(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transactions, err := h.getTransactionsWithFilter(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search transactions"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

func (h *TransactionHandler) GetSpendingSummary(c *gin.Context) {
	userID := c.Query("user_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var summaries []models.SpendingSummary
	query := `
		SELECT 
			category,
			SUM(ABS(amount)) as total_amount,
			COUNT(*) as count,
			(SUM(ABS(amount)) * 100.0 / (
				SELECT SUM(ABS(amount)) 
				FROM transactions t2 
				JOIN accounts a2 ON t2.account_id = a2.id 
				WHERE a2.user_id = ? AND t2.date BETWEEN ? AND ?
			)) as percentage
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE a.user_id = ? AND t.date BETWEEN ? AND ? AND amount < 0
		GROUP BY category
		ORDER BY total_amount DESC
	`

	if err := h.db.Raw(query, userID, startDate, endDate, userID, startDate, endDate).Scan(&summaries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get spending summary"})
		return
	}

	c.JSON(http.StatusOK, summaries)
}

func (h *TransactionHandler) getTransactionsWithFilter(filter models.TransactionFilter) ([]models.Transaction, error) {
	var transactions []models.Transaction
	
	query := h.db.Model(&models.Transaction{}).Preload("Account")

	if filter.UserID != nil {
		query = query.Joins("JOIN accounts ON transactions.account_id = accounts.id").
			Where("accounts.user_id = ?", *filter.UserID)
	}

	if filter.AccountID != nil {
		query = query.Where("account_id = ?", *filter.AccountID)
	}

	if filter.Category != nil {
		query = query.Where("category = ?", *filter.Category)
	}

	if filter.StartDate != nil {
		query = query.Where("date >= ?", *filter.StartDate)
	}

	if filter.EndDate != nil {
		query = query.Where("date <= ?", *filter.EndDate)
	}

	if filter.MinAmount != nil {
		query = query.Where("amount >= ?", *filter.MinAmount)
	}

	if filter.MaxAmount != nil {
		query = query.Where("amount <= ?", *filter.MaxAmount)
	}

	if filter.Description != nil {
		query = query.Where("description ILIKE ?", "%"+*filter.Description+"%")
	}

	query = query.Order("date DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Find(&transactions).Error; err != nil {
		return nil, err
	}

	return transactions, nil
}