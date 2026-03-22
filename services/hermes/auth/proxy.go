package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/hermes/config"
)

// AuthProxy handles authentication requests by proxying to Thesaurus service
type AuthProxy struct {
	thesaurusURL string
	httpClient   *http.Client
}

// NewAuthProxy creates a new authentication proxy
func NewAuthProxy(cfg *config.Config) *AuthProxy {
	return &AuthProxy{
		thesaurusURL: cfg.Services.Thesaurus,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoginHandler proxies login requests to Thesaurus
func (ap *AuthProxy) LoginHandler(c *gin.Context) {
	ap.proxyRequest(c, "/api/v1/auth/login")
}

// RegisterHandler proxies register requests to Thesaurus
func (ap *AuthProxy) RegisterHandler(c *gin.Context) {
	ap.proxyRequest(c, "/api/v1/auth/register")
}

// LogoutHandler proxies logout requests to Thesaurus
func (ap *AuthProxy) LogoutHandler(c *gin.Context) {
	ap.proxyRequest(c, "/api/v1/auth/logout")
}

// RefreshHandler proxies token refresh requests to Thesaurus
func (ap *AuthProxy) RefreshHandler(c *gin.Context) {
	ap.proxyRequest(c, "/api/v1/auth/refresh")
}

// ValidateHandler proxies token validation requests to Thesaurus
func (ap *AuthProxy) ValidateHandler(c *gin.Context) {
	ap.proxyRequest(c, "/api/v1/auth/validate")
}

// ChangePasswordHandler proxies change password requests to Thesaurus
func (ap *AuthProxy) ChangePasswordHandler(c *gin.Context) {
	ap.proxyRequest(c, "/api/v1/auth/change-password")
}

// proxyRequest handles the actual proxying logic
func (ap *AuthProxy) proxyRequest(c *gin.Context, endpoint string) {
	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Create request to Thesaurus
	url := ap.thesaurusURL + endpoint
	req, err := http.NewRequest(c.Request.Method, url, bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make request
	resp, err := ap.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Authentication service unavailable",
			"details": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Return response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// AuthMiddleware validates JWT tokens by calling Thesaurus
func (ap *AuthProxy) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header provided"})
			c.Abort()
			return
		}

		// Validate token with Thesaurus
		validateReq := map[string]string{
			"token": extractTokenFromHeader(authHeader),
		}

		jsonBody, _ := json.Marshal(validateReq)
		req, err := http.NewRequest("POST", ap.thesaurusURL+"/api/v1/auth/validate", bytes.NewBuffer(jsonBody))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate token"})
			c.Abort()
			return
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := ap.httpClient.Do(req)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Authentication service unavailable"})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token validation failed"})
			c.Abort()
			return
		}

		var validationResp struct {
			Valid  bool   `json:"valid"`
			UserID string `json:"user_id"`
			User   struct {
				ID        string `json:"id"`
				Email     string `json:"email"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			} `json:"user"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&validationResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode validation response"})
			c.Abort()
			return
		}

		if !validationResp.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user context for downstream handlers
		c.Set("user_id", validationResp.UserID)
		c.Set("user_email", validationResp.User.Email)
		c.Set("user", validationResp.User)

		c.Next()
	}
}

// extractTokenFromHeader extracts Bearer token from Authorization header
func extractTokenFromHeader(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}