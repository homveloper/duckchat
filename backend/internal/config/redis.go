package config

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// DefaultRedisConfig returns default Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}
}

// NewRedisClient creates a new Redis client with health check
func NewRedisClient(config *RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// Health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Test RedisJSON support
	_, err = client.Do(ctx, "JSON.SET", "health_check", "$", `{"status":"ok"}`).Result()
	if err != nil {
		return nil, fmt.Errorf("RedisJSON module not available: %w", err)
	}

	// Clean up test key
	client.Del(ctx, "health_check")

	log.Printf("Redis connected successfully with RedisJSON support at %s:%d", config.Host, config.Port)
	return client, nil
}

// GetRedisClient returns a configured Redis client instance
func GetRedisClient() (*redis.Client, error) {
	config := DefaultRedisConfig()
	return NewRedisClient(config)
}