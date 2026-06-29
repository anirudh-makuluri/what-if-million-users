# Job Queue

> What if background tasks need to survive crashes and retries?

A Spring Boot job queue with Redis-backed queue state, MSSQL-backed durability, worker leases, retry backoff, job tracking, and automatic recovery for stuck jobs.

## Architecture

- **MSSQL durability** — every job is stored with status, retry counts, processor ownership, and timestamps
- **Redis queue state** — queued jobs live in a ready sorted set and claimed jobs move into a processing sorted set with lease expirations
- **Worker leases** — claiming a job creates a time-bounded processing lock so crashed workers do not hold jobs forever
- **Retry backoff** — failed jobs re-enter the queue with exponential backoff until the retry budget is exhausted
- **Failure recovery** — a scheduled recovery loop detects expired processing leases and re-queues jobs automatically

## Stack

- **Java 21**
- **Spring Boot 3.5**
- **Spring Data JPA**
- **Spring Data Redis**
- **Microsoft SQL Server**
- **Docker Compose**

## Why MSSQL

We used MSSQL over Postgres because this job queue is modeled like an enterprise internal automation system, where Redis handles the hot-path queue coordination and the relational database mainly acts as the durable system of record for job history, retries, failure recovery, and auditability. In that kind of Microsoft-heavy environment, SQL Server is a realistic choice because teams often already depend on it for operational tooling, governance, backups, and long-term support.

## Quick Start

```bash
cd projects/job-queue
docker compose up -d --build
```

The API will be available at `http://localhost:8084`.

## Endpoints

### POST /jobs

Create a new queued job.

```bash
curl -X POST http://localhost:8084/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "payload": "send welcome email to user-123",
    "maxRetries": 3,
    "delaySeconds": 0
  }'
```

### POST /jobs/{id}/claim

Claim a queued job for a worker and start a processing lease.

```bash
curl -X POST http://localhost:8084/jobs/<job-id>/claim \
  -H "Content-Type: application/json" \
  -d '{
    "processorId": "worker-a",
    "leaseSeconds": 60
  }'
```

### POST /jobs/{id}/complete

Mark a claimed job as completed.

```bash
curl -X POST http://localhost:8084/jobs/<job-id>/complete \
  -H "Content-Type: application/json" \
  -d '{
    "processorId": "worker-a"
  }'
```

### POST /jobs/{id}/fail

Fail a claimed job and optionally retry it.

```bash
curl -X POST http://localhost:8084/jobs/<job-id>/fail \
  -H "Content-Type: application/json" \
  -d '{
    "processorId": "worker-a",
    "errorMessage": "SMTP provider timed out",
    "retryable": true
  }'
```

### GET /jobs/{id}

Fetch the current state of a job.

```bash
curl http://localhost:8084/jobs/<job-id>
```

## State Model

- `QUEUED` — ready now or delayed until `availableAt`
- `PROCESSING` — claimed by a worker until `lockExpiresAt`
- `COMPLETED` — finished successfully
- `FAILED` — retry budget exhausted or explicitly failed without retry

## Recovery Behavior

- Jobs are retried with exponential backoff starting at `15s`
- If a worker crashes and the lease expires, the scheduler re-queues the job and counts that expiration as a failed attempt
- On service startup, queued jobs are re-hydrated from MSSQL back into Redis so a Redis restart does not lose the queue

## Health Check

Spring Boot actuator health is exposed at:

```bash
curl http://localhost:8084/actuator/health
```

## Cleanup

```bash
docker compose down -v
```
