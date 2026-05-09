package auth

import (
	"fmt"
	"time"
)

// User represents the User entity in the domain
type User struct {
	id       string
	tenantID TenantID
	username string
	role     string
	isActive bool
	createdAt time.Time
	updatedAt time.Time
}

// NewUser creates a new User entity
func NewUser(id string, tenantID TenantID, username, role string) (*User, error) {
	if id == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	
	if role == "" {
		return nil, fmt.Errorf("role cannot be empty")
	}
	
	// Basic username validation
	if len(username) < 3 || len(username) > 50 {
		return nil, fmt.Errorf("username must be between 3 and 50 characters")
	}
	
	now := time.Now()
	
	return &User{
		id:        id,
		tenantID:  tenantID,
		username:  username,
		role:      role,
		isActive:  true,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// ID returns the user ID
func (u User) ID() string {
	return u.id
}

// GetID returns the UUID of the user (alias for ID method)
func (u User) GetID() string {
	return u.id
}

// TenantID returns the tenant ID
func (u User) TenantID() TenantID {
	return u.tenantID
}

// Username returns the username
func (u User) Username() string {
	return u.username
}

// Role returns the user role
func (u User) Role() string {
	return u.role
}

// IsActive returns whether the user is active
func (u User) IsActive() bool {
	return u.isActive
}

// CreatedAt returns the creation timestamp
func (u User) CreatedAt() time.Time {
	return u.createdAt
}

// UpdatedAt returns the last update timestamp
func (u User) UpdatedAt() time.Time {
	return u.updatedAt
}

// Deactivate deactivates the user
func (u *User) Deactivate() {
	u.isActive = false
	u.updatedAt = time.Now()
}

// Activate activates the user
func (u *User) Activate() {
	u.isActive = true
	u.updatedAt = time.Now()
}

// UpdateUsername updates the username with validation
func (u *User) UpdateUsername(newUsername string) error {
	if newUsername == "" {
		return fmt.Errorf("username cannot be empty")
	}
	
	if len(newUsername) < 3 || len(newUsername) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}
	
	u.username = newUsername
	u.updatedAt = time.Now()
	
	return nil
}

// UpdateRole updates the user role with validation
func (u *User) UpdateRole(newRole string) error {
	if newRole == "" {
		return fmt.Errorf("role cannot be empty")
	}
	
	u.role = newRole
	u.updatedAt = time.Now()
	
	return nil
}

// BelongsToTenant checks if the user belongs to the given tenant
func (u User) BelongsToTenant(tenantID TenantID) bool {
	return u.tenantID.Equals(tenantID)
}

// Equals checks if two users are equal
func (u User) Equals(other User) bool {
	return u.id == other.id && u.tenantID.Equals(other.tenantID)
}

