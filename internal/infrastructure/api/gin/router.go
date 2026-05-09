package gin

import (
	"github.com/gin-gonic/gin"
	
	"github.com/your-project/internal/application/auth"
)

// RouterConfig holds the configuration for the Gin router
type RouterConfig struct {
	AuthHandler *AuthHandler
}

// NewRouter creates and configures a new Gin router with all routes and middleware
func NewRouter(config RouterConfig) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	
	// Create Gin engine
	router := gin.New()
	
	// Add global middleware
	router.Use(RecoveryMiddleware())
	router.Use(RequestLoggerMiddleware())
	router.Use(CORSMiddleware())
	
	// Add tenant middleware for all API routes
	api := router.Group("/api")
	api.Use(TenantMiddleware())
	
	// Setup authentication routes
	setupAuthRoutes(api, config.AuthHandler)
	
	// Health check endpoint (doesn't require tenant middleware)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"service": "auth-gateway",
		})
	})
	
	return router
}

// setupAuthRoutes configures all authentication-related routes
func setupAuthRoutes(api *gin.RouterGroup, authHandler *AuthHandler) {
	authGroup := api.Group("/auth")
	
	// Login with email and password
	authGroup.POST("/login", authHandler.Login)
	
	// OTP-related routes
	otpGroup := authGroup.Group("/otp")
	{
		// Generate OTP for a user
		otpGroup.POST("/generate", authHandler.GenerateOTP)
		
		// Validate OTP (without authentication)
		otpGroup.POST("/validate", authHandler.ValidateOTP)
		
		// Login with OTP
		otpGroup.POST("/login", authHandler.LoginWithOTP)
	}
}

// SetupRoutes is an alternative method that can be used if you prefer a different approach
func SetupRoutes(router *gin.Engine, authUseCase *auth.AuthUseCase) {
	// Create handler
	authHandler := NewAuthHandler(authUseCase)
	
	// Setup middleware
	router.Use(RecoveryMiddleware())
	router.Use(RequestLoggerMiddleware())
	router.Use(CORSMiddleware())
	
	// API routes with tenant middleware
	api := router.Group("/api")
	api.Use(TenantMiddleware())
	
	// Authentication routes
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		
		// OTP routes
		otpGroup := authGroup.Group("/otp")
		{
			otpGroup.POST("/generate", authHandler.GenerateOTP)
			otpGroup.POST("/validate", authHandler.ValidateOTP)
			otpGroup.POST("/login", authHandler.LoginWithOTP)
		}
	}
	
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"service": "auth-gateway",
		})
	})
}
