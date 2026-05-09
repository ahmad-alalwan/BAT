package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/your-project/internal/application/auth"
	"github.com/your-project/internal/domain/auth"
	"github.com/your-project/internal/infrastructure/api/gin"
	"github.com/your-project/internal/infrastructure/api/client"
	"github.com/your-project/internal/infrastructure/cache"
)

// MockRegistryURLResolver implements RegistryURLResolver for testing
type MockRegistryURLResolver struct{}

func (m *MockRegistryURLResolver) Resolve(tenantID auth.TenantID) (string, error) {
	// For testing, always resolve to the mock tenant service
	return "http://mock-tenant:8081", nil
}

// MockUserVerifier implements UserVerifier interface
type MockUserVerifier struct {
	tenantClient *client.TenantInternalClient
}

func (m *MockUserVerifier) VerifyCredentials(ctx context.Context, tenantID auth.TenantID, username string, password auth.PlainTextPassword) (*auth.User, error) {
	return m.tenantClient.Verify(ctx, tenantID, username, password)
}

func initTracing() (*sdktrace.TracerProvider, error) {
	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://jaeger:14268/api/traces")))
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("bat-api"),
			semconv.ServiceVersionKey.String("1.0.0"),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	log.Println("🚀 Starting BoundedAuthT (BAT) API Gateway...")

	// Initialize tracing
	tp, err := initTracing()
	if err != nil {
		log.Printf("⚠️  Failed to initialize tracing: %v", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("⚠️  Error shutting down tracer provider: %v", err)
			}
		}()
	}

	// Initialize Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password
		DB:       0,  // default DB
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Connected to Redis")

	// Initialize infrastructure components
	registryResolver := &MockRegistryURLResolver{}
	tenantClient := client.NewTenantInternalClient(registryResolver)
	otpStore := cache.NewRedisOTPStore(rdb)
	userVerifier := &MockUserVerifier{tenantClient: tenantClient}

	// Initialize domain services
	authDomainService := auth.NewAuthDomainService(nil, otpStore, userVerifier)

	// Initialize application use cases
	authUseCase := auth.NewAuthUseCase(authDomainService)

	// Initialize HTTP handlers
	authHandler := gin.NewAuthHandler(authUseCase)

	// Setup Gin router
	routerConfig := gin.RouterConfig{
		AuthHandler: authHandler,
	}
	router := gin.NewRouter(routerConfig)

	// Add internal exchange endpoint for OTP exchange
	router.POST("/internal/exchange", func(c *gin.Context) {
		var req struct {
			Code string `json:"code" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		// Create a tracer
		tracer := otel.Tracer("bat-api")
		spanCtx, span := tracer.Start(c.Request.Context(), "otp-exchange")
		defer span.End()

		log.Printf("🔄 Processing OTP exchange for code: %s", req.Code)

		// Retrieve and delete OTP from Redis
		otpPayload, err := otpStore.GetAndDeleteOTP(spanCtx, req.Code)
		if err != nil {
			log.Printf("❌ Failed to retrieve OTP: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
			return
		}

		if otpPayload == nil {
			log.Printf("❌ OTP not found: %s", req.Code)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
			return
		}

		// Generate JWT token (simplified)
		token := fmt.Sprintf("jwt-token-%s-%d", otpPayload.UserID, time.Now().Unix())

		// Create user entity (mock data)
		tenantID, err := auth.NewTenantID(otpPayload.TenantID)
		if err != nil {
			log.Printf("❌ Invalid tenant ID in OTP payload: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		user, err := auth.NewUser(otpPayload.UserID, *tenantID, "user", "user")
		if err != nil {
			log.Printf("❌ Failed to create user entity: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		response := gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID(),
				"username": user.Username(),
				"role":     user.Role(),
			},
		}

		log.Printf("✅ OTP exchange successful for user: %s", user.ID())
		c.JSON(http.StatusOK, response)
	})

	// Add dashboard endpoint
	router.GET("/dashboard", func(c *gin.Context) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>🔐 BAT API Gateway Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f8f9fa; }
        .container { max-width: 1000px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .flow-diagram { background: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .step { margin: 15px 0; padding: 15px; border-left: 4px solid #007bff; background: white; }
        .test-section { margin: 20px 0; padding: 20px; border: 1px solid #dee2e6; border-radius: 4px; }
        .form-group { margin: 10px 0; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input, button { padding: 8px 12px; margin: 5px 0; }
        button { background-color: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background-color: #0056b3; }
        .result { margin-top: 15px; padding: 10px; border-radius: 4px; }
        .success { background-color: #d4edda; color: #155724; }
        .error { background-color: #f8d7da; color: #721c24; }
        .info { background-color: #d1ecf1; color: #0c5460; }
        .arrow { font-size: 20px; color: #007bff; }
        .status { background: #e9ecef; padding: 10px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 BAT API Gateway Dashboard</h1>
            <p>BoundedAuthT - Zero-Knowledge Multi-Tenant Auth Gateway</p>
            <div class="status">
                <strong>Services:</strong> bat-api (8080) | mock-tenant (8081) | redis (6379) | apisix (9080)
            </div>
        </div>

        <div class="flow-diagram">
            <h3>🔄 Authentication Flow</h3>
            <div class="step">
                <strong>1. User Login</strong><br>
                User enters credentials → Tenant subdomain → BAT API
            </div>
            <div class="arrow">⬇️</div>
            <div class="step">
                <strong>2. Credential Verification</strong><br>
                BAT API calls Tenant Internal API → Validates username/password
            </div>
            <div class="arrow">⬇️</div>
            <div class="step">
                <strong>3. OTP Generation</strong><br>
                If valid → Generate OTP → Store in Redis → Return to user
            </div>
            <div class="arrow">⬇️</div>
            <div class="step">
                <strong>4. OTP Exchange</strong><br>
                User submits OTP → BAT API validates → Returns JWT token
            </div>
        </div>

        <div class="test-section">
            <h3>🧪 Test Authentication Flow</h3>
            <div class="form-group">
                <label for="tenant-id">Tenant ID (subdomain):</label>
                <input type="text" id="tenant-id" value="test-tenant" placeholder="e.g., company-a">
            </div>
            <div class="form-group">
                <label for="username">Username:</label>
                <input type="text" id="username" value="admin" placeholder="admin">
            </div>
            <div class="form-group">
                <label for="password">Password:</label>
                <input type="password" id="password" value="123456" placeholder="123456">
            </div>
            <button onclick="testLogin()">🔐 Test Login</button>
            <button onclick="testOTPGeneration()">📱 Generate OTP</button>
            <button onclick="testOTPExchange()">🔄 Exchange OTP</button>
            <div id="test-result"></div>
        </div>

        <div class="test-section">
            <h3>📊 Service Status</h3>
            <button onclick="checkServices()">🔍 Check Services</button>
            <div id="service-status"></div>
        </div>
    </div>

    <script>
        let currentOTP = '';
        let currentToken = '';

        async function testLogin() {
            const tenantId = document.getElementById('tenant-id').value;
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const resultDiv = document.getElementById('test-result');
            
            try {
                const response = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Host': tenantId + '.localhost:8080'
                    },
                    body: JSON.stringify({
                        email: username,
                        password: password
                    })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    resultDiv.innerHTML = '<div class="result success">✅ Login Successful! User ID: ' + data.user_id + ', Token: ' + data.token + '</div>';
                    currentToken = data.token;
                } else {
                    resultDiv.innerHTML = '<div class="result error">❌ Login Failed: ' + (data.error || 'Unknown error') + '</div>';
                }
            } catch (error) {
                resultDiv.innerHTML = '<div class="result error">❌ Network Error: ' + error.message + '</div>';
            }
        }
        
        async function testOTPGeneration() {
            const tenantId = document.getElementById('tenant-id').value;
            const username = document.getElementById('username').value;
            const resultDiv = document.getElementById('test-result');
            
            try {
                // First login to get user ID (simplified for demo)
                const loginResponse = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Host': tenantId + '.localhost:8080'
                    },
                    body: JSON.stringify({
                        email: username,
                        password: '123456'
                    })
                });
                
                if (!loginResponse.ok) {
                    resultDiv.innerHTML = '<div class="result error">❌ Please login first</div>';
                    return;
                }
                
                const loginData = await loginResponse.json();
                
                // Generate OTP
                const response = await fetch('/api/auth/otp/generate', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Host': tenantId + '.localhost:8080'
                    },
                    body: JSON.stringify({
                        user_id: loginData.user_id
                    })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    resultDiv.innerHTML = '<div class="result success">📱 OTP Generated: ' + data.otp_code + '</div>';
                    currentOTP = data.otp_code;
                } else {
                    resultDiv.innerHTML = '<div class="result error">❌ OTP Generation Failed: ' + (data.error || 'Unknown error') + '</div>';
                }
            } catch (error) {
                resultDiv.innerHTML = '<div class="result error">❌ Network Error: ' + error.message + '</div>';
            }
        }
        
        async function testOTPExchange() {
            if (!currentOTP) {
                const resultDiv = document.getElementById('test-result');
                resultDiv.innerHTML = '<div class="result error">❌ Please generate OTP first</div>';
                return;
            }
            
            const resultDiv = document.getElementById('test-result');
            
            try {
                const response = await fetch('/internal/exchange', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        code: currentOTP
                    })
                });
                
                const data = await response.json();
                
                if (response.ok) {
                    resultDiv.innerHTML = '<div class="result success">✅ OTP Exchange Successful! Token: ' + data.token + '</div>';
                    currentToken = data.token;
                } else {
                    resultDiv.innerHTML = '<div class="result error">❌ OTP Exchange Failed: ' + (data.error || 'Unknown error') + '</div>';
                }
            } catch (error) {
                resultDiv.innerHTML = '<div class="result error">❌ Network Error: ' + error.message + '</div>';
            }
        }
        
        async function checkServices() {
            const statusDiv = document.getElementById('service-status');
            const services = [
                { name: 'BAT API', url: '/health' },
                { name: 'Mock Tenant', url: 'http://mock-tenant:8081/health' },
                { name: 'Redis', url: 'N/A (internal)' }
            ];
            
            let statusHTML = '<h4>Service Status:</h4>';
            
            for (const service of services) {
                if (service.url === 'N/A (internal)') {
                    statusHTML += '<div class="info">✅ ' + service.name + ': ' + service.url + '</div>';
                } else {
                    try {
                        const response = await fetch(service.url);
                        const status = response.ok ? '✅' : '❌';
                        statusHTML += '<div class="' + (response.ok ? 'success' : 'error') + '">' + status + ' ' + service.name + ': ' + (response.ok ? 'OK' : 'Error') + '</div>';
                    } catch (error) {
                        statusHTML += '<div class="error">❌ ' + service.name + ': Unreachable</div>';
                    }
                }
            }
            
            statusDiv.innerHTML = statusHTML;
        }
        
        // Auto-check services on load
        window.onload = function() {
            checkServices();
        };
    </script>
</body>
</html>`
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"service":   "bat-api",
			"version":   "1.0.0",
			"timestamp": time.Now().Unix(),
		})
	})

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Println("🌐 BAT API Gateway starting on port 8080")
		log.Println("📊 Dashboard available at: http://localhost:8080/dashboard")
		log.Println("🏥 Health check at: http://localhost:8080/health")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🔄 Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("❌ Server forced to shutdown:", err)
	}

	log.Println("✅ Server exited")
}
