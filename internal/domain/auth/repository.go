package auth

import (
	"context"
	"time"
)

// UserVerifier defines the interface for verifying user credentials
// This interface belongs to the domain layer but will be implemented by infrastructure
type UserVerifier interface {
	// VerifyCredentials verifies user credentials and returns the user if valid
	VerifyCredentials(ctx context.Context, tenantID TenantID, username string, password PlainTextPassword) (*User, error)
}

// OTPPayload represents the payload stored with an OTP code
type OTPPayload struct {
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// OTPStore defines the interface for storing and retrieving OTP codes
// This interface belongs to the domain layer but will be implemented by infrastructure (Redis, etc.)
type OTPStore interface {
	// SaveOTP saves an OTP code with its payload and TTL
	SaveOTP(ctx context.Context, code string, payload *OTPPayload, ttl time.Duration) error
	
	// GetAndDeleteOTP retrieves and deletes an OTP code, returning the payload
	GetAndDeleteOTP(ctx context.Context, code string) (*OTPPayload, error)
}
