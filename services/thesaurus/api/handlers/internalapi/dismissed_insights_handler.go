// Package internalapi hosts handlers for the /internal/* route group.
// These endpoints are called by sibling services (Sophia, Logos) and
// are NOT protected by AuthMiddleware. They must be mounted under
// /internal/ in api/routes.go.
//
// The directory is named "internalapi" rather than "internal" because
// Go reserves "internal" as a package-visibility keyword, which would
// block imports from sibling api/ packages.
package internalapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"github.com/sagarjhaa/localfinance/services/thesaurus/repository"
	"gorm.io/gorm"
)

// DismissedInsightsHandler exposes CRUD for dismissed insights.
type DismissedInsightsHandler struct {
	repo *repository.DismissedInsightRepo
}

func NewDismissedInsightsHandler(db *gorm.DB) *DismissedInsightsHandler {
	return &DismissedInsightsHandler{repo: repository.NewDismissedInsightRepo(db)}
}

// Create handles POST /internal/users/:user_id/dismissed-insights.
//
// Body: {"insight_key": "...", "rule_id": "..."}
//
// Returns 201 with the row on first dismissal, 200 with the existing
// row if (user_id, insight_key) was already dismissed.
func (h *DismissedInsightsHandler) Create(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req struct {
		InsightKey string `json:"insight_key" binding:"required"`
		RuleID     string `json:"rule_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check existence first so we can distinguish 200 vs 201 cleanly.
	existed, err := h.repo.Exists(c.Request.Context(), userID, req.InsightKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check dismissal"})
		return
	}

	row := &models.DismissedInsight{
		UserID:     userID,
		InsightKey: req.InsightKey,
		RuleID:     req.RuleID,
	}
	if err := h.repo.Create(c.Request.Context(), row); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to dismiss insight"})
		return
	}

	if existed {
		c.JSON(http.StatusOK, row)
		return
	}
	c.JSON(http.StatusCreated, row)
}

// List handles GET /internal/users/:user_id/dismissed-insights.
func (h *DismissedInsightsHandler) List(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	rows, err := h.repo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list dismissals"})
		return
	}
	if rows == nil {
		rows = []models.DismissedInsight{}
	}
	c.JSON(http.StatusOK, rows)
}

// Delete handles DELETE /internal/users/:user_id/dismissed-insights/:insight_key.
// Used to un-dismiss an insight. Returns 404 if no such dismissal exists.
func (h *DismissedInsightsHandler) Delete(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	insightKey := c.Param("insight_key")
	if insightKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insight_key is required"})
		return
	}

	affected, err := h.repo.Delete(c.Request.Context(), userID, insightKey)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete dismissal"})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "dismissal not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "dismissal removed"})
}
