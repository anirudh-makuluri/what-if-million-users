# what-if-million-users

A collection of backend systems built with one question in mind:

> **"What does this look like when a million users show up?"**

Most tutorials build the happy path. This repo builds the production path — caching, async processing, observability, fault tolerance, and scale-aware design, from day one.

Each project is self-contained. Pick any one, run it locally, and see how the architecture answers the scale question.

---

## Projects

### [rate-limiting](./projects/rate-limiting)
> What if you need to protect APIs from abuse at scale?

A Go API rate limiter where Redis enforces token-bucket limits with an atomic Lua script, Kafka captures async event logs, and Prometheus exposes real-time metrics. It protects APIs by allowing bursts, refilling tokens over time, and rejecting requests when a client exceeds its budget.

### [url-shortener](./projects/url-shortener)
> What if a simple redirect needs to handle millions of clicks?

A URL shortener where every architectural decision is made with traffic in mind. Redis sits in front of DynamoDB so the database never gets hit twice for the same short code. Every redirect publishes an async Kafka event so analytics never slow down the user. Prometheus tracks cache hits, misses, and latency in real time.

## Shared Stack

Most projects in this repo use the same production-ready base:

- Go
- Gin
- Redis
- Kafka
- Prometheus
- Docker Compose

---

## Philosophy

Every project in this repo follows the same principles:

**Cache aggressively** — the fastest request is one that never touches the database. Every project has a caching layer designed before the storage layer.

**Never block the hot path** — analytics, logging, and side effects happen asynchronously. The user gets their response first, everything else follows.

**Fail gracefully** — a Redis outage should not take down the app. A Kafka failure should not break a redirect. Dependencies fail independently.

**Measure everything** — if it is not in Prometheus, it did not happen. Every project ships with metrics from day one, not as an afterthought.

**Run locally, think globally** — every project runs with a single `docker-compose up`. The local setup mirrors what a production deployment would look like.

---

## Running Any Project

Each project has its own `docker-compose.yml` and `README.md`. Navigate into the project folder and follow its setup guide.

```bash
cd url-shortener
docker-compose up -d --build
```

---

## What Is Coming Next

| Project | Question |
|---|---|
| `job-queue` | What if background tasks need to survive crashes and retries? |
| `distributed-cache` | What if cache invalidation needs to work across regions? |
| `leaderboard` | What if millions of users are updating scores simultaneously? |

---

## Author

[Anirudh Makuluri](https://github.com/anirudh-makuluri)
