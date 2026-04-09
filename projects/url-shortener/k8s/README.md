# Kubernetes Manifest Files - Complete Guide

This document explains the purpose and role of each Kubernetes manifest file in the `k8s/` directory.

---

## Directory Structure

```
k8s/
├── namespace.yaml                      # Phase 1: Namespace setup
├── app-configmap.yaml                  # Phase 1: App configuration
├── app-service.yaml                    # Phase 1: App service exposure
├── app-deployment.yaml                 # Phase 1: App deployment
├── redis-pvc.yaml                      # Phase 2: Redis persistent storage
├── redis-service.yaml                  # Phase 2: Redis network exposure
├── redis-deployment.yaml               # Phase 2: Redis container
├── dynamodb-pvc.yaml                   # Phase 3: DynamoDB persistent storage
├── dynamodb-service.yaml               # Phase 3: DynamoDB network exposure
├── dynamodb-deployment.yaml            # Phase 3: DynamoDB container
├── zookeeper-pvc.yaml                  # Phase 4: Zookeeper persistent storage
├── zookeeper-service.yaml              # Phase 4: Zookeeper network exposure
├── zookeeper-deployment.yaml           # Phase 4: Zookeeper container
├── kafka-pvc.yaml                      # Phase 4: Kafka persistent storage
├── kafka-service.yaml                  # Phase 4: Kafka network exposure
├── kafka-deployment.yaml               # Phase 4: Kafka container
├── prometheus-configmap.yaml           # Phase 5: Prometheus configuration
├── prometheus-pvc.yaml                 # Phase 5: Prometheus persistent storage
├── prometheus-service.yaml             # Phase 5: Prometheus network exposure
├── prometheus-deployment.yaml          # Phase 5: Prometheus container
├── grafana-datasources-configmap.yaml  # Phase 5: Grafana datasource config
├── grafana-pvc.yaml                    # Phase 5: Grafana persistent storage
├── grafana-service.yaml                # Phase 5: Grafana network exposure
└── grafana-deployment.yaml             # Phase 5: Grafana container
```

---

## Kubernetes Resource Types Explained

### ConfigMap
**Purpose**: Store configuration as key-value pairs (non-sensitive data)
- Configuration lives separately from container images
- Can be updated without rebuilding Docker images
- Environment variables are automatically loaded into containers
- Files: `app-configmap.yaml`, `prometheus-configmap.yaml`, `grafana-datasources-configmap.yaml`

### PersistentVolumeClaim (PVC)
**Purpose**: Request storage from the cluster
- Survives pod restarts (unlike emptyDir volumes)
- K8s automatically provisions a PersistentVolume to back it
- Multiple pods can potentially share the same PVC (depends on accessModes)
- Files: `*-pvc.yaml` (redis, dynamodb, kafka, prometheus, grafana)

### Service
**Purpose**: Enable network communication to pods
- Pods are ephemeral (can be deleted/recreated), but Services are stable
- Provides DNS-resolvable addresses (e.g., `redis:6379`, `kafka:9092`)
- Three main types:
  - `ClusterIP: None` (headless) - internal DNS only, no load balancing
  - `ClusterIP` (default) - internal only, load balanced
  - `LoadBalancer` - external access (used for app, prometheus, grafana)
- Files: `*-service.yaml` (app, redis, dynamodb, zookeeper, kafka, prometheus, grafana)

### Deployment
**Purpose**: Run containers in pods with desired state management
- Manages pod replicas, rolling updates, health monitoring
- Contains: image, ports, environment variables, volumes, health checks
- Automatically restarts failed pods
- Files: `*-deployment.yaml` (app, redis, dynamodb, zookeeper, kafka, prometheus, grafana)

### Namespace
**Purpose**: Logical isolation of resources within the cluster
- Group related resources together
- Enable multi-tenancy (dev, staging, prod namespaces)
- File: `namespace.yaml`

---

## Phase-by-Phase Breakdown

### Phase 1: Core Application

#### `namespace.yaml`
- Creates `url-shortener` namespace
- All resources in this phase (and subsequent phases) live in this namespace
- Allows easy cleanup: `kubectl delete namespace url-shortener` deletes everything

#### `app-configmap.yaml`
- Stores environment variables needed by the URL Shortener app
- Configuration keys:
  - Application settings: `PORT`, `METRICS_PORT`
  - DynamoDB: `AWS_REGION`, `DYNAMO_TABLE`, `DYNAMO_ENDPOINT`
  - Redis: `REDIS_ADDR`
  - Kafka: `KAFKA_BROKER`, `KAFKA_TOPIC`
- Referenced by `app-deployment.yaml` via `envFrom`

