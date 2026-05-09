package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/go-redis/redis/v8"
	
	"github.com/your-project/internal/domain/auth"
)

// RedisOTPStore implements the OTPStore interface using Redis
type RedisOTPStore struct {
	client *redis.Client
	prefix string
}

// NewRedisOTPStore creates a new RedisOTPStore with the given Redis client
func NewRedisOTPStore(client *redis.Client) *RedisOTPStore {
	return &RedisOTPStore{
		client: client,
		prefix: "otp:",
	}
}

// SaveOTP saves an OTP code with its payload and TTL
func (r *RedisOTPStore) SaveOTP(ctx context.Context, code string, payload *auth.OTPPayload, ttl time.Duration) error {
	if code == "" {
		return fmt.Errorf("OTP code cannot be empty")
	}
	
	if payload == nil {
		return fmt.Errorf("OTP payload cannot be nil")
	}
	
	// Serialize payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize OTP payload: %w", err)
	}
	
	// Store OTP code as key and payload as value
	key := r.buildOTPKey(code)
	err = r.client.Set(ctx, key, payloadJSON, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store OTP in Redis: %w", err)
	}
	
	return nil
}

// GetAndDeleteOTP retrieves and deletes an OTP code, returning the payload
func (r *RedisOTPStore) GetAndDeleteOTP(ctx context.Context, code string) (*auth.OTPPayload, error) {
	if code == "" {
		return nil, fmt.Errorf("OTP code cannot be empty")
	}
	
	key := r.buildOTPKey(code)
	
	// Use Redis transaction to get and delete atomically
	pipe := r.client.TxPipeline()
	getCmd := pipe.Get(ctx, key)
	pipe.Del(ctx, key)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		if err == redis.Nil {
			return nil, nil // OTP not found (expired or doesn't exist)
		}
		return nil, fmt.Errorf("failed to retrieve OTP from Redis: %w", err)
	}
	
	// Get the payload value
	payloadJSON, err := getCmd.Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // OTP not found
		}
		return nil, fmt.Errorf("failed to get OTP value: %w", err)
	}
	
	// Deserialize payload
	var payload auth.OTPPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return nil, fmt.Errorf("failed to deserialize OTP payload: %w", err)
	}
	
	return &payload, nil
}

// buildKey builds the Redis key for storing OTP (legacy method)
func (r *RedisOTPStore) buildKey(tenantID auth.TenantID, userID string) string {
	return fmt.Sprintf("%s%s:%s", r.prefix, tenantID.String(), userID)
}

// buildOTPKey builds the Redis key for OTP code (new interface)
func (r *RedisOTPStore) buildOTPKey(code string) string {
	return fmt.Sprintf("%scode:%s", r.prefix, code)
}

// GetOTPCount gets the number of active OTP codes for a tenant
// This is a helper method not part of the interface but useful for monitoring
func (r *RedisOTPStore) GetOTPCount(ctx context.Context, tenantID auth.TenantID) (int64, error) {
	pattern := r.buildKey(tenantID, "*")
	
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get OTP keys: %w", err)
	}
	
	return int64(len(keys)), nil
}

// IsOTPExists checks if an OTP exists for a user
// This is a helper method not part of the interface
func (r *RedisOTPStore) IsOTPExists(ctx context.Context, tenantID auth.TenantID, userID string) (bool, error) {
	key := r.buildKey(tenantID, userID)
	
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check OTP existence: %w", err)
	}
	
	return exists > 0, nil
}

// GetOTPTTL gets the remaining time-to-live for an OTP
// This is a helper method not part of the interface
func (r *RedisOTPStore) GetOTPTTL(ctx context.Context, tenantID auth.TenantID, userID string) (time.Duration, error) {
	key := r.buildKey(tenantID, userID)
	
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get OTP TTL: %w", err)
	}
	
	return ttl, nil
}
