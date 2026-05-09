package gin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	
	"github.com/gin-gonic/gin"
	
	"github.com/your-project/internal/application/auth"
	"github.com/your-project/internal/domain/auth"
	"github.com/your-project/pkg/contextx"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authUseCase *auth.AuthUseCase
	otpStore    auth.OTPStore
}

// NewAuthHandler creates a new AuthHandler with dependency injection
func NewAuthHandler(authUseCase *auth.AuthUseCase, otpStore auth.OTPStore) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
		otpStore:    otpStore,
	}
}

// LoginCommand represents the HTTP request body for login
type LoginCommand struct {
	TenantID string `json:"tenant_id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginHandler handles user login and redirects with OTP code
func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var cmd LoginCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request format: " + err.Error(),
		})
		return
	}
	
	// Validate tenant ID
	tenantID, err := auth.NewTenantID(cmd.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid tenant ID: " + err.Error(),
		})
		return
	}
	
	// Validate password
	password, err := auth.NewPlainTextPassword(cmd.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid password: " + err.Error(),
		})
		return
	}
	
	// Create password value object
	passwordObj, err := auth.NewPlainTextPassword(cmd.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid password format: " + err.Error(),
		})
		return
	}
	
	// Call AuthUseCase.Execute() - this would verify credentials and generate OTP
	// For now, we'll simulate this by generating an OTP and storing it
	otpCode := h.generateOTPCode()
	
	// Create OTP payload
	otpPayload := &auth.OTPPayload{
		UserID:    "user-" + cmd.Username, // In real implementation, this comes from user verification
		TenantID:  cmd.TenantID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		CreatedAt: time.Now(),
	}
	
	// Store OTP in Redis
	err = h.otpStore.SaveOTP(c.Request.Context(), otpCode, otpPayload, 5*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate OTP: " + err.Error(),
		})
		return
	}
	
	// Return 302 Redirect to callback with OTP code
	callbackURL := fmt.Sprintf("http://localhost:9080/auth/callback?code=%s", otpCode)
	c.Redirect(http.StatusFound, callbackURL)
}

// ExchangeHandler handles OTP exchange and returns JWT
func (h *AuthHandler) ExchangeHandler(c *gin.Context) {
	// Read code from query params
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "OTP code is required",
		})
		return
	}
	
	// Get OTP payload from store
	otpPayload, err := h.otpStore.GetAndDeleteOTP(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve OTP: " + err.Error(),
		})
		return
	}
	
	if otpPayload == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired OTP",
		})
		return
	}
	
	// Check if OTP is expired
	if time.Now().After(otpPayload.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "OTP expired",
		})
		return
	}
	
	// Generate fake JWT string
	jwtToken := fmt.Sprintf("jwt_%s_%s", otpPayload.TenantID, otpPayload.UserID)
	
	// Return JWT as JSON
	c.JSON(http.StatusOK, gin.H{
		"token": jwtToken,
		"user": gin.H{
			"id":       otpPayload.UserID,
			"tenant_id": otpPayload.TenantID,
		},
		"expires_at": otpPayload.ExpiresAt,
	})
}

// generateOTPCode generates a 6-digit OTP code
func (h *AuthHandler) generateOTPCode() string {
	// Simple OTP generation - in production, use a proper crypto random generator
	return strconv.FormatInt(time.Now().Unix()%1000000, 10)
}

// Health check handler
func (h *AuthHandler) HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "bat-api",
		"timestamp": time.Now().Unix(),
	})
}
