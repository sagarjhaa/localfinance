package api

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/hermes/auth"
	"github.com/sagarjhaa/localfinance/services/hermes/config"
)

// SetupRoutes configures the API gateway and frontend routes with authentication
func SetupRoutes(router *gin.Engine, cfg *config.Config) {
	// CORS configuration for frontend
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:3000", "http://localhost:3001"}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	// Initialize auth proxy
	authProxy := auth.NewAuthProxy(cfg)

	// Serve static frontend files
	router.Static("/static", "./frontend/build/static")
	router.StaticFile("/favicon.ico", "./frontend/build/favicon.ico")
	
	// API Gateway routes
	api := router.Group("/api")
	{
		// Public authentication routes (no auth required)
		authRoutes := api.Group("/auth")
		{
			authRoutes.POST("/login", authProxy.LoginHandler)
			authRoutes.POST("/register", authProxy.RegisterHandler)
			authRoutes.POST("/validate", authProxy.ValidateHandler)
		}

		// Protected authentication routes (require auth)
		authProtected := api.Group("/auth")
		authProtected.Use(authProxy.AuthMiddleware())
		{
			authProtected.POST("/logout", authProxy.LogoutHandler)
			authProtected.POST("/refresh", authProxy.RefreshHandler)
			authProtected.POST("/change-password", authProxy.ChangePasswordHandler)
		}

		// Protected API routes (require authentication)
		protected := api.Group("/v1")
		protected.Use(authProxy.AuthMiddleware())
		{
			// Thesaurus (CRUD/Database) service routes
			protected.Any("/transactions/*path", proxyToService(cfg.Services.Thesaurus, "/api/v1/transactions"))
			protected.Any("/accounts/*path", proxyToService(cfg.Services.Thesaurus, "/api/v1/accounts"))
			protected.Any("/budgets/*path", proxyToService(cfg.Services.Thesaurus, "/api/v1/budgets"))
			protected.Any("/users/*path", proxyToService(cfg.Services.Thesaurus, "/api/v1/users"))

			// Sophia (AI) service routes  
			protected.Any("/ai/*path", proxyToService(cfg.Services.Sophia, "/api/v1"))
			protected.Any("/insights/*path", proxyToService(cfg.Services.Sophia, "/api/v1/insights"))
			protected.Any("/chat/*path", proxyToService(cfg.Services.Sophia, "/api/v1/chat"))
			protected.Any("/categorize/*path", proxyToService(cfg.Services.Sophia, "/api/v1/categorize"))

			// Logos (Document Processing) service routes
			protected.Any("/documents/*path", proxyToService(cfg.Services.Logos, "/api/v1/documents"))
			protected.Any("/upload/*path", proxyToService(cfg.Services.Logos, "/api/v1/upload"))
			protected.Any("/processing/*path", proxyToService(cfg.Services.Logos, "/api/v1/processing"))
		}
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "hermes-gateway",
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

// proxyToService creates a reverse proxy to a backend service with path rewriting
func proxyToService(serviceURL, targetPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		remote, err := url.Parse(serviceURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid service URL"})
			return
		}

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(remote)
		
		// Custom director to handle path rewriting and auth forwarding
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			
			// Rewrite the path to target service path
			originalPath := req.URL.Path
			pathSuffix := strings.TrimPrefix(originalPath, strings.Split(targetPath, "*")[0])
			req.URL.Path = targetPath + pathSuffix
			
			// Forward authentication headers and user context
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
			req.Host = remote.Host
		}

		// Handle errors
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			io.WriteString(w, `{"error":"Backend service unavailable","details":"`+err.Error()+`"}`)
		}

		// Serve the request
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}