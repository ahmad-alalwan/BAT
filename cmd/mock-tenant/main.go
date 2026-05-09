package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VerifyRequest represents the request for user verification
type VerifyRequest struct {
	TenantID string `json:"tenant_id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// VerifyResponse represents the response for user verification
type VerifyResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

// ExchangeRequest represents the OTP exchange request
type ExchangeRequest struct {
	Code string `json:"code"`
}

// ExchangeResponse represents the response from BAT API exchange
type ExchangeResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user"`
}

func main() {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Tenant-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// POST /internal/verify - User verification endpoint
	router.POST("/internal/verify", func(c *gin.Context) {
		var req VerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		log.Printf("Mock tenant received verification request: %+v", req)

		// Hardcoded credentials check
		if req.Username == "admin" && req.Password == "123456" {
			response := VerifyResponse{
				ID:       "uuid-123",
				Username: req.Username,
				Role:     "admin",
				IsActive: true,
			}
			c.JSON(http.StatusOK, response)
			log.Printf("Verification successful for user: %s", req.Username)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			log.Printf("Verification failed for user: %s", req.Username)
		}
	})

	// POST /auth/exchange - OTP exchange endpoint
	router.POST("/auth/exchange", func(c *gin.Context) {
		var req ExchangeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		log.Printf("Mock tenant received OTP exchange request with code: %s", req.Code)

		// Make HTTP POST to bat-api:8080/internal/exchange
		exchangePayload := map[string]string{
			"code": req.Code,
		}

		jsonData, err := json.Marshal(exchangePayload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal exchange payload"})
			return
		}

		// Create HTTP request with timeout
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		reqURL := "http://bat-api:8080/internal/exchange"
		httpReq, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exchange request"})
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Mock-Tenant", "true")

		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("Failed to call BAT API exchange: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange OTP"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("BAT API exchange returned status: %d", resp.StatusCode)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Exchange failed"})
			return
		}

		var exchangeResp ExchangeResponse
		if err := json.NewDecoder(resp.Body).Decode(&exchangeResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode exchange response"})
			return
		}

		log.Printf("Exchange successful, token: %s", exchangeResp.Token)

		// Return HTML page showing login success
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Login Successful</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 50px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .success { color: #28a745; font-size: 24px; margin-bottom: 20px; }
        .token { background: #f8f9fa; padding: 15px; border-radius: 4px; word-break: break-all; border: 1px solid #dee2e6; }
        .user-info { margin-top: 20px; }
        .timestamp { color: #6c757d; font-size: 14px; margin-top: 30px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="success">✅ Login Successful!</div>
        <div class="user-info">
            <strong>User ID:</strong> %s<br>
            <strong>Username:</strong> %s<br>
            <strong>Role:</strong> %s
        </div>
        <div style="margin-top: 20px;">
            <strong>JWT Token:</strong>
            <div class="token">%s</div>
        </div>
        <div class="timestamp">
            Generated at: %s
        </div>
    </div>
</body>
</html>`,
			exchangeResp.User.ID,
			exchangeResp.User.Username,
			exchangeResp.User.Role,
			exchangeResp.Token,
			time.Now().Format("2006-01-02 15:04:05"),
		)

		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"service":   "mock-tenant",
			"timestamp": time.Now().Unix(),
		})
	})

	// Static dashboard endpoint
	router.GET("/dashboard", func(c *gin.Context) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Mock Tenant Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f8f9fa; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .test-section { margin: 20px 0; padding: 20px; border: 1px solid #dee2e6; border-radius: 4px; }
        .form-group { margin: 10px 0; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input, button { padding: 8px 12px; margin: 5px 0; }
        button { background-color: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background-color: #0056b3; }
        .result { margin-top: 15px; padding: 10px; border-radius: 4px; }
        .success { background-color: #d4edda; color: #155724; }
        .error { background-color: #f8d7da; color: #721c24; }
        .credentials { background-color: #fff3cd; padding: 15px; border-radius: 4px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🏢 Mock Tenant Dashboard</h1>
            <p>Testing interface for BoundedAuthT (BAT) authentication flow</p>
        </div>
        
        <div class="credentials">
            <strong>🔑 Test Credentials:</strong><br>
            Username: <code>admin</code><br>
            Password: <code>123456</code>
        </div>

        <div class="test-section">
            <h3>🔐 User Verification Test</h3>
            <div class="form-group">
                <label for="username">Username:</label>
                <input type="text" id="username" value="admin">
            </div>
            <div class="form-group">
                <label for="password">Password:</label>
                <input type="password" id="password" value="123456">
            </div>
            <button onclick="testVerification()">Test Verification</button>
            <div id="verify-result"></div>
        </div>

        <div class="test-section">
            <h3>🔄 OTP Exchange Test</h3>
            <div class="form-group">
                <label for="otp-code">OTP Code:</label>
                <input type="text" id="otp-code" placeholder="Enter OTP code">
            </div>
            <button onclick="testExchange()">Test Exchange</button>
            <div id="exchange-result"></div>
        </div>
    </div>

    <script>
        async function testVerification() {
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const resultDiv = document.getElementById('verify-result');
            
            try {
                const response = await fetch('/internal/verify', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        tenant_id: 'test-tenant',
                        username: username,
                        password: password
                    })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    resultDiv.innerHTML = '<div class="result success">✅ Verification Successful! User ID: ' + data.id + ', Role: ' + data.role + '</div>';
                } else {
                    resultDiv.innerHTML = '<div class="result error">❌ Verification Failed: ' + (data.error || 'Unknown error') + '</div>';
                }
            } catch (error) {
                resultDiv.innerHTML = '<div class="result error">❌ Network Error: ' + error.message + '</div>';
            }
        }
        
        async function testExchange() {
            const code = document.getElementById('otp-code').value;
            const resultDiv = document.getElementById('exchange-result');
            
            if (!code) {
                resultDiv.innerHTML = '<div class="result error">❌ Please enter an OTP code</div>';
                return;
            }
            
            try {
                const response = await fetch('/auth/exchange', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        code: code
                    })
                });
                
                if (response.ok) {
                    // This will return HTML, so we'll open it in a new window
                    const blob = await response.blob();
                    const url = window.URL.createObjectURL(blob);
                    window.open(url, '_blank');
                    resultDiv.innerHTML = '<div class="result success">✅ Exchange Successful! Check new window for results.</div>';
                } else {
                    const errorData = await response.json();
                    resultDiv.innerHTML = '<div class="result error">❌ Exchange Failed: ' + (errorData.error || 'Unknown error') + '</div>';
                }
            } catch (error) {
                resultDiv.innerHTML = '<div class="result error">❌ Network Error: ' + error.message + '</div>';
            }
        }
    </script>
</body>
</html>`
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	})

	// Start server
	port := ":8081"
	log.Printf("Mock Tenant server starting on port %s", port)
	log.Printf("Dashboard available at: http://localhost%s/dashboard", port)
	log.Printf("Health check at: http://localhost%s/health", port)
	
	if err := router.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
