# Leaderboard

> What if millions of users are updating scores simultaneously?

A Spring Boot leaderboard with Redis-backed ranking, MSSQL durability, optimistic write retries, and cache hydration so rankings can be rebuilt after Redis loss without dropping player state.

## Architecture

- **Redis ranking hot path** — scores live in a sorted set so top-N and around-player reads stay fast
- **MSSQL durability** — each player's canonical score and profile metadata are stored in SQL Server
- **Highest-score wins** — lower submissions do not overwrite an existing personal best
- **Optimistic retries** — concurrent updates retry against MSSQL version conflicts before surfacing a write conflict
- **Cache hydration** — startup and scheduled audits repopulate Redis from MSSQL if the cache falls behind or restarts

## Stack

- **Java 21**
- **Spring Boot 3.5**
- **Spring Data JPA**
- **Spring Data Redis**
- **Microsoft SQL Server**
- **Docker Compose**

## Why Redis + MSSQL

The leaderboard needs two different strengths. Redis handles the ranking hot path because sorted sets make rank lookups and top-N queries cheap even under high write volume. SQL Server acts as the durable source of truth so score state survives cache restarts, supports audit-friendly persistence, and mirrors the kind of enterprise backend stack where operational data already lives in MSSQL.

## Quick Start

```bash
cd projects/leaderboard
docker compose up -d --build
```

The API will be available at `http://localhost:8085`.

## Endpoints

### POST /leaderboard/scores

Submit a player's latest score. The service keeps the player's best score.

```bash
curl -X POST http://localhost:8085/leaderboard/scores \
  -H "Content-Type: application/json" \
  -d '{
    "playerId": "player-123",
    "displayName": "SkylineRunner",
    "score": 9800
  }'
```

### GET /leaderboard/players/{playerId}

Fetch a player's current score and rank.

```bash
curl http://localhost:8085/leaderboard/players/player-123
```

### GET /leaderboard/top?limit=10

Fetch the current top players.

```bash
curl "http://localhost:8085/leaderboard/top?limit=10"
```

### GET /leaderboard/players/{playerId}/neighbors?before=2&after=2

Fetch a rank window around a specific player.

```bash
curl "http://localhost:8085/leaderboard/players/player-123/neighbors?before=2&after=2"
```

## Ranking Rules

- Higher score ranks above lower score
- If two players have the same score, `playerId` breaks the tie deterministically
- Submitting a lower score keeps the existing personal best but still refreshes profile metadata like `displayName`

## Recovery Behavior

- Redis cache is rebuilt from MSSQL on startup
- A scheduled cache audit compares Redis cardinality against MSSQL row count and rehydrates if Redis falls behind
- If Redis is temporarily unavailable during reads, the service falls back to MSSQL for top, player, and around-player lookups

## Health Check

Spring Boot actuator health is exposed at:

```bash
curl http://localhost:8085/actuator/health
```

## Cleanup

```bash
docker compose down -v
```
