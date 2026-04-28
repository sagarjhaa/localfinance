package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/data/models"
	"gorm.io/gorm"
)

type DocumentHandler struct {
	db *gorm.DB
}

func NewDocumentHandler(db *gorm.DB) *DocumentHandler {
	return &DocumentHandler{db: db}
}

// GetDocument returns a document by ID
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	id := c.Param("id")

	var doc models.Document
	if err := h.db.Where("id = ?", id).First(&doc).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// UpdateDocumentStatus updates a document's processing status (called by Logos)
func (h *DocumentHandler) UpdateDocumentStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status        string `json:"status"`
		ErrorMessage  string `json:"error_message"`
		ExtractedText string `json:"extracted_text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"status": req.Status,
	}
	if req.ErrorMessage != "" {
		updates["error_message"] = req.ErrorMessage
	}
	if req.ExtractedText != "" {
		updates["extracted_text"] = req.ExtractedText
	}

	result := h.db.Model(&models.Document{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated", "status": req.Status})
}

// GetTransactionsByDocument returns transactions for a specific document
// with the account record preloaded so the UI can show "Capital One Savor"
// instead of an account UUID.
func (h *DocumentHandler) GetTransactionsByDocument(c *gin.Context) {
	documentID := c.Query("document_id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document_id query parameter required"})
		return
	}

	var transactions []models.Transaction
	if err := h.db.Preload("Account").Where("document_id = ?", documentID).Order("date ASC").Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
		"document_id":  documentID,
	})
}

// DocumentSummary is the row shape returned by ListUserDocuments. Each row
// is one uploaded statement enriched with its parse outcome — txn count,
// money totals (split by direction), and the date range its transactions
// cover. The UI groups these by transaction period to show a "by month"
// view, but consumers can also group by upload date if preferred.
type DocumentSummary struct {
	ID               string  `json:"id"`
	OriginalFilename string  `json:"original_filename"`
	Status           string  `json:"status"`
	ErrorMessage     string  `json:"error_message,omitempty"`
	CreatedAt        string  `json:"created_at"`
	TxnCount         int     `json:"txn_count"`
	TxnPeriodStart   string  `json:"txn_period_start,omitempty"`
	TxnPeriodEnd     string  `json:"txn_period_end,omitempty"`
	TotalSpend       float64 `json:"total_spend"`
	TotalIncome      float64 `json:"total_income"`
	AccountName      string  `json:"account_name,omitempty"`
}

// ListUserDocuments returns every statement uploaded by the user, newest-
// first. Each row carries enough metadata for the UI to render a list
// without a second roundtrip per row.
func (h *DocumentHandler) ListUserDocuments(c *gin.Context) {
	userID := userIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}

	var rows []DocumentSummary
	err := h.db.Raw(`
		SELECT
		  d.id                                                                AS id,
		  COALESCE(NULLIF(d.original_filename, ''), d.filename)                AS original_filename,
		  d.status                                                             AS status,
		  COALESCE(d.error_message, '')                                        AS error_message,
		  to_char(d.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')                  AS created_at,
		  COALESCE(stats.txn_count, 0)                                         AS txn_count,
		  COALESCE(to_char(stats.period_start, 'YYYY-MM-DD'), '')              AS txn_period_start,
		  COALESCE(to_char(stats.period_end,   'YYYY-MM-DD'), '')              AS txn_period_end,
		  COALESCE(stats.total_spend, 0)                                       AS total_spend,
		  COALESCE(stats.total_income, 0)                                      AS total_income,
		  COALESCE(NULLIF(acct.name, ''), acct.institution, '')                AS account_name
		FROM documents d
		LEFT JOIN LATERAL (
		  SELECT
		    count(*)                                       AS txn_count,
		    min(t.date)                                    AS period_start,
		    max(t.date)                                    AS period_end,
		    sum(CASE WHEN t.amount > 0 THEN t.amount ELSE 0 END)  AS total_spend,
		    sum(CASE WHEN t.amount < 0 THEN -t.amount ELSE 0 END) AS total_income,
		    (array_agg(t.account_id ORDER BY t.date))[1]  AS account_id
		  FROM transactions t
		  WHERE t.document_id = d.id
		) stats ON true
		LEFT JOIN accounts acct ON acct.id = stats.account_id
		WHERE d.user_id = ?
		ORDER BY d.created_at DESC
	`, userID).Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"documents": rows})
}