#### `app-service.yaml`
- Exposes the URL Shortener app outside the cluster
- `LoadBalancer` type makes it accessible from your machine
- Maps two ports:
  - 8080: Application API
  - 9090: Prometheus metrics endpoint
- DNS name inside cluster: `url-shortener:8080` and `url-shortener:9090`

#### `app-deployment.yaml`
- Deploys the actual URL Shortener container
- Key features:
  - `imagePullPolicy: Never` - uses local Docker image
  - `replicas: 1` - single instance (can scale to multiple)
  - Loads config from `app-configmap.yaml`
  - Hardcoded AWS credentials (valid for local DynamoDB testing)
  - **Liveness probe**: Restarts pod if `/health` endpoint fails
  - **Readiness probe**: Removes pod from service if unhealthy (for gradual shutdown)
  - Resource requests/limits prevent resource hogging

---

### Phase 2: Redis (In-Memory Cache)

#### `redis-pvc.yaml`
- Requests 1Gi persistent storage for Redis data
- K8s creates storage automatically (implementation depends on storage class)

#### `redis-service.yaml`
- Exposes Redis at `redis:6379` within the cluster
- `clusterIP: None` - internal only, no external access
- Allows app to connect via: `redis.url-shortener.svc.cluster.local:6379` (or just `redis:6379`)

#### `redis-deployment.yaml`
- Runs Redis 7 Alpine (minimal image)
- Mounts PVC at `/data` for persistence
- **Liveness probe**: Uses `redis-cli ping` to verify Redis responds to commands
- **Readiness probe**: Similar, but used for gradual shutdown during updates
- Lower resource limits (50m CPU, 64Mi memory) - Redis is lightweight

---

### Phase 3: DynamoDB (NoSQL Database)

#### `dynamodb-pvc.yaml`
- Requests 2Gi storage for DynamoDB data
- Larger than Redis since databases store more data

#### `dynamodb-service.yaml`
- Exposes DynamoDB at `dynamodb-local:8000` within the cluster
- Port 8000 is the standard DynamoDB local port

#### `dynamodb-deployment.yaml`
- Runs AWS DynamoDB Local image
- Command: `-dbPath /data` stores data on the PVC (instead of in-memory)
- **Health checks**: HTTP GET to verify service is responding
- Larger resource limits (100m CPU, 256Mi memory) - DynamoDB uses more resources

---

### Phase 4: Message Queue (Kafka + Zookeeper)

#### Zookeeper Files

**`zookeeper-pvc.yaml`**: 1Gi storage for Zookeeper coordination data

**`zookeeper-service.yaml`**: Exposes Zookeeper at `zookeeper:2181` for Kafka

**`zookeeper-deployment.yaml`**:
- Zookeeper is the distributed coordination service for Kafka
- **Important**: Uses `emptyDir` volumes (temporary, pod-lifetime storage)
  - Zookeeper just coordinates brokers; losing its data isn't critical for local testing
  - In production, you'd use PVCs
- **Health check**: `echo ruok` (Zookeeper admin command) checks if responsive
- Kafka won't start until Zookeeper is ready

#### Kafka Files

**`kafka-pvc.yaml`**: 2Gi storage for Kafka message logs

**`kafka-service.yaml`**: Exposes Kafka at `kafka:9092` for producers/consumers

**`kafka-deployment.yaml`**:
- Kafka broker for pub/sub messaging
- Key environment variables:
  - `KAFKA_BROKER_ID: "1"` - unique identifier in cluster
  - `KAFKA_ZOOKEEPER_CONNECT: "zookeeper:2181"` - registers with Zookeeper
  - `KAFKA_ADVERTISED_LISTENERS: "PLAINTEXT://kafka:9092"` - tells clients how to reach this broker
  - `KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: "1"` - replication for distributed setup
- **Health check**: `kafka-broker-api-versions` verifies broker is healthy
- **Critical**: Must wait for Zookeeper to be ready before deploying

---

### Phase 5: Monitoring (Prometheus + Grafana)

#### Prometheus Files

**`prometheus-configmap.yaml`**:
- Configuration defining what metrics to scrape
- **Scrape targets**:
  - `url-shortener:9090` - your app's metrics endpoint
  - `localhost:9090` - Prometheus internal metrics
- **Scrape interval**: 15 seconds (how often to collect metrics)
- Mounted at `/etc/prometheus/prometheus.yml` in the container

**`prometheus-pvc.yaml`**: 5Gi storage for time-series data
- Larger storage since metrics accumulate over time

**`prometheus-service.yaml`**:
- Exposes Prometheus at `prometheus:9090`
- `LoadBalancer` type allows external access
- Access UI at `http://localhost:9090` (via port-forward)

**`prometheus-deployment.yaml`**:
- Time-series database for metrics
- Mounts ConfigMap for configuration
- Stores metrics at `/prometheus` on PVC
- **Health checks**: Uses Prometheus endpoints `/-/healthy` and `/-/ready`

#### Grafana Files

**`grafana-datasources-configmap.yaml`**:
- Automatically configures Grafana to use Prometheus as datasource
- Mounted at `/etc/grafana/provisioning/datasources/prometheus.yaml`
- Grafana discovers and queries metrics from Prometheus using this config

**`grafana-pvc.yaml`**: 1Gi storage for dashboards, alerts, user data

**`grafana-service.yaml`**:
- Exposes Grafana at `grafana:3000`
- `LoadBalancer` type for external access
- Access UI at `http://localhost:3000` (via port-forward)

**`grafana-deployment.yaml`**:
- Visualization and dashboard platform
- `GF_SECURITY_ADMIN_PASSWORD: "admin"` - sets login credentials (admin/admin)
- Mounts datasource config so Prometheus is available immediately
- Stores dashboards and user data on PVC
- **Health checks**: Uses `/api/health` endpoint

---

## How They Work Together

```
┌─────────────────────────────────────────────────────────────────┐
│                     url-shortener Namespace                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────┐         ┌──────────────────┐                   │
│  │   Your App  │         │ ConfigMap        │                   │
│  │ (Deployment)├────────→│ • PORT=8080      │                   │
│  └─────────────┘         │ • KAFKA_BROKER   │                   │
│        ↓                  │ • REDIS_ADDR     │                   │
│   Service (Port 8080)     └──────────────────┘                   │
│        │                                                          │
│        ├──→ Redis (Caching)                                      │
│        │    • Service: redis:6379                               │
│        │    • Deployment + PVC                                  │
│        │                                                          │
│        ├──→ DynamoDB (Database)                                 │
│        │    • Service: dynamodb-local:8000                      │
│        │    • Deployment + PVC                                  │
│        │                                                          │
│        └──→ Kafka (Message Queue)                               │
│             • Service: kafka:9092                               │
│             • Zookeeper: zookeeper:2181 (coordinator)           │
│             • Both: Deployment + PVC                            │
│                                                                   │
│  ┌─────────────────────────────────────────┐                    │
│  │ Monitoring Stack                        │                    │
│  ├─────────────────────────────────────────┤                    │
│  │ Prometheus (tsdb for metrics)           │                    │
│  │ • Scrapes from app:9090 every 15s       │                    │
│  │ • Stores time-series data (PVC: 5Gi)    │                    │
│  │ • Accessible: prometheus:9090           │                    │
│  │                                         │                    │
│  │ Grafana (visualization)                 │                    │
│  │ • Reads metrics from Prometheus         │                    │
│  │ • Creates dashboards                    │                    │
│  │ • Accessible: grafana:3000              │                    │
│  └─────────────────────────────────────────┘                    │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Deployment Order

1. **Phase 1**: Namespace → App ConfigMap → App Service → App Deployment
2. **Phase 2**: Redis PVC → Redis Service → Redis Deployment
3. **Phase 3**: DynamoDB PVC → DynamoDB Service → DynamoDB Deployment
4. **Phase 4**: 
   - Zookeeper PVC → Zookeeper Service → Zookeeper Deployment (wait for ready!)
   - Kafka PVC → Kafka Service → Kafka Deployment
5. **Phase 5**: 
   - Prometheus ConfigMap → Prometheus PVC → Prometheus Service → Prometheus Deployment
   - Grafana Datasources ConfigMap → Grafana PVC → Grafana Service → Grafana Deployment

---

## Common kubectl Commands

```bash
# Deploy everything at once
kubectl apply -f k8s/

# Deploy a specific phase
kubectl apply -f k8s/app-*.yaml
kubectl apply -f k8s/redis-*.yaml

# Check deployment status
kubectl get deployments -n url-shortener
kubectl get services -n url-shortener
kubectl get pods -n url-shortener

# View logs
kubectl logs deployment/url-shortener -n url-shortener -f
kubectl logs deployment/redis -n url-shortener

# Port forward for external access
kubectl port-forward svc/url-shortener 8083:8080 -n url-shortener
kubectl port-forward svc/prometheus 9090:9090 -n url-shortener
kubectl port-forward svc/grafana 3000:3000 -n url-shortener

# Delete everything
kubectl delete namespace url-shortener
```

---

## File Naming Conventions

- `*-configmap.yaml` - Configuration files (ConfigMap resources)
- `*-pvc.yaml` - Persistent storage claims (PersistentVolumeClaim resources)
- `*-service.yaml` - Network exposure (Service resources)
- `*-deployment.yaml` - Running containers (Deployment resources)

This naming makes it easy to understand what each file does at a glance!
