package cache

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ttl   time.Duration
}


func NewRedisCache() (*RedisCache, error) {
	addr := os.Getenv("REDIS_ADDR")

	client := redis.NewClient(&redis.Options{
		Addr: addr,
		Password: "",
		DB: 0,
		DialTimeout: 5 * time.Second,
		ReadTimeout: 3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err	
	}

	return &RedisCache{
		client: client,
		ttl: 24 * time.Hour,
	}, nil
}

func (r *RedisCache) Set(ctx context.Context, shortCode string, longURL string) error {
	return r.client.Set(ctx, shortCode, longURL, r.ttl).Err()
}

func (r *RedisCache) Get(ctx context.Context, shortCode string) (string, bool, error) {
	val, err := r.client.Get(ctx, shortCode).Result()
	if err == redis.Nil {
		// key does not exist, not an error
		return "", false, nil
	}
	if err != nil {
		// actual error
		return "", false, err
	}
	return val, true, nil
}


func (r *RedisCache) Delete(ctx context.Context, shortCode string) error {
	return r.client.Del(ctx, shortCode).Err()
}