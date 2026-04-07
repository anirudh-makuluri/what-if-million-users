# URL Shortener — Overengineered Edition

A production-grade URL shortener built with Go, featuring Redis caching, DynamoDB persistence, Kafka analytics streaming, and Prometheus observability. Runs fully locally via Docker Compose.

---

## Architecture

```
HTTP Request
    └── Gin Router
            └── Handler
                    ├── Redis Cache        (cache hit → redirect immediately)
                    ├── DynamoDB           (cache miss → fetch from DB → populate cache)
                    └── Kafka Producer     (async analytics event on every redirect)

Prometheus scrapes /metrics every 15s
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| HTTP Framework | Gin |
| Cache | Redis 7 |
| Database | DynamoDB (local) |
| Event Streaming | Kafka + Zookeeper |
| Observability | Prometheus |
| Language | Go 1.22 |
| Infrastructure | Docker Compose |

---

## Project Structure

```
url-shortener/
├── cmd/
│   └── main.go                  # Entry point, bootstraps server
├── internal/
│   ├── handler/
│   │   └── handler.go           # HTTP handlers, redirect logic
│   ├── store/
│   │   └── dynamodb.go          # DynamoDB read/write
│   ├── cache/
│   │   └── redis.go             # Redis get/set with TTL
│   ├── kafka/
│   │   └── producer.go          # Kafka async event publisher
│   └── metrics/
│       └── metrics.go           # Prometheus counters and histograms
├── Dockerfile                   # Multi-stage build
├── docker-compose.yml           # Full local environment
├── prometheus.yml               # Prometheus scrape config
├── item.json                    # Sample DynamoDB seed item
└── go.mod / go.sum
```

---

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `PORT` | App HTTP port | `8080` |
| `METRICS_PORT` | Prometheus metrics port | `9090` |
| `AWS_REGION` | AWS region | `us-east-1` |
| `AWS_ACCESS_KEY_ID` | AWS key (dummy for local) | `dummy` |
| `AWS_SECRET_ACCESS_KEY` | AWS secret (dummy for local) | `dummy` |
| `DYNAMO_ENDPOINT` | DynamoDB endpoint | `http://dynamodb-local:8000` |
| `DYNAMO_TABLE` | DynamoDB table name | `urls` |
| `REDIS_ADDR` | Redis address | `redis:6379` |
| `KAFKA_BROKER` | Kafka broker address | `kafka:9092` |
| `KAFKA_TOPIC` | Kafka topic name | `redirect-events` |

---

## Getting Started

### Prerequisites

- Docker Desktop
- AWS CLI (for seeding DynamoDB)
- Go 1.22+ (for local development)

---

### 1. Clone and navigate to the project

```bash
cd url-shortener
```

### 2. Create prometheus.yml in project root

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: url-shortener
    static_configs:
      - targets: ["app:9090"]
```

### 3. Start all containers

```powershell
docker-compose up -d --build
```

### 4. Verify all containers are running

```powershell
docker-compose ps
```

Expected services:
- `app` — Go HTTP server
- `dynamodb-local` — Local DynamoDB
- `redis` — Redis cache
- `zookeeper` — Kafka dependency
- `kafka` — Kafka broker
- `prometheus` — Metrics scraper

---

### 5. Create DynamoDB table

```powershell
aws dynamodb create-table `
  --table-name urls `
  --attribute-definitions AttributeName=short_code,AttributeType=S `
  --key-schema AttributeName=short_code,KeyType=HASH `
  --billing-mode PAY_PER_REQUEST `
  --endpoint-url http://localhost:8000
```

### 6. Seed a test record

Create `item.json`:

```json
{
  "short_code": {"S": "abc123"},
  "long_url": {"S": "https://google.com"}
}
```

Then run:

```powershell
aws dynamodb put-item `
  --table-name urls `
  --item file://item.json `
  --endpoint-url http://localhost:8000
```

### 7. Create Kafka topic

```powershell
docker-compose exec kafka kafka-topics --create `
  --topic redirect-events `
  --bootstrap-server kafka:9092 `
  --partitions 3 `
  --replication-factor 1
```

---

## Testing the Flow

### Create a short URL

```powershell
Invoke-RestMethod `
  -Method POST `
  -Uri "http://localhost:8083/shorten" `
  -ContentType "application/json" `
  -Body '{"short_code":"abc123","long_url":"https://google.com"}'
