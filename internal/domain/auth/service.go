package auth

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// AuthDomainService defines the interface for authentication domain services
// This service contains core business logic for authentication
type AuthDomainService interface {
	// AuthenticateUser authenticates a user with email and password
	// Follows zero-knowledge pattern - delegates password verification to external boundary
	AuthenticateUser(ctx context.Context, tenantID TenantID, email string, password PlainTextPassword) (*User, error)
	
	// GenerateOTP generates a new OTP code for a user
	GenerateOTP(ctx context.Context, tenantID TenantID, userID string) (*OTPCode, error)
	
	// ValidateOTP validates an OTP code for a user
	ValidateOTP(ctx context.Context, tenantID TenantID, userID string, otpCode OTPCode) (bool, error)
	
	// AuthenticateWithOTP authenticates a user using OTP code
	AuthenticateWithOTP(ctx context.Context, tenantID TenantID, userID string, otpCode OTPCode) (*User, error)
}

// authDomainService implements AuthDomainService
type authDomainService struct {
	userRepo        UserRepository
	otpStore        OTPStore
	passwordVerifier PasswordVerifier
}

// NewAuthDomainService creates a new instance of AuthDomainService
// Uses constructor-based dependency injection
func NewAuthDomainService(
	userRepo UserRepository,
	otpStore OTPStore,
	passwordVerifier PasswordVerifier,
) AuthDomainService {
	return &authDomainService{
		userRepo:        userRepo,
		otpStore:        otpStore,
		passwordVerifier: passwordVerifier,
	}
}

// AuthenticateUser authenticates a user with email and password
func (s *authDomainService) AuthenticateUser(
	ctx context.Context,
	tenantID TenantID,
	email string,
	password PlainTextPassword,
) (*User, error) {
	// Sanitize password input
	password = password.Sanitize()
	
	// Find user by email within the specified tenant
	user, err := s.userRepo.FindByEmail(ctx, tenantID, email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	
	// Check if user is active
	if !user.IsActive() {
		return nil, fmt.Errorf("user account is deactivated")
	}
	
	// Verify password using the external password verifier (zero-knowledge pattern)
	isValid, err := s.passwordVerifier.Verify(ctx, tenantID, user.ID(), password)
	if err != nil {
		return nil, fmt.Errorf("password verification failed: %w", err)
	}
	
	if !isValid {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	return user, nil
}

// GenerateOTP generates a new OTP code for a user
func (s *authDomainService) GenerateOTP(
	ctx context.Context,
	tenantID TenantID,
	userID string,
) (*OTPCode, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	
	// Generate 6-digit OTP code
	otpCode := s.generateRandomOTP()
	
	// Store the OTP code
	err := s.otpStore.Store(ctx, tenantID, userID, *otpCode)
	if err != nil {
		return nil, fmt.Errorf("failed to store OTP: %w", err)
	}
	
	return otpCode, nil
}

// ValidateOTP validates an OTP code for a user
func (s *authDomainService) ValidateOTP(
	ctx context.Context,
	tenantID TenantID,
	userID string,
	otpCode OTPCode,
) (bool, error) {
	if userID == "" {
		return false, fmt.Errorf("user ID cannot be empty")
	}
	
	// Validate OTP format
	if !otpCode.IsValid() {
		return false, fmt.Errorf("invalid OTP format")
	}
	
	// Use the OTP store to validate
	isValid, err := s.otpStore.Validate(ctx, tenantID, userID, otpCode)
	if err != nil {
		return false, fmt.Errorf("OTP validation failed: %w", err)
	}
	
	return isValid, nil
}

// AuthenticateWithOTP authenticates a user using OTP code
func (s *authDomainService) AuthenticateWithOTP(
	ctx context.Context,
	tenantID TenantID,
	userID string,
	otpCode OTPCode,
) (*User, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	
	// Validate OTP first
	isValid, err := s.ValidateOTP(ctx, tenantID, userID, otpCode)
	if err != nil {
		return nil, fmt.Errorf("OTP validation failed: %w", err)
	}
	
	if !isValid {
		return nil, fmt.Errorf("invalid or expired OTP")
	}
	
	// Find user by ID within the specified tenant
	user, err := s.userRepo.FindByID(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	
	// Check if user is active
	if !user.IsActive() {
		return nil, fmt.Errorf("user account is deactivated")
	}
	
	// Delete the OTP after successful authentication
	err = s.otpStore.Delete(ctx, tenantID, userID)
	if err != nil {
		// Log error but don't fail authentication
		// In production, you might want to use proper logging
		fmt.Printf("Warning: failed to delete OTP after authentication: %v\n", err)
	}
	
	return user, nil
}

// generateRandomOTP generates a random 6-digit OTP code
func (s *authDomainService) generateRandomOTP() *OTPCode {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
	
	// Generate random 6-digit number
	otpValue := rand.Intn(900000) + 100000 // Ensures 6 digits (100000-999999)
	
	otpCode, err := NewOTPCode(fmt.Sprintf("%d", otpValue))
	if err != nil {
		// This should never happen since we control the format
		panic(fmt.Sprintf("Failed to generate OTP: %v", err))
	}
	
	return otpCode
}
