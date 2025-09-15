package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Repository interface for authentication operations
type Repository interface {
	StoreToken(ctx context.Context, userID, token string, expiry time.Duration) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	BlacklistToken(ctx context.Context, token string, expiry time.Duration) error
	CleanupExpiredTokens(ctx context.Context) error
}

// redisRepository implements Repository using Redis
type redisRepository struct {
	client *redis.Client
}

// NewRepository creates a new Redis-based auth repository
func NewRepository(client *redis.Client) Repository {
	return &redisRepository{client: client}
}

// StoreToken stores an active token in Redis with expiry
func (r *redisRepository) StoreToken(ctx context.Context, userID, token string, expiry time.Duration) error {
	key := fmt.Sprintf("token:%s", userID)

	err := r.client.Set(ctx, key, token, expiry).Err()
	if err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// IsTokenBlacklisted checks if a token is in the blacklist
func (r *redisRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", token)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	return exists > 0, nil
}

// BlacklistToken adds a token to the blacklist
func (r *redisRepository) BlacklistToken(ctx context.Context, token string, expiry time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", token)

	err := r.client.Set(ctx, key, "revoked", expiry).Err()
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// CleanupExpiredTokens removes expired tokens (Redis handles this automatically with TTL)
func (r *redisRepository) CleanupExpiredTokens(ctx context.Context) error {
	// Redis automatically handles expired keys with TTL
	// This method can be used for manual cleanup if needed
	return nil
}

// GetUserToken retrieves the active token for a user
func (r *redisRepository) GetUserToken(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("token:%s", userID)

	token, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("no active token for user: %s", userID)
		}
		return "", fmt.Errorf("failed to get user token: %w", err)
	}

	return token, nil
}

// RevokeUserToken removes the active token for a user
func (r *redisRepository) RevokeUserToken(ctx context.Context, userID string) error {
	key := fmt.Sprintf("token:%s", userID)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to revoke user token: %w", err)
	}

	return nil
}

// StoreRefreshToken stores a refresh token (for future use)
func (r *redisRepository) StoreRefreshToken(ctx context.Context, userID, refreshToken string, expiry time.Duration) error {
	key := fmt.Sprintf("refresh:%s", userID)

	err := r.client.Set(ctx, key, refreshToken, expiry).Err()
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// ValidateRefreshToken validates a refresh token (for future use)
func (r *redisRepository) ValidateRefreshToken(ctx context.Context, userID, refreshToken string) (bool, error) {
	key := fmt.Sprintf("refresh:%s", userID)

	storedToken, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return storedToken == refreshToken, nil
}