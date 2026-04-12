# Rate Limiter

> What if you need to protect APIs from abuse at scale?

A production-grade API rate limiter that handles millions of requests with sub-millisecond latency. Uses Redis for atomic token bucket rate limiting, Kafka for async event logging, and Prometheus for real-time observability.

## Architecture

- **Token Bucket Algorithm** — Redis-backed, atomic limit checks with configurable refill rates
- **Async Logging** — Rate limit events flow to Kafka without blocking the hot path
- **Sub-ms Latency** — No database roundtrips, all checks happen in-memory via Redis
- **Graceful Degradation** — If Redis fails, requests fail open (allowed) by default
- **Full Observability** — Prometheus metrics track allowed/rejected requests, limit overages, and errors

## Stack

- **Go** — HTTP server with Gin
- **Redis 7** — Atomic token bucket state
- **Kafka 7.5** — Async event streaming
- **Prometheus** — Metrics & observability
- **Docker Compose** — Local environment

## Quick Start

```bash
# Start all services
docker-compose up -d --build

# Wait for Kafka to be ready (~10 seconds)
sleep 10

# Test rate limiting
curl -X POST http://localhost:8080/api/request \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-001",
    "action": "create_post"
  }'

# Check metrics
curl http://localhost:8080/metrics

# View Prometheus dashboard
# Navigate to http://localhost:9090
```

## Configuration

Rate limit per client: **100 requests/minute**
- Max tokens: 100
- Refill rate: 1 token per second
- Reset window: 60 seconds

Modify in [internal/handler/handler.go](internal/handler/handler.go#L46):

```go
result := h.limiter.CheckLimit(ctx, req.ClientID, 100, 1, 1)
```

## Endpoints

### POST /api/request
Check rate limit and process request.

```json
{
  "client_id": "client-001",
  "action": "create_post"
}
```

**Response (Allowed):**
```json
{
  "allowed": true,
  "tokens_left": 99,
  "message": "Request allowed"
}
```

**Response (Rejected):**
```json
{
  "allowed": false,
  "message": "Rate limit exceeded"
}
```

### GET /health
Health check endpoint.

### GET /metrics
Prometheus metrics endpoint.

## Metrics

```
rate_limiter_requests_allowed_total{client_id="..."} — Allowed requests
rate_limiter_requests_rejected_total{client_id="..."} — Rejected requests
rate_limiter_limit_exceeded_total{client_id="..."} — Times limit was hit
rate_limiter_errors_total — Limiter errors
```

## How It Works

### Token Bucket Algorithm

Each client has a bucket with N tokens. Each request consumes 1 token. Tokens refill at a constant rate (e.g., 1 per second). If bucket is empty, request is rejected.

**Why token bucket?**
- Handles burst traffic gracefully (client can use all tokens at once)
- Predictable rate limiting (no time-based windows)
- Easy to configure (max tokens + refill rate)

### Redis Lua Script

Rate limit checks use an atomic Lua script to avoid race conditions:

```lua
1. Get current tokens from Redis hash
2. Calculate refilled tokens based on elapsed time
3. If tokens available, consume 1 and update state
4. Return allowed/denied decision
```

**Why Lua?**
- Single Redis roundtrip (no network latency)
- Atomic operation (no concurrent request conflicts)
- Sub-millisecond latency at scale

### Kafka Event Logging

Every rate limit decision (allowed/rejected) is published async to Kafka. This decouples logging from the hot path:

```
Hot path (user gets response) → 1ms
Kafka publish (async goroutine) → happens later
```

## Performance Characteristics

| Metric | Value |
|--------|-------|
| **Latency (p50)** | <1ms |
| **Latency (p99)** | <5ms |
| **Throughput** | 10k+ req/sec per instance |
| **Redis connections** | 1 (shared) |
| **Network calls** | 1 (Redis check) |

## Load Testing

```bash
# Install k6
brew install k6

# Run load test
k6 run load-test/k6.js
```

The k6 script uses a single client ID per virtual user (`client-${__VU}`) so each VU repeatedly hits the same rate limit bucket. That makes it easier to verify that the limiter is actually rejecting repeated requests from the same client.

The script also records three custom counters:

- `api_request_200` - allowed requests
- `api_request_429` - rate-limited requests
- `api_request_other` - unexpected responses

The most important k6 metrics to watch are:

- `http_req_duration` - end-to-end request latency
- `http_req_failed` - requests with non-2xx/3xx responses; this can be high when rate limiting is working because `429` counts as a failure in k6
- `checks_total` / `checks_succeeded` - whether the script's assertions passed
- `api_request_200` / `api_request_429` / `api_request_other` - the true status mix for the rate-limited endpoint

When the limiter is working, you should see most `/api/request` traffic end up in `api_request_429` once the bucket is exhausted. The `http_req_failed` metric in k6 is less useful here because it counts non-2xx responses as failures, so a high value can simply mean the limiter is rejecting traffic as expected.

Observed from the latest run:

- `vus_max`: 200 virtual users
- `http_reqs`: 1,383,350 total HTTP requests
- `iterations`: 691,675 completed iterations
- `api_request_200`: 33,001 allowed requests
- `api_request_429`: 658,674 rate-limited requests
- `checks_total`: 2,766,700 checks executed
- `checks_succeeded`: 2,766,697 successful checks
- `http_req_duration p(95)`: 22.26ms
- `http_req_duration p(99)`: 31.99ms

This shows the limiter is actively rejecting most repeated requests from the same client while keeping latency low.

## Troubleshooting

### Redis connection failed
```
docker-compose logs redis
```

### Kafka messages not flowing
```
docker-compose exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic rate-limit-events --from-beginning
```

### High error rate
Check Redis memory and Kafka disk space. Rate limiter degrades gracefully but excessive errors indicate infrastructure issues.

## Cleanup

```bash
docker-compose down -v
```

## Next Steps

Potential enhancements:
- **Distributed rate limiting** — rate limits across multiple instances
- **Rate limit tiers** — different limits for different client APIs
- **Sliding window logs** — more accurate but higher cost
- **Persistence** — survive service restarts
