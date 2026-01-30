package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pushp314/devconnect-backend/internal/config"
	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client
var Ctx = context.Background()

func InitRedis() {
	Redis = redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       0,
	})

	_, err := Redis.Ping(Ctx).Result()
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v. Rate limiting and caching will be disabled.", err)
	} else {
		log.Println("Connected to Redis successfully")
	}
}

// Rate Limiting
func CheckRateLimit(userId string, limit int, duration time.Duration) (bool, error) {
	key := fmt.Sprintf("rate_limit:%s", userId)
	count, err := Redis.Incr(Ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		Redis.Expire(Ctx, key, duration)
	}

	if count > int64(limit) {
		return false, nil
	}
	return true, nil
}

// Caching
func CacheSet(key string, value interface{}, expiration time.Duration) error {
	json, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Redis.Set(Ctx, key, json, expiration).Err()
}

func CacheGet(key string, dest interface{}) error {
	val, err := Redis.Get(Ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func CacheInvalidate(pattern string) error {
	keys, err := Redis.Keys(Ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return Redis.Del(Ctx, keys...).Err()
	}
	return nil
}

// ===========================================
// P0 FIX: JWT Token Blacklist for Revocation
// ===========================================

const tokenBlacklistPrefix = "jwt:blacklist:"

// BlacklistToken adds a token JTI to the blacklist with the given TTL
// This is used for logout and password reset to immediately invalidate tokens
func BlacklistToken(jti string, ttl time.Duration) error {
	if Redis == nil {
		return fmt.Errorf("redis not initialized")
	}
	if jti == "" {
		return nil // Silently ignore tokens without JTI (legacy tokens)
	}

	key := tokenBlacklistPrefix + jti
	return Redis.Set(Ctx, key, "1", ttl).Err()
}

// IsTokenBlacklisted checks if a token JTI is in the blacklist
// Returns true if blacklisted, false otherwise
// On Redis error, returns false to avoid blocking auth (fail-open for availability)
func IsTokenBlacklisted(jti string) bool {
	if Redis == nil || jti == "" {
		return false // If Redis is down or no JTI, allow through (legacy compat)
	}

	key := tokenBlacklistPrefix + jti
	exists, err := Redis.Exists(Ctx, key).Result()
	if err != nil {
		// Log error but don't block auth on Redis failure
		log.Printf("Warning: Redis blacklist check failed: %v", err)
		return false
	}

	return exists > 0
}
