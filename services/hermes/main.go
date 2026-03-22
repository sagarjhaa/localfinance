package main

import (
	"log" 
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	router := gin.Default()

	// Health endpoint with correlation ID
	router.GET("/health", func(c *gin.Context) {
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = "lf_" + time.Now().Format("20060102150405")
		}
		
		c.Header("X-Correlation-ID", correlationID)
		c.Header("X-Service-Name", "hermes")
		c.Header("Access-Control-Expose-Headers", "X-Correlation-ID")
		
		c.JSON(http.StatusOK, gin.H{
			"status":         "healthy",
			"service":        "hermes-gateway", 
			"version":        "2.0.0",
			"correlation_id": correlationID,
			"features":       []string{"correlation_id", "microservices", "health_check"},
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Web interface root page
	router.GET("/", func(c *gin.Context) {
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = "lf_" + time.Now().Format("20060102150405")
		}
		c.Header("X-Correlation-ID", correlationID)
		
		html := `<!DOCTYPE html>
<html>
<head>
    <title>LocalFinance Microservices</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .service { background: #f8f9fa; padding: 15px; margin: 10px 0; border-radius: 5px; border-left: 4px solid #007bff; }
        .healthy { border-left-color: #28a745; }
        .correlation { background: #e3f2fd; padding: 10px; border-radius: 5px; font-family: monospace; margin: 20px 0; }
        button { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 5px; cursor: pointer; margin: 5px; }
        button:hover { background: #0056b3; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 LocalFinance Microservices</h1>
        <div class="correlation">
            <strong>Request ID:</strong> ` + correlationID + `
            <button onclick="navigator.clipboard.writeText('` + correlationID + `')">📋 Copy</button>
        </div>
        
        <h2>🏗️ Available Services:</h2>
        
        <div class="service healthy">
            <h3>🎭 Hermes Gateway</h3>
            <p>API Gateway and Web Interface</p>
            <p><strong>Port:</strong> 3000 | <strong>Status:</strong> ✅ Healthy</p>
            <button onclick="testService('/health')">Test Health</button>
        </div>
        
        <div class="service">
            <h3>🏛️ Thesaurus Database</h3>
            <p>Data storage and CRUD operations</p>
            <p><strong>Port:</strong> 8001</p>
            <button onclick="testService('http://10.0.0.16:8001/health')">Test Service</button>
        </div>
        
        <div class="service">
            <h3>🦉 Sophia AI</h3>
            <p>AI processing and financial insights</p>
            <p><strong>Port:</strong> 8002</p>
            <button onclick="testService('http://10.0.0.16:8002/health')">Test Service</button>
        </div>
        
        <div class="service">
            <h3>📜 Logos Processing</h3>
            <p>Document and transaction processing</p>
            <p><strong>Port:</strong> 8003</p>
            <button onclick="testService('http://10.0.0.16:8003/health')">Test Service</button>
        </div>
        
        <div id="results"></div>
        
        <h2>🔗 Correlation ID Features:</h2>
        <ul>
            <li>✅ Request tracing across all services</li>
            <li>✅ Headers: X-Correlation-ID</li>
            <li>✅ Response body includes correlation_id</li>
            <li>✅ CORS headers for frontend access</li>
        </ul>
    </div>

    <script>
        async function testService(url) {
            const resultsDiv = document.getElementById('results');
            const correlationId = 'web_test_' + Date.now();
            
            try {
                const response = await fetch(url, {
                    headers: { 'X-Correlation-ID': correlationId }
                });
                
                const data = await response.json();
                const responseCorrelationId = response.headers.get('X-Correlation-ID');
                
                resultsDiv.innerHTML = '<h3>Test Result:</h3><pre>' + 
                    JSON.stringify(data, null, 2) + 
                    '</pre><p><strong>Correlation ID:</strong> ' + 
                    responseCorrelationId + '</p>';
            } catch (error) {
                resultsDiv.innerHTML = '<h3>Error:</h3><p>' + error.message + '</p>';
            }
        }
    </script>
</body>
</html>`
		
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	log.Printf("🎭 Hermes Gateway starting on port %s", port)
	log.Printf("🌐 Web interface: http://localhost:%s", port)
	log.Printf("🔗 Health endpoint: http://localhost:%s/health", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Hermes: %v", err)
	}
}
