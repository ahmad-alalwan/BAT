package contextx

import (
	"context"
	"time"
	
	"github.com/your-project/internal/domain/auth"
)

// tenantContextKey is the type for the tenant ID context key
type tenantContextKey string

const (
	// TenantKey is the context key used to store and retrieve TenantID
	TenantKey tenantContextKey = "tenant_id"
)

// SetTenantID injects a TenantID into the context
func SetTenantID(ctx context.Context, tenantID auth.TenantID) context.Context {
	return context.WithValue(ctx, TenantKey, tenantID)
}

// GetTenantID retrieves a TenantID from the context
// Returns the TenantID and a boolean indicating if it was found
func GetTenantID(ctx context.Context) (auth.TenantID, bool) {
	tenantID, ok := ctx.Value(TenantKey).(auth.TenantID)
	return tenantID, ok
}

// MustGetTenantID retrieves a TenantID from the context and panics if not found
// Use this function only when you're certain the tenant ID should be present
func MustGetTenantID(ctx context.Context) auth.TenantID {
	tenantID, ok := GetTenantID(ctx)
	if !ok {
		panic("tenant ID not found in context")
	}
	return tenantID
}

// WithTenant creates a new context with tenant ID and executes the function
// This is a helper function for operations that require tenant context
func WithTenant(ctx context.Context, tenantID auth.TenantID, fn func(context.Context) error) error {
	tenantCtx := SetTenantID(ctx, tenantID)
	return fn(tenantCtx)
}

// ExtractTenantFromContext extracts tenant ID string from context
// Returns empty string if not found
func ExtractTenantFromContext(ctx context.Context) string {
	tenantID, ok := GetTenantID(ctx)
	if !ok {
		return ""
	}
	return tenantID.String()
}

// TenantAwareContext wraps a context with tenant information
type TenantAwareContext struct {
	context.Context
	tenantID auth.TenantID
}

// NewTenantAwareContext creates a new tenant-aware context
func NewTenantAwareContext(ctx context.Context, tenantID auth.TenantID) *TenantAwareContext {
	return &TenantAwareContext{
		Context:  SetTenantID(ctx, tenantID),
		tenantID: tenantID,
	}
}

// TenantID returns the tenant ID for this context
func (tac *TenantAwareContext) TenantID() auth.TenantID {
	return tac.tenantID
}

// Value implements context.Value method
func (tac *TenantAwareContext) Value(key interface{}) interface{} {
	if key == TenantKey {
		return tac.tenantID
	}
	return tac.Context.Value(key)
}

// WithTenantTimeout creates a context with tenant ID and timeout
func WithTenantTimeout(parent context.Context, tenantID auth.TenantID, timeout time.Duration) (context.Context, context.CancelFunc) {
	tenantCtx := SetTenantID(parent, tenantID)
	return context.WithTimeout(tenantCtx, timeout)
}

// WithTenantCancel creates a context with tenant ID and cancel function
func WithTenantCancel(parent context.Context, tenantID auth.TenantID) (context.Context, context.CancelFunc) {
	tenantCtx := SetTenantID(parent, tenantID)
	return context.WithCancel(tenantCtx)
}

// WithTenantDeadline creates a context with tenant ID and deadline
func WithTenantDeadline(parent context.Context, tenantID auth.TenantID, deadline time.Time) (context.Context, context.CancelFunc) {
	tenantCtx := SetTenantID(parent, tenantID)
	return context.WithDeadline(tenantCtx, deadline)
}
