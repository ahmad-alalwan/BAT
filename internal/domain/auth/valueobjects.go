package auth

import (
	"fmt"
	"regexp"
	"strings"
)

// TenantID represents a tenant identifier as a value object
type TenantID struct {
	value string
}

// NewTenantID creates a new TenantID with validation
func NewTenantID(value string) (*TenantID, error) {
	if value == "" {
		return nil, fmt.Errorf("tenant ID cannot be empty")
	}
	
	// Validate slug format: lowercase letters, numbers, and hyphens only
	slugRegex := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	if !slugRegex.MatchString(value) {
		return nil, fmt.Errorf("tenant ID must be in slug format (lowercase letters, numbers, and hyphens only)")
	}
	
	if len(value) > 50 {
		return nil, fmt.Errorf("tenant ID cannot exceed 50 characters")
	}
	
	return &TenantID{value: value}, nil
}

// String returns the string representation of TenantID
func (t TenantID) String() string {
	return t.value
}

// Equals checks if two TenantIDs are equal
func (t TenantID) Equals(other TenantID) bool {
	return t.value == other.value
}

// PlainTextPassword represents a plain text password as a value object
type PlainTextPassword struct {
	value string
}

// NewPlainTextPassword creates a new PlainTextPassword
func NewPlainTextPassword(value string) (*PlainTextPassword, error) {
	if value == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}
	
	if len(value) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters long")
	}
	
	if len(value) > 128 {
		return nil, fmt.Errorf("password cannot exceed 128 characters")
	}
	
	return &PlainTextPassword{value: value}, nil
}

// Sanitize removes leading/trailing whitespace and returns the sanitized password
func (p PlainTextPassword) Sanitize() PlainTextPassword {
	sanitized := strings.TrimSpace(p.value)
	return PlainTextPassword{value: sanitized}
}

// String returns the string representation of PlainTextPassword
func (p PlainTextPassword) String() string {
	return p.value
}

// IsEmpty checks if the password is empty
func (p PlainTextPassword) IsEmpty() bool {
	return p.value == ""
}

// OTPCode represents a one-time password code as a value object
type OTPCode struct {
	value string
}

// NewOTPCode creates a new OTPCode with validation
func NewOTPCode(value string) (*OTPCode, error) {
	if value == "" {
		return nil, fmt.Errorf("OTP code cannot be empty")
	}
	
	// Validate OTP format: exactly 6 digits
	otpRegex := regexp.MustCompile(`^\d{6}$`)
	if !otpRegex.MatchString(value) {
		return nil, fmt.Errorf("OTP code must be exactly 6 digits")
	}
	
	return &OTPCode{value: value}, nil
}

// String returns the string representation of OTPCode
func (o OTPCode) String() string {
	return o.value
}

// Equals checks if two OTPCodes are equal
func (o OTPCode) Equals(other OTPCode) bool {
	return o.value == other.value
}

// IsValid checks if the OTP code is in valid format
func (o OTPCode) IsValid() bool {
	otpRegex := regexp.MustCompile(`^\d{6}$`)
	return otpRegex.MatchString(o.value)
}
