package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	
	"github.com/your-project/internal/domain/auth"
)

// RegistryURLResolver resolves tenant ID to tenant service URL
type RegistryURLResolver interface {
	Resolve(tenantID auth.TenantID) (string, error)
}

// TenantInternalClient represents an HTTP client for calling tenant internal APIs
type TenantInternalClient struct {
	resolver   RegistryURLResolver
	httpClient *http.Client
}

// NewTenantInternalClient creates a new TenantInternalClient with registry URL resolver
func NewTenantInternalClient(resolver RegistryURLResolver) *TenantInternalClient {
	return &TenantInternalClient{
		resolver: resolver,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// VerifyRequest represents the request payload for user verification
type VerifyRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// VerifyResponse represents the response from tenant verification API
type VerifyResponse struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Verify makes an HTTP POST to the tenant's internal API and returns a User entity or error
func (c *TenantInternalClient) Verify(ctx context.Context, tenantID auth.TenantID, username string, password auth.PlainTextPassword) (*auth.User, error) {
	// Resolve tenant service URL
	baseURL, err := c.resolver.Resolve(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tenant URL: %w", err)
	}
	
	// Build verification endpoint URL
	url := fmt.Sprintf("%s/api/internal/auth/verify", baseURL)
	
	// Prepare request payload
	payload := VerifyRequest{
		Username: username,
		Password: password.String(),
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "auth-gateway")
	req.Header.Set("X-Tenant-ID", tenantID.String())
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute verification request: %w", err)
	}
	defer resp.Body.Close()
	
	// Handle response status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Success case - decode user response
		var verifyResp VerifyResponse
		if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
			return nil, fmt.Errorf("failed to decode verification response: %w", err)
		}
		
		// Convert response to User entity
		user, err := auth.NewUser(verifyResp.UserID, tenantID, verifyResp.Username, verifyResp.Role)
		if err != nil {
			return nil, fmt.Errorf("failed to create user entity: %w", err)
		}
		
		// Update user state based on response
		if !verifyResp.IsActive {
			user.Deactivate()
		}
		
		return user, nil
		
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("invalid credentials")
		
	case http.StatusNotFound:
		return nil, fmt.Errorf("user not found")
		
	case http.StatusForbidden:
		return nil, fmt.Errorf("user account is disabled")
		
	default:
		return nil, fmt.Errorf("tenant service returned unexpected status: %d", resp.StatusCode)
	}
}
