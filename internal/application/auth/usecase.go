package auth

import (
	"context"
	"fmt"
	
	"github.com/your-project/internal/domain/auth"
)

// AuthUseCase orchestrates authentication operations
// This is the application layer that coordinates between domain services and infrastructure
type AuthUseCase struct {
	authService auth.AuthDomainService
}

// NewAuthUseCase creates a new AuthUseCase with dependency injection
func NewAuthUseCase(authService auth.AuthDomainService) *AuthUseCase {
	return &AuthUseCase{
		authService: authService,
	}
}

// LoginResponse represents the response after successful authentication
type LoginResponse struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	Token    string `json:"token"` // In a real implementation, this would be a JWT or similar
}

// Login handles user authentication with email and password
func (uc *AuthUseCase) Login(ctx context.Context, cmd LoginCommand) (*LoginResponse, error) {
	// Validate the command
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("invalid login command: %w", err)
	}
	
	// Convert password to value object
	password, err := cmd.ToPlainTextPassword()
	if err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}
	
	// Authenticate user using domain service
	user, err := uc.authService.AuthenticateUser(ctx, cmd.TenantID, cmd.Email, *password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	// Generate token (in a real implementation, you'd use JWT or similar)
	token := uc.generateToken(user)
	
	return &LoginResponse{
		UserID:   user.ID(),
		Email:    user.Email(),
		TenantID: user.TenantID().String(),
		Token:    token,
	}, nil
}

// GenerateOTPResponse represents the response after generating OTP
type GenerateOTPResponse struct {
	OTPCode string `json:"otp_code"` // In production, you might not want to return the actual OTP
	Message string `json:"message"`
}

// GenerateOTP generates an OTP code for a user
func (uc *AuthUseCase) GenerateOTP(ctx context.Context, cmd GenerateOTPCommand) (*GenerateOTPResponse, error) {
	// Validate the command
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("invalid generate OTP command: %w", err)
	}
	
	// Generate OTP using domain service
	otpCode, err := uc.authService.GenerateOTP(ctx, cmd.TenantID, cmd.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP: %w", err)
	}
	
	// In a real implementation, you would send this OTP via email/SMS
	// For demo purposes, we'll return it in the response
	return &GenerateOTPResponse{
		OTPCode: otpCode.String(),
		Message: "OTP generated successfully",
	}, nil
}

// ValidateOTPResponse represents the response after validating OTP
type ValidateOTPResponse struct {
	IsValid bool   `json:"is_valid"`
	Message string `json:"message"`
}

// ValidateOTP validates an OTP code for a user
func (uc *AuthUseCase) ValidateOTP(ctx context.Context, cmd ValidateOTPCommand) (*ValidateOTPResponse, error) {
	// Validate the command
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("invalid validate OTP command: %w", err)
	}
	
	// Convert OTP code to value object
	otpCode, err := cmd.ToOTPCode()
	if err != nil {
		return nil, fmt.Errorf("invalid OTP code: %w", err)
	}
	
	// Validate OTP using domain service
	isValid, err := uc.authService.ValidateOTP(ctx, cmd.TenantID, cmd.UserID, *otpCode)
	if err != nil {
		return nil, fmt.Errorf("OTP validation failed: %w", err)
	}
	
	message := "OTP is invalid"
	if isValid {
		message = "OTP is valid"
	}
	
	return &ValidateOTPResponse{
		IsValid: isValid,
		Message: message,
	}, nil
}

// LoginWithOTP handles user authentication using OTP
func (uc *AuthUseCase) LoginWithOTP(ctx context.Context, cmd LoginWithOTPCommand) (*LoginResponse, error) {
	// Validate the command
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("invalid login with OTP command: %w", err)
	}
	
	// Convert OTP code to value object
	otpCode, err := cmd.ToOTPCode()
	if err != nil {
		return nil, fmt.Errorf("invalid OTP code: %w", err)
	}
	
	// Authenticate user using domain service
	user, err := uc.authService.AuthenticateWithOTP(ctx, cmd.TenantID, cmd.UserID, *otpCode)
	if err != nil {
		return nil, fmt.Errorf("OTP authentication failed: %w", err)
	}
	
	// Generate token (in a real implementation, you'd use JWT or similar)
	token := uc.generateToken(user)
	
	return &LoginResponse{
		UserID:   user.ID(),
		Email:    user.Email(),
		TenantID: user.TenantID().String(),
		Token:    token,
	}, nil
}

// generateToken generates a simple token for the user
// In a real implementation, you would use JWT or another secure token mechanism
func (uc *AuthUseCase) generateToken(user *auth.User) string {
	// This is a placeholder implementation
	// In production, use a proper JWT library or secure token generation
	return fmt.Sprintf("token_%s_%s_%d", user.ID(), user.TenantID().String(), user.CreatedAt().Unix())
}
