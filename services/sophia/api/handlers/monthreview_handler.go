package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/sophia/monthreview"
)

// MonthReviewHandler exposes the per-period review endpoints.
type MonthReviewHandler struct {
	svc *monthreview.Service
}

// NewMonthReviewHandler wires a handler to its service.
func NewMonthReviewHandler(svc *monthreview.Service) *MonthReviewHandler {
	return &MonthReviewHandler{svc: svc}
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
