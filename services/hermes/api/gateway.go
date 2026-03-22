package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/services/hermes/config"
	"github.com/sagarjhaa/localfinance/shared/middleware"
)

// SetupRoutes configures the API gateway and frontend routes with correlation ID propagation
func SetupRoutes(router *gin.Engine, cfg *config.Config) {
	// Health check endpoint for Hermes gateway
	router.GET("/health", func(c *gin.Context) {
		correlationID := middleware.GetCorrelationID(c)
		c.JSON(http.StatusOK, gin.H{
			"status":         "healthy",
			"service":        "hermes-gateway",
			"version":        "2.0.0",
			"correlation_id": correlationID,
			"features":       []string{"correlation_id", "microservices", "proxy_gateway"},
			"upstream_services": map[string]string{
				"thesaurus": cfg.Services.Thesaurus,
				"sophia":    cfg.Services.Sophia,
				"logos":     cfg.Services.Logos,
			},
		})
	})

	// Serve static frontend files
	router.Static("/static", "./frontend/build/static")
	router.StaticFile("/favicon.ico", "./frontend/build/favicon.ico")
	
	// API Gateway routes with correlation ID propagation
	api := router.Group("/api")
	{
		// Thesaurus (CRUD/Database) service routes
		api.Any("/v1/transactions/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))
		api.Any("/v1/users/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))
		api.Any("/v1/accounts/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))
		api.Any("/v1/budgets/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))
		api.Any("/v1/upload/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))
		api.Any("/v1/documents/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))
		api.Any("/v1/settings/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))
		api.Any("/v1/dashboard/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))
		api.Any("/v1/processing-status/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))

		// Authentication routes (handled by Thesaurus)
		api.Any("/auth/*path", createProxyWithCorrelation(cfg.Services.Thesaurus, "thesaurus"))

		// Sophia (AI) service routes
		api.Any("/ai/*path", createProxyWithCorrelation(cfg.Services.Sophia, "sophia"))
		api.Any("/chat/*path", createProxyWithCorrelation(cfg.Services.Sophia, "sophia"))

		// Logos (Document Processing) service routes
		api.Any("/process/*path", createProxyWithCorrelation(cfg.Services.Logos, "logos"))
	}

	// Default route serves React frontend
	router.NoRoute(func(c *gin.Context) {
		// Add correlation ID to frontend response
		correlationID := middleware.GetCorrelationID(c)
		c.Header("X-Correlation-ID", correlationID)
		
		c.File("./frontend/build/index.html")
	})
}

// createProxyWithCorrelation creates a reverse proxy that propagates correlation IDs
func createProxyWithCorrelation(targetURL, targetService string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic(fmt.Sprintf("Invalid target URL for %s: %v", targetService, err))
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Custom director to add correlation ID and service headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// Get correlation ID from gin context
		correlationID := req.Header.Get(middleware.CorrelationIDHeader)
		
		// Call original director
		originalDirector(req)
		
		// Add correlation and service headers
		req.Header.Set(middleware.CorrelationIDHeader, correlationID)
		req.Header.Set("X-Forwarded-By", "hermes")
		req.Header.Set("X-Target-Service", targetService)
		
		// Preserve original host and add proxy headers
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	}

	// Custom error handler
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		correlationID := req.Header.Get(middleware.CorrelationIDHeader)
		
		rw.Header().Set("X-Correlation-ID", correlationID)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusBadGateway)
		
		response := fmt.Sprintf(`{
			"error": "Service unavailable",
			"service": "%s",
			"correlation_id": "%s",
			"details": "%v"
		}`, targetService, correlationID, err)
		
		rw.Write([]byte(response))
	}

	// Custom response modifier to ensure correlation ID is preserved
	proxy.ModifyResponse = func(resp *http.Response) error {
		correlationID := resp.Header.Get(middleware.CorrelationIDHeader)
		if correlationID != "" {
			resp.Header.Set(middleware.CorrelationIDHeader, correlationID)
		}
		return nil
	}

	return gin.WrapH(proxy)
}

// loggingRoundTripper wraps http.RoundTripper to log outgoing requests
type loggingRoundTripper struct {
	next          http.RoundTripper
	correlationID string
	targetService string
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log outgoing request
	logOutgoingRequest(req, l.correlationID, l.targetService)
	
	// Make request
	resp, err := l.next.RoundTrip(req)
	
	// Log response
	if err == nil {
		logIncomingResponse(resp, l.correlationID, l.targetService)
	}
	
	return resp, err
}

// logOutgoingRequest logs requests going to downstream services
func logOutgoingRequest(req *http.Request, correlationID, targetService string) {
	var body interface{}
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		
		if len(bodyBytes) > 0 {
			body = string(bodyBytes)
			if len(bodyBytes) > 500 {
				body = string(bodyBytes[:500]) + "..."
			}
		}
	}

	fmt.Printf(`[OUTGOING] {"correlation_id":"%s","service":"hermes","target":"%s","method":"%s","url":"%s","body":%q}`,
		correlationID, targetService, req.Method, req.URL.String(), body)
}

// logIncomingResponse logs responses from downstream services  
func logIncomingResponse(resp *http.Response, correlationID, targetService string) {
	var body interface{}
	if resp.Body != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		
		if len(bodyBytes) > 0 {
			body = string(bodyBytes)
			if len(bodyBytes) > 500 {
				body = string(bodyBytes[:500]) + "..."
			}
		}
	}

	fmt.Printf(`[INCOMING] {"correlation_id":"%s","service":"hermes","from":"%s","status":%d,"body":%q}`,
		correlationID, targetService, resp.StatusCode, body)
}