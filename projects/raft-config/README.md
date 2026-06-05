# Raft Config Store

> What if feature flags and rate limits must survive leader death and stay consistent across the fleet?

A 3-node **distributed configuration store** backed by [HashiCorp Raft](https://github.com/hashicorp/raft). Writes go through the **leader** and replicate to a quorum; reads are served from the local state machine on any node (with the usual follower staleness tradeoff).

This project teaches the same primitives etcd uses internally—without building Raft from scratch first.

## Architecture

```
Client PUT/DELETE ──► HTTP (any node) ──► leader only ──► Raft log ──► quorum commit ──► FSM (KV map)
Client GET        ──► HTTP (any node) ──► local FSM read (may lag leader slightly)
```

- **Raft** — leader election, log replication, durable commit via BoltDB
- **FSM** — in-memory `map[string]string` applied from committed log entries
- **Snapshots** — periodic FSM snapshots to disk under `RAFT_DATA_DIR/snapshots`
- **Prometheus** — leader flag, term, commit index, read/write counters

## Stack

- Go + Gin
- HashiCorp Raft + raft-boltdb
- Docker Compose (3 nodes + Prometheus)
- Prometheus

## Quick Start

```bash
cd projects/raft-config
docker-compose up -d --build

# Wait for leader election (~5–15 seconds)
sleep 15

# Cluster status on each node
curl http://localhost:8081/api/cluster
curl http://localhost:8082/api/cluster
curl http://localhost:8083/api/cluster

# Write (must hit leader — retry on 503 with X-Raft-Leader-Addr)
curl -X PUT http://localhost:8081/api/config/rate_limit.default \
  -H "Content-Type: application/json" \
  -d '{"value":"100"}'

# Read from any node
curl http://localhost:8082/api/config/rate_limit.default
curl http://localhost:8083/api/config

# Metrics
curl http://localhost:8081/metrics

# Prometheus UI
# http://localhost:9090
```

## Endpoints

| Method | Path | Notes |
|--------|------|--------|
| GET | `/health` | Liveness + local Raft role |
| GET | `/api/cluster` | Leader, term, commit index |
| GET | `/api/config` | List all keys (local FSM) |
| GET | `/api/config/:key` | Get one key |
| PUT | `/api/config/:key` | `{"value":"..."}` — **leader only** |
| DELETE | `/api/config/:key` | **leader only** |
| GET | `/metrics` | Prometheus |

**Not leader (503):** response includes `leader_id`, `leader_addr`, and headers `X-Raft-Leader-Id`, `X-Raft-Leader-Addr`. Point writes at the leader (or retry until the request lands on it).

## Learning exercises

1. **Leader failover** — `docker stop raft-config-raft-node-1-1`, write via `:8082` or `:8083`, confirm new leader in `/api/cluster`.
2. **Replication lag** — write on leader, immediately read follower; observe when value appears (usually fast locally).
3. **Split exposure** — stop two of three nodes; writes should fail (no quorum); reads still return last local state on the survivor.
4. **Persistence** — write keys, `docker-compose down`, `up` again with same volumes; data should remain.

## Configuration (environment)

| Variable | Description |
|----------|-------------|
| `NODE_ID` | Raft server ID (`node-1`, …) |
| `RAFT_BIND` | TCP bind for Raft (`0.0.0.0:7000`) |
| `RAFT_ADVERTISE` | Address other nodes use (`raft-node-1:7000`) |
| `RAFT_DATA_DIR` | BoltDB + snapshots |
| `RAFT_BOOTSTRAP` | `true` only on first boot of a new cluster |
| `RAFT_PEERS` | `id=host:port,...` for initial `BootstrapCluster` |
| `HTTP_PORT` | Gin listen port |

## Metrics

```
raft_config_is_leader{node_id="..."}
raft_config_term{node_id="..."}
raft_config_commit_index{node_id="..."}
raft_config_writes_total
raft_config_reads_total
raft_config_apply_errors_total
raft_config_not_leader_total
```

## Ports (host)

| Node | HTTP | Raft |
|------|------|------|
| node-1 | 8081 | 7001 |
| node-2 | 8082 | 7002 |
| node-3 | 8083 | 7003 |

## What this is / isn't

**Is:** a small, strongly consistent control-plane KV (flags, limits, maintenance mode).

**Isn't:** a high-throughput event bus (use Kafka in sibling projects) or a place for large payloads.

## Next steps (repo roadmap)

- Linearizable reads (`read_index` / leader confirmation)
- SSE **watch** on key changes
- Wire **rate-limiting** to poll limits from this store
- Optional: toy hand-rolled Raft module for comparison
