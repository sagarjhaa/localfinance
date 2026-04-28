package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/monthreview"
	"gorm.io/gorm"
)

// MonthReviewHandler exposes the per-period review endpoints.
type MonthReviewHandler struct {
	svc *monthreview.Service
	db  *gorm.DB
}

// NewMonthReviewHandler wires a handler to its service. db is used by the
// /periods endpoint to enumerate months that have transactions.
func NewMonthReviewHandler(svc *monthreview.Service, db *gorm.DB) *MonthReviewHandler {
	return &MonthReviewHandler{svc: svc, db: db}
}

// GenerateRequest is the body shape for the internal POST.
type GenerateRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Period string `json:"period" binding:"required"`
}

// Generate is the internal endpoint called by Logos after a successful upload.
// No auth (mounted under /internal/).
func (h *MonthReviewHandler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	period, err := monthreview.ParsePeriod(req.Period)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	review, err := h.svc.Generate(c.Request.Context(), req.UserID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, review)
}

// userIDFromContext extracts the user ID from the request context. Tries the
// JWT-populated key first ("user_id" set by auth middleware) and falls back
// to a query param so tests / internal callers can exercise the endpoint
// without forging a JWT.
func userIDFromContext(c *gin.Context) string {
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return c.Query("user_id")
}

// Get returns the cached review (generating if missing) for the JWT user.
func (h *MonthReviewHandler) Get(c *gin.Context) {
	userID := userIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}
	period, err := monthreview.ParsePeriod(c.Param("period"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	review, err := h.svc.GetOrGenerate(c.Request.Context(), userID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, review)
}

// Periods returns the distinct YYYY-MM months for which the user has at
// least one transaction, sorted newest-first. Used by the UI to populate
// the month-review dropdown so users can only select periods that
// actually have data.
func (h *MonthReviewHandler) Periods(c *gin.Context) {
	userID := userIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}
	var rows []struct {
		Period string
	}
	// to_char works on Postgres; the embedded postgres we ship is real
	// Postgres so this is portable across our dev + .app paths.
	err := h.db.Raw(`
		SELECT DISTINCT to_char(date, 'YYYY-MM') AS period
		FROM transactions
		WHERE user_id = ?
		ORDER BY period DESC
	`, userID).Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	periods := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Period != "" {
			periods = append(periods, r.Period)
		}
	}
	c.JSON(http.StatusOK, gin.H{"periods": periods})
}

// Delete invalidates the cache so the next GET regenerates.
func (h *MonthReviewHandler) Delete(c *gin.Context) {
	userID := userIDFromContext(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in context"})
		return
	}
	period, err := monthreview.ParsePeriod(c.Param("period"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.svc.Invalidate(userID, period)
	c.JSON(http.StatusOK, gin.H{"status": "invalidated", "period": period.String()})
}
