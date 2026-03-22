package api

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/hermes/config"
)

// SetupRoutes configures the API gateway and frontend routes
func SetupRoutes(router *gin.Engine, cfg *config.Config) {
	// Serve static frontend files
	router.Static("/static", "./frontend/build/static")
	router.StaticFile("/favicon.ico", "./frontend/build/favicon.ico")
	
	// API Gateway routes
	api := router.Group("/api")
	{
		// Thesaurus (CRUD/Database) service routes
		api.Any("/v1/transactions/*path", proxyToService(cfg.Services.Thesaurus))
		api.Any("/v1/accounts/*path", proxyToService(cfg.Services.Thesaurus))
		api.Any("/v1/budgets/*path", proxyToService(cfg.Services.Thesaurus))
		api.Any("/v1/users/*path", proxyToService(cfg.Services.Thesaurus))

		// Sophia (AI) service routes  
		api.Any("/v1/ai/*path", proxyToService(cfg.Services.Sophia))
		api.Any("/v1/insights/*path", proxyToService(cfg.Services.Sophia))
		api.Any("/v1/chat/*path", proxyToService(cfg.Services.Sophia))

		// Logos (Document Processing) service routes
		api.Any("/v1/documents/*path", proxyToService(cfg.Services.Logos))
		api.Any("/v1/upload/*path", proxyToService(cfg.Services.Logos))
		api.Any("/v1/processing/*path", proxyToService(cfg.Services.Logos))
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "hermes",
			"services": gin.H{
				"thesaurus": cfg.Services.Thesaurus + "/health",
				"sophia":    cfg.Services.Sophia + "/health",
				"logos":     cfg.Services.Logos + "/health",
			},
		})
	})

	// Frontend route (catch-all for React Router)
	router.NoRoute(func(c *gin.Context) {
		// If it's an API request and not found, return 404
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		// Otherwise, serve the React app
		c.File("./frontend/build/index.html")
	})
}

// proxyToService creates a reverse proxy to a backend service
func proxyToService(serviceURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		remote, err := url.Parse(serviceURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid service URL"})
			return
		}

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(remote)
		
		// Modify the request
		c.Request.URL.Host = remote.Host
		c.Request.URL.Scheme = remote.Scheme
		c.Request.Header.Set("X-Forwarded-Host", c.Request.Header.Get("Host"))
		c.Request.Host = remote.Host

		// Custom director to handle path rewriting
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			
			// Remove the service prefix from the path
			originalPath := req.URL.Path
			if strings.HasPrefix(originalPath, "/api/v1/ai") {
				req.URL.Path = strings.Replace(originalPath, "/api/v1/ai", "/api/v1", 1)
			} else if strings.HasPrefix(originalPath, "/api/v1/documents") {
				req.URL.Path = strings.Replace(originalPath, "/api/v1/documents", "/api/v1", 1)
			} else if strings.HasPrefix(originalPath, "/api/v1/upload") {
				req.URL.Path = strings.Replace(originalPath, "/api/v1/upload", "/api/v1", 1)
			} else if strings.HasPrefix(originalPath, "/api/v1/processing") {
				req.URL.Path = strings.Replace(originalPath, "/api/v1/processing", "/api/v1", 1)
			} else if strings.HasPrefix(originalPath, "/api/v1/insights") {
				req.URL.Path = strings.Replace(originalPath, "/api/v1/insights", "/api/v1", 1)
			} else if strings.HasPrefix(originalPath, "/api/v1/chat") {
				req.URL.Path = strings.Replace(originalPath, "/api/v1/chat", "/api/v1", 1)
			}
		}

		// Handle errors
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			io.WriteString(w, `{"error":"Service unavailable"}`)
		}

		// Serve the request
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}