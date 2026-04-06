package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/thesaurus/auth"
	"github.com/sagarjhaa/localfinance/services/thesaurus/models"
	"gorm.io/gorm"
)

// AuthMiddleware validates JWT tokens and sets user context
func AuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header provided"})
			c.Abort()
			return
		}

		// Extract token from header
		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Validate JWT token
		claims, err := auth.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Check if session is still active in database
		var session models.UserSession
		if err := db.Where("token = ? AND is_active = ? AND expires_at > ?", 
			token, true, time.Now()).First(&session).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
			c.Abort()
			return
		}

		// Check if user still exists and is active
		var user models.User
		if err := db.Where("id = ? AND is_active = ?", claims.UserID, true).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found or inactive"})
			c.Abort()
			return
		}

		// Set user context for downstream handlers
		c.Set("user_id", claims.UserID.String())
		c.Set("user_email", claims.Email)
		c.Set("user", user)

		c.Next()
	}
}

// OptionalAuthMiddleware validates JWT tokens but doesn't require them
func OptionalAuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue without setting user context
			c.Next()
			return
		}

		// Extract token from header
		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			// Invalid header format, continue without setting user context
			c.Next()
			return
		}

		// Validate JWT token
		claims, err := auth.ValidateToken(token)
		if err != nil {
			// Invalid token, continue without setting user context
			c.Next()
			return
		}

		// Check if session is still active in database
		var session models.UserSession
		if err := db.Where("token = ? AND is_active = ? AND expires_at > ?", 
			token, true, time.Now()).First(&session).Error; err != nil {
			// Session expired, continue without setting user context
			c.Next()
			return
		}

		// Check if user still exists and is active
		var user models.User
		if err := db.Where("id = ? AND is_active = ?", claims.UserID, true).First(&user).Error; err != nil {
			// User not found, continue without setting user context
			c.Next()
			return
		}

		// Set user context for downstream handlers
		c.Set("user_id", claims.UserID.String())
		c.Set("user_email", claims.Email)
		c.Set("user", user)

		c.Next()
	}
}