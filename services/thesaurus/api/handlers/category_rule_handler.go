package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/gorm"
)

type CategoryRuleHandler struct {
	db *gorm.DB
}

func NewCategoryRuleHandler(db *gorm.DB) *CategoryRuleHandler {
	return &CategoryRuleHandler{db: db}
}

func (h *CategoryRuleHandler) CreateRule(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var req struct {
		Pattern  string `json:"pattern" binding:"required"`
		Category string `json:"category" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := models.CategoryRule{
		UserID:   userID,
		Pattern:  req.Pattern,
		Category: req.Category,
		IsActive: true,
	}

	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

func (h *CategoryRuleHandler) ListRules(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var rules []models.CategoryRule
	if err := h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list category rules"})
		return
	}

	c.JSON(http.StatusOK, rules)
}

func (h *CategoryRuleHandler) UpdateRule(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID"})
		return
	}

	var rule models.CategoryRule
	if err := h.db.Where("id = ? AND user_id = ?", ruleID, userID).First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category rule not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find category rule"})
		return
	}

	var req struct {
		Pattern  *string `json:"pattern"`
		Category *string `json:"category"`
		IsActive *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Pattern != nil {
		updates["pattern"] = *req.Pattern
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.Model(&rule).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (h *CategoryRuleHandler) DeleteRule(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID"})
		return
	}

	result := h.db.Where("id = ? AND user_id = ?", ruleID, userID).Delete(&models.CategoryRule{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category rule"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category rule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category rule deleted successfully"})
}

func (h *CategoryRuleHandler) MatchDescription(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	description := c.Query("description")
	if description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description query parameter is required"})
		return
	}

	var rule models.CategoryRule
	if err := h.db.Where("user_id = ? AND is_active = ? AND ? ILIKE '%' || pattern || '%'", userID, true, description).
		First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"matched": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to match category rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"matched":  true,
		"category": rule.Category,
		"rule_id":  rule.ID,
	})
}

// InternalMatchDescription handles internal match requests from other services (no auth)
func (h *CategoryRuleHandler) InternalMatchDescription(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query parameter is required"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	description := c.Query("description")
	if description == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description query parameter is required"})
		return
	}

	var rule models.CategoryRule
	if err := h.db.Where("user_id = ? AND is_active = ? AND ? ILIKE '%' || pattern || '%'", userID, true, description).
		First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"matched": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to match category rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"matched":  true,
		"category": rule.Category,
		"rule_id":  rule.ID,
	})
}
