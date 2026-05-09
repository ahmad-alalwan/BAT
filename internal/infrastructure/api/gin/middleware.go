package gin

import (
	"fmt"
	"net/http"
	"strings"
	
	"github.com/gin-gonic/gin"
	
	"github.com/your-project/internal/domain/auth"
	"github.com/your-project/pkg/contextx"
)

// TenantContextMiddleware extracts the subdomain from c.Request.Host, validates it, and injects it into context
// Expected format: "tenant-a.app.com" -> "tenant-a"
// If invalid, aborts with 404
func TenantContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract tenant ID from subdomain
		host := c.Request.Host
		tenantID := extractTenantFromSubdomain(host)
		
		// If no tenant ID found in subdomain, abort with 404
		if tenantID == "" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "tenant not found",
			})
			c.Abort()
			return
		}
		
		// Validate tenant ID format
		validTenantID, err := auth.NewTenantID(tenantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "invalid tenant format",
			})
			c.Abort()
			return
		}
		
		// Inject tenant ID into context using contextx package
		ctx := contextx.SetTenantID(c.Request.Context(), *validTenantID)
		c.Request = c.Request.WithContext(ctx)
		
		c.Next()
	}
}

// extractTenantFromSubdomain extracts tenant ID from subdomain
// Example: tenant1.yourdomain.com -> tenant1
func extractTenantFromSubdomain(host string) string {
	// Remove port if present
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}
	
	// Split by dots
	parts := strings.Split(host, ".")
	
	// If we have at least 2 parts (subdomain.domain), the first part is the tenant
	if len(parts) >= 2 && parts[0] != "www" && parts[0] != "api" {
		return parts[0]
	}
	
	return ""
}

// extractTenantFromPath extracts tenant ID from URL path
// Example: /api/tenant1/login -> tenant1
func extractTenantFromPath(path string) string {
	// Split path by slashes
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	// Look for tenant pattern in path segments
	for _, part := range parts {
		// Simple validation: check if it looks like a tenant ID (contains letters and hyphens)
		if strings.Contains(part, "-") || isAlphaNumeric(part) {
			// Additional validation can be added here
			return part
		}
	}
	
	return ""
}

// isAlphaNumeric checks if a string contains only letters and numbers
func isAlphaNumeric(s string) bool {
	for _, char := range s {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}
	return len(s) > 0
}

// CORSMiddleware adds CORS headers to the response
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Tenant-ID")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// RequestLoggerMiddleware logs HTTP requests
func RequestLoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Extract tenant ID from context if available
		tenantID := "unknown"
		if ctxTenantID, exists := contextx.GetTenantID(param.Request.Context()); exists {
			tenantID = ctxTenantID.String()
		}
		
		return fmt.Sprintf("[%s] %s %s %d %s %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			tenantID,
		)
	})
}

// RecoveryMiddleware recovers from panics and returns a proper error response
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Extract tenant ID for logging
		tenantID := "unknown"
		if ctxTenantID, exists := contextx.GetTenantID(c.Request.Context()); exists {
			tenantID = ctxTenantID.String()
		}
		
		// Log the panic (in production, use proper logging)
		fmt.Printf("Panic recovered for tenant %s: %v\n", tenantID, recovered)
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
	})
}
