package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/auth"
	"github.com/sagarjhaa/localfinance/internal/data/models"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

// Login authenticates a user and returns a JWT token
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by email
	var user models.User
	if err := h.db.Where("email = ? AND is_active = ?", req.Email, true).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Verify password
	valid, err := auth.VerifyPassword(req.Password, user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password verification error"})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate JWT token
	token, expiresAt, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation error"})
		return
	}

	// Store session in database
	session := models.UserSession{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Unix(expiresAt, 0),
		IsActive:  true,
	}

	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session creation error"})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, models.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	})
}

// Register creates a new user account
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password hashing error"})
		return
	}

	// Create new user
	user := models.User{
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		IsActive:  true,
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User creation error"})
		return
	}

	// Generate JWT token for immediate login
	token, expiresAt, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation error"})
		return
	}

	// Store session
	session := models.UserSession{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Unix(expiresAt, 0),
		IsActive:  true,
	}

	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session creation error"})
		return
	}

	c.JSON(http.StatusCreated, models.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	})
}

// Logout invalidates a user's session
func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header"})
		return
	}

	token, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
		return
	}

	// Invalidate session in database
	if err := h.db.Model(&models.UserSession{}).
		Where("token = ?", token).
		Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Logout error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// ValidateToken validates a JWT token
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	var req models.TokenValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate JWT token
	claims, err := auth.ValidateToken(req.Token)
	if err != nil {
		c.JSON(http.StatusOK, models.TokenValidationResponse{
			Valid: false,
		})
		return
	}

	// Check if session is still active in database
	var session models.UserSession
	if err := h.db.Where("token = ? AND is_active = ? AND expires_at > ?", 
		req.Token, true, time.Now()).First(&session).Error; err != nil {
		c.JSON(http.StatusOK, models.TokenValidationResponse{
			Valid: false,
		})
		return
	}

	// Get user information
	var user models.User
	if err := h.db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, models.TokenValidationResponse{
			Valid: false,
		})
		return
	}

	c.JSON(http.StatusOK, models.TokenValidationResponse{
		Valid:  true,
		UserID: claims.UserID,
		User:   &user,
	})
}

// RefreshToken generates a new token for a user
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header"})
		return
	}

	token, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
		return
	}

	// Generate new token
	newToken, expiresAt, err := auth.RefreshToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Update session in database
	if err := h.db.Model(&models.UserSession{}).
		Where("token = ?", token).
		Updates(map[string]interface{}{
			"token":      newToken,
			"expires_at": time.Unix(expiresAt, 0),
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token refresh error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      newToken,
		"expires_at": expiresAt,
	})
}

// ChangePassword allows a user to change their password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get user
	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify current password
	valid, err := auth.VerifyPassword(req.CurrentPassword, user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password verification error"})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password hashing error"})
		return
	}

	// Update password
	if err := h.db.Model(&user).Update("password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password update error"})
		return
	}

	// Invalidate other sessions for this user, but keep the current one active
	// so the client doesn't have to re-login after a password change.
	currentToken, _ := auth.ExtractTokenFromHeader(c.GetHeader("Authorization"))
	q := h.db.Model(&models.UserSession{}).Where("user_id = ?", userID)
	if currentToken != "" {
		q = q.Where("token <> ?", currentToken)
	}
	if err := q.Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session cleanup error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}