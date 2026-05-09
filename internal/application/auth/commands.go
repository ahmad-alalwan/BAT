package auth

import (
	"context"
	"fmt"
	
	"github.com/your-project/internal/domain/auth"
)

// LoginCommand represents a login request with multi-tenancy support
type LoginCommand struct {
	TenantID auth.TenantID `json:"tenant_id"`
	Email    string        `json:"email"`
	Password string        `json:"password"`
}

// NewLoginCommand creates a new LoginCommand with validation
func NewLoginCommand(tenantID auth.TenantID, email, password string) (*LoginCommand, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}
	
	return &LoginCommand{
		TenantID: tenantID,
		Email:    email,
		Password: password,
	}, nil
}

// Validate validates the login command
func (c LoginCommand) Validate() error {
	if c.Email == "" {
		return fmt.Errorf("email is required")
	}
	
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}
	
	return nil
}

// ToPlainTextPassword converts the password string to PlainTextPassword value object
func (c LoginCommand) ToPlainTextPassword() (*auth.PlainTextPassword, error) {
	return auth.NewPlainTextPassword(c.Password)
}

// GenerateOTPCommand represents a request to generate OTP for a user
type GenerateOTPCommand struct {
	TenantID auth.TenantID `json:"tenant_id"`
	UserID   string        `json:"user_id"`
}

// NewGenerateOTPCommand creates a new GenerateOTPCommand with validation
func NewGenerateOTPCommand(tenantID auth.TenantID, userID string) (*GenerateOTPCommand, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	
	return &GenerateOTPCommand{
		TenantID: tenantID,
		UserID:   userID,
	}, nil
}

// Validate validates the generate OTP command
func (c GenerateOTPCommand) Validate() error {
	if c.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	
	return nil
}

// ValidateOTPCommand represents a request to validate an OTP code
type ValidateOTPCommand struct {
	TenantID auth.TenantID  `json:"tenant_id"`
	UserID   string         `json:"user_id"`
	OTPCode  string         `json:"otp_code"`
}

// NewValidateOTPCommand creates a new ValidateOTPCommand with validation
func NewValidateOTPCommand(tenantID auth.TenantID, userID, otpCode string) (*ValidateOTPCommand, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	
	if otpCode == "" {
		return nil, fmt.Errorf("OTP code cannot be empty")
	}
	
	return &ValidateOTPCommand{
		TenantID: tenantID,
		UserID:   userID,
		OTPCode:  otpCode,
	}, nil
}

// Validate validates the validate OTP command
func (c ValidateOTPCommand) Validate() error {
	if c.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	
	if c.OTPCode == "" {
		return fmt.Errorf("OTP code is required")
	}
	
	return nil
}

// ToOTPCode converts the OTP code string to OTPCode value object
func (c ValidateOTPCommand) ToOTPCode() (*auth.OTPCode, error) {
	return auth.NewOTPCode(c.OTPCode)
}

// LoginWithOTPCommand represents a login request using OTP
type LoginWithOTPCommand struct {
	TenantID auth.TenantID `json:"tenant_id"`
	UserID   string        `json:"user_id"`
	OTPCode  string        `json:"otp_code"`
}

// NewLoginWithOTPCommand creates a new LoginWithOTPCommand with validation
func NewLoginWithOTPCommand(tenantID auth.TenantID, userID, otpCode string) (*LoginWithOTPCommand, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	
	if otpCode == "" {
		return nil, fmt.Errorf("OTP code cannot be empty")
	}
	
	return &LoginWithOTPCommand{
		TenantID: tenantID,
		UserID:   userID,
		OTPCode:  otpCode,
	}, nil
}

// Validate validates the login with OTP command
func (c LoginWithOTPCommand) Validate() error {
	if c.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	
	if c.OTPCode == "" {
		return fmt.Errorf("OTP code is required")
	}
	
	return nil
}

// ToOTPCode converts the OTP code string to OTPCode value object
func (c LoginWithOTPCommand) ToOTPCode() (*auth.OTPCode, error) {
	return auth.NewOTPCode(c.OTPCode)
}
