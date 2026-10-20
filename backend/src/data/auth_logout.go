package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBlacklistRepository struct{ Client *redis.Client }

func (r TokenBlacklistRepository) client() (*redis.Client, error) {
	client := r.Client
	if client == nil {
		client = GetRedis()
	}
	if client == nil {
		return nil, errors.New("Redis is not initialized")
	}
	return client, nil
}

func (r TokenBlacklistRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	client, err := r.client()
	if err != nil {
		return false, err
	}
	count, err := client.Exists(ctx, "blacklist:"+token).Result()
	if err != nil {
		return false, fmt.Errorf("check token blacklist: %w", err)
	}
	return count > 0, nil
}

// BlacklistToken 原子写入，已有记录时不刷新其过期时间。
func (r TokenBlacklistRepository) BlacklistToken(ctx context.Context, token string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, errors.New("blacklist TTL must be positive")
	}
	client, err := r.client()
	if err != nil {
		return false, err
	}
	added, err := client.SetNX(ctx, "blacklist:"+token, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("write token blacklist: %w", err)
	}
	return added, nil
}
