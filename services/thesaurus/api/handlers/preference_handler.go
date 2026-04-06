package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/gorm"
)

type PreferenceHandler struct {
	db *gorm.DB
}

func NewPreferenceHandler(db *gorm.DB) *PreferenceHandler {
	return &PreferenceHandler{db: db}
}

// GET /api/v1/preferences — get user's preferences (auth required)
func (h *PreferenceHandler) GetPreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found"})
		return
	}

	var pref models.UserPreference
	result := h.db.Where("user_id = ?", userID).First(&pref)
	if result.Error != nil {
		// Return default if no preference exists
		c.JSON(http.StatusOK, gin.H{"chat_model": "llama3.2:1b"})
		return
	}

	c.JSON(http.StatusOK, pref)
}

// PUT /api/v1/preferences — create or update preferences (auth required)
func (h *PreferenceHandler) UpdatePreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found"})
		return
	}

	var req struct {
		ChatModel string `json:"chat_model" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, _ := uuid.Parse(userID)
	var pref models.UserPreference
	result := h.db.Where("user_id = ?", userID).First(&pref)

	if result.Error != nil {
		// Create new preference
		pref = models.UserPreference{
			UserID:    uid,
			ChatModel: req.ChatModel,
		}
		if err := h.db.Create(&pref).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Update existing
		pref.ChatModel = req.ChatModel
		h.db.Save(&pref)
	}

	c.JSON(http.StatusOK, pref)
}

// GET /internal/preferences/:user_id — get user's chat model (called by Sophia, no auth)
func (h *PreferenceHandler) InternalGetPreference(c *gin.Context) {
	userID := c.Param("user_id")

	var pref models.UserPreference
	result := h.db.Where("user_id = ?", userID).First(&pref)
	if result.Error != nil {
		c.JSON(http.StatusOK, gin.H{"chat_model": "llama3.2:1b"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chat_model": pref.ChatModel})
}
