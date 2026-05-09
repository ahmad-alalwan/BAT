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

	"github.com/your-project/internal/application/auth"
	"github.com/your-project/internal/domain/auth"
	"github.com/your-project/internal/infrastructure/api/client"
	"github.com/your-project/internal/infrastructure/api/gin"
	"github.com/your-project/internal/infrastructure/cache"
)

// MockRegistryURLResolver implements RegistryURLResolver interface
type MockRegistryURLResolver struct{}

func (m *MockRegistryURLResolver) Resolve(tenantID auth.TenantID) (string, error) {
	// Hardcode to mock tenant URL
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
	jaegerEndpoint := os.Getenv("OTEL_EXPORTER_JAEGER_ENDPOINT")
	if jaegerEndpoint == "" {
		jaegerEndpoint = "http://jaeger:14268/api/traces"
	}
	
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
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
	log.Println("🚀 Starting BAT API Gateway...")

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

	// Initialize Redis connection
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

	// Initialize infrastructure implementations
	registryResolver := &MockRegistryURLResolver{}
	tenantClient := client.NewTenantInternalClient(registryResolver)
	otpStore := cache.NewRedisOTPStore(rdb)
	userVerifier := &MockUserVerifier{tenantClient: tenantClient}

	// Initialize domain services
	authDomainService := auth.NewAuthDomainService(nil, otpStore, userVerifier)

	// Initialize application use cases
	authUseCase := auth.NewAuthUseCase(authDomainService)

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

	// Initialize HTTP handlers
	authHandler := gin.NewAuthHandler(authUseCase, otpStore)

	// Setup routes
	router.GET("/health", authHandler.HealthHandler)

	// API routes
	api := router.Group("/api")
	{
		bat := api.Group("/bat")
		{
			bat.POST("/login", authHandler.LoginHandler)
		}
	}

	// Auth callback route (for redirect handling)
	router.GET("/auth/callback", authHandler.ExchangeHandler)

	// Static files
	router.Static("/static", "./internal/infrastructure/api/gin/static")
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/static/index.html")
	})

	// Setup server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Println("🌐 BAT API Gateway starting on port 8080")
		log.Println("📊 Dashboard available at: http://localhost:8080")
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
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("❌ Server forced to shutdown:", err)
	}

	log.Println("✅ Server exited")
}