```

Expected:

```json
{"message":"short URL created successfully"}
```

### Hit the redirect endpoint

```powershell
curl.exe -v "http://localhost:8083/abc123"
```

Expected: `301 Moved Permanently` → `https://google.com`

### Hit it again to verify Redis cache

```powershell
curl.exe -v "http://localhost:8083/abc123"
```

Same response but served from Redis this time — no DynamoDB call.

### Health check

```powershell
Invoke-RestMethod -Uri "http://localhost:8083/health"
```

Expected: `{"status":"ok"}`

---

## Observability

### View raw metrics

```powershell
curl.exe "http://localhost:9090/metrics"
```

### Prometheus UI

Open `http://localhost:9091` in your browser.

Useful queries:

```
url_shortener_cache_hits_total
url_shortener_cache_misses_total
url_shortener_cache_errors_total
url_shortener_dynamo_errors_total
url_shortener_kafka_errors_total
url_shortener_request_duration_seconds
```

> Prometheus scrapes every 15 seconds — wait briefly after hitting endpoints before querying.

### Check Prometheus scrape targets

Open `http://localhost:9091/targets` — the `url-shortener` job should show `UP`.

---

## Load Testing

### Run k6 against the local app

This script creates one short URL in `setup()` and then load-tests the redirect path without following the external redirect target.

```powershell
k6 run .\load-test\k6.js
```

Optional overrides:

```powershell
$env:BASE_URL="http://localhost:8083"
$env:VUS="25"
$env:DURATION="1m"
k6 run .\load-test\k6.js
```

Current local reference result:

- `1250` max VUs over a `55s` staged run
- `0.00%` failed requests
- `p95 = 122.14ms`
- `3759.92 req/s`
- `0` duplicate `409` responses

---

## Kafka Debugging

### List topics

```powershell
docker-compose exec kafka kafka-topics --list `
  --bootstrap-server kafka:9092
```

### Consume messages from the topic

```powershell
docker-compose exec kafka kafka-console-consumer `
  --topic redirect-events `
  --bootstrap-server kafka:9092 `
  --from-beginning
```

Then hit the redirect endpoint in another terminal and watch events appear:

```json
{
  "short_code": "abc123",
  "long_url": "https://google.com",
  "timestamp": "2026-04-03T10:08:00Z",
  "user_agent": "curl/7.88.1",
  "ip_address": "172.18.0.1"
}
```

---

## Useful Docker Commands

```powershell
# Stream all container logs
docker-compose logs -f

# Stream app logs only
docker-compose logs -f app

# Stop all containers
docker-compose down

# Stop and wipe all volumes (fresh state)
docker-compose down -v

# Rebuild and restart
docker-compose up -d --build
```

---

## How the Redirect Flow Works

```
1. Request arrives: GET /:shortCode

2. Check Redis
   ├── HIT  → increment CacheHits
   │         → publish Kafka event (async)
   │         → 301 redirect
   │
   └── MISS → increment CacheMisses
             → query DynamoDB
             ├── Not found → 404
             └── Found     → write to Redis (TTL 24h)
                           → publish Kafka event (async)
                           → 301 redirect
```

---

## Key Design Decisions

**Redis returns `(string, bool, error)`** — a bool signals cache hit/miss cleanly for primitive string values. A pointer (`*string`) would also work but requires dereferencing on every use.

**DynamoDB returns `(*URLRecord, error)`** — a nil pointer is the idiomatic Go way to say "record not found" for structs.

**Kafka uses `Async: true`** — the redirect never blocks waiting for Kafka acknowledgement. Analytics loss is acceptable; redirect latency is not.

**`dynamodbav` and `json` struct tags** — control the serialized key names across storage boundaries. DynamoDB stores attributes as a map, Kafka sends JSON bytes. Without tags, Go's PascalCase field names would leak into storage.

**Metrics on port 9090, app on 8080** — the `/metrics` endpoint should never be publicly exposed. Prometheus scrapes it on the internal Docker network only.

**Multi-stage Dockerfile** — builder stage uses `golang:1.22-alpine` (~300MB), final image uses `alpine:3.19` (~7MB). `CGO_ENABLED=0` produces a fully static binary with no external dependencies.
