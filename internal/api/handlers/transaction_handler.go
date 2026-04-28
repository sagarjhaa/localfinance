package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/internal/data/models"
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

// CreateBulkTransactions creates transactions and optionally processes statement metadata
func (h *TransactionHandler) CreateBulkTransactions(c *gin.Context) {
	var req struct {
		AccountID    uuid.UUID            `json:"account_id" binding:"required"`
		Transactions []models.Transaction `json:"transactions" binding:"required"`
		Metadata     *struct {
			AccountType     string  `json:"account_type"`
			Institution     string  `json:"institution"`
			AccountNumber   string  `json:"account_number"`
			PeriodStart     *string `json:"period_start"`
			PeriodEnd       *string `json:"period_end"`
			PaymentDueDate  *string `json:"payment_due_date"`
			PaymentDueDay   int     `json:"payment_due_day"`
			CreditLimit     float64 `json:"credit_limit"`
			MinPaymentDue   float64 `json:"min_payment_due"`
			OpeningBalance  float64 `json:"opening_balance"`
			ClosingBalance  float64 `json:"closing_balance"`
			BillingCycleDay int     `json:"billing_cycle_day"`
		} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set account_id on all transactions
	for i := range req.Transactions {
		req.Transactions[i].AccountID = req.AccountID
	}

	if err := h.db.Create(&req.Transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transactions"})
		return
	}

	// Update account with detected metadata
	if req.Metadata != nil {
		updates := map[string]interface{}{}
		if req.Metadata.AccountType != "" {
			updates["type"] = req.Metadata.AccountType
		}
		if req.Metadata.Institution != "" {
			updates["institution"] = req.Metadata.Institution
		}
		if req.Metadata.AccountNumber != "" {
			updates["account_number"] = req.Metadata.AccountNumber
		}
		if req.Metadata.CreditLimit > 0 {
			updates["credit_limit"] = req.Metadata.CreditLimit
		}
		if req.Metadata.PaymentDueDay > 0 {
			updates["payment_due_day"] = req.Metadata.PaymentDueDay
		}
		if req.Metadata.BillingCycleDay > 0 {
			updates["billing_cycle_day"] = req.Metadata.BillingCycleDay
		}
		if len(updates) > 0 {
			h.db.Model(&models.Account{}).Where("id = ?", req.AccountID).Updates(updates)
		}

		// Create StatementPeriod record
		documentID := ""
		if len(req.Transactions) > 0 {
			documentID = req.Transactions[0].DocumentID
		}

		var totalCredits, totalDebits float64
		for _, t := range req.Transactions {
			if t.Amount > 0 {
				totalCredits += t.Amount
			} else {
				totalDebits += t.Amount
			}
		}

		period := models.StatementPeriod{
			AccountID:        req.AccountID,
			DocumentID:       documentID,
			OpeningBalance:   req.Metadata.OpeningBalance,
			ClosingBalance:   req.Metadata.ClosingBalance,
			TotalCredits:     totalCredits,
			TotalDebits:      totalDebits,
			MinPaymentDue:    req.Metadata.MinPaymentDue,
			TransactionCount: len(req.Transactions),
		}

		if req.Metadata.PeriodStart != nil {
			if t, err := time.Parse(time.RFC3339, *req.Metadata.PeriodStart); err == nil {
				period.PeriodStart = t
			}
		}
		if req.Metadata.PeriodEnd != nil {
			if t, err := time.Parse(time.RFC3339, *req.Metadata.PeriodEnd); err == nil {
				period.PeriodEnd = t
			}
		}
		if req.Metadata.PaymentDueDate != nil {
			if t, err := time.Parse(time.RFC3339, *req.Metadata.PaymentDueDate); err == nil {
				period.PaymentDueDate = &t
			}
		}

		h.db.Create(&period)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Transactions created",
		"count":        len(req.Transactions),
		"transactions": req.Transactions,
	})
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
// SubmitFeedback records the user's thumb up/down on a transaction. On
// "down" the body may also include corrected_category, reason, and a
// free-form note — those become the source of truth for chat and future
// parses.
//
// POST /api/v1/transactions/:id/feedback
// body: { signal: "up" | "down", reason?: "...", corrected_category?: "...", note?: "..." }
func (h *TransactionHandler) SubmitFeedback(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction id"})
		return
	}
	var req struct {
		Signal              string `json:"signal" binding:"required"`
		Reason              string `json:"reason,omitempty"`
		CorrectedCategory   string `json:"corrected_category,omitempty"`
		Note                string `json:"note,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Signal != "up" && req.Signal != "down" && req.Signal != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "signal must be 'up', 'down', or empty"})
		return
	}

	updates := map[string]interface{}{
		"user_signal":             req.Signal,
		"user_feedback_reason":    req.Reason,
		"user_corrected_category": req.CorrectedCategory,
		"user_feedback_note":      req.Note,
	}
	// If user gave a corrected category, also update the live Category so
	// reads everywhere (chat, insights, dashboard) reflect the user's
	// intent without us having to teach every consumer about the override.
	if req.CorrectedCategory != "" {
		updates["category"] = req.CorrectedCategory
	}
	if err := h.db.Model(&models.Transaction{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record feedback"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}
