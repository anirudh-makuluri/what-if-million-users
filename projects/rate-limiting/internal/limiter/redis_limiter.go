package limiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"rate-limiting/internal/metrics"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter implements token bucket rate limiting using Redis
type RedisLimiter struct {
	client *redis.Client
}

// LimitResult contains the result of rate limit check
type LimitResult struct {
	Allowed      bool
	TokensLeft   int
	ResetAfter   time.Duration
	RateLimitKey string
}

// NewRedisLimiter creates a new Redis-backed rate limiter
func NewRedisLimiter(client *redis.Client) *RedisLimiter {
	return &RedisLimiter{client: client}
}

func (rl *RedisLimiter) CheckLimit(ctx context.Context, clientID string, maxTokens int, refillRate int, refillIntervalSeconds int) *LimitResult {
	key := fmt.Sprintf("rate_limit:%s", clientID)
	now := time.Now().Unix()

	// Lua script to atomically check and update tokens
	script := redis.NewScript(`
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local max_tokens = tonumber(ARGV[2])
		local refill_rate = tonumber(ARGV[3])
		local refill_interval = tonumber(ARGV[4])

		local current = redis.call('HGETALL', key)
		local tokens = max_tokens
		local last_refill = now

		if #current > 0 then
			tokens = tonumber(current[2])
			last_refill = tonumber(current[4])
		end

		-- Calculate refilled tokens
		local elapsed = now - last_refill
		local refills = math.floor(elapsed / refill_interval)
		local refilled_tokens = tokens + (refills * refill_rate)
		if refilled_tokens > max_tokens then
			refilled_tokens = max_tokens
		end

		local allowed = 0
		if refilled_tokens >= 1 then
			allowed = 1
			refilled_tokens = refilled_tokens - 1
		end

		redis.call('HSET', key, 'tokens', refilled_tokens, 'last_refill', now)
		redis.call('EXPIRE', key, 3600)

		return {allowed, refilled_tokens, last_refill}
	`)

	var cmd *redis.Cmd = script.Run(ctx, rl.client, []string{key}, now, maxTokens, refillRate, refillIntervalSeconds)
	if err := cmd.Err(); err != nil {
		metrics.RecordLimiterError()
		return &LimitResult{Allowed: false}
	}
	result := cmd.Val()

	resultSlice := result.([]interface{})
	allowed := resultSlice[0].(int64) == 1
	tokensLeft := int(resultSlice[1].(int64))

	if allowed {
		metrics.RecordAllowedRequest()
	} else {
		metrics.RecordRejectedRequest()
	}

	return &LimitResult{
		Allowed:      allowed,
		TokensLeft:   tokensLeft,
		RateLimitKey: key,
	}
}

// GetStatus gets the current status of a rate limit key
func (rl *RedisLimiter) GetStatus(ctx context.Context, clientID string) map[string]interface{} {
	key := fmt.Sprintf("rate_limit:%s", clientID)
	data, err := rl.client.HGetAll(ctx, key).Result()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	if len(data) == 0 {
		return map[string]interface{}{
			"exists": false,
		}
	}

	tokens, _ := strconv.Atoi(data["tokens"])
	return map[string]interface{}{
		"exists": true,
		"tokens": tokens,
	}
}

// Reset resets the rate limit for a client
func (rl *RedisLimiter) Reset(ctx context.Context, clientID string) error {
	key := fmt.Sprintf("rate_limit:%s", clientID)
	return rl.client.Del(ctx, key).Err()
}
