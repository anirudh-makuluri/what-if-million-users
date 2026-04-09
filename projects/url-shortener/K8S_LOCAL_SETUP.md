# Switching to Kubernetes - Local Setup Guide

This guide walks you through migrating from Docker Compose to Kubernetes locally using either **Docker Desktop** or **Minikube**.

## Prerequisites

- Docker Desktop or Minikube already installed ✓
- `kubectl` CLI installed (comes with Docker Desktop, or install via `minikube`)
- Git Bash or PowerShell

---

## Option 1: Using Docker Desktop (Recommended for Windows)

### Step 1: Enable Kubernetes in Docker Desktop

1. Open **Docker Desktop**
2. Go to **Settings** → **Kubernetes**
3. Check **Enable Kubernetes**
4. Click **Apply & Restart**
5. Wait for Kubernetes to start (can take 2-3 minutes)

### Step 2: Verify kubectl access

```bash
kubectl cluster-info
kubectl get nodes
```

You should see output showing your local cluster is running.

---

## Option 2: Using Minikube

### Step 1: Start Minikube

```bash
# Start minikube with enough resources
minikube start --cpus=4 --memory=8192 --disk-size=20gb

# Verify it's running
minikube status
```

### Step 2: Configure kubectl to use Minikube

```bash
# Automatically done by minikube, but verify:
kubectl config current-context
# Should output: minikube
```

### Step 3: (Optional) Use Minikube's Docker daemon

If you want to build images directly in Minikube:

```bash
eval $(minikube docker-env)
```

---

## Deploying to Kubernetes

### Step 1: Build and push the Docker image

**For Docker Desktop K8s:**
```bash
cd projects/url-shortener
docker build -t url-shortener:latest .
```

**For Minikube:**
```bash
cd projects/url-shortener

# Option A: Build and push to registry (requires registry)
docker build -t url-shortener:latest .

# Option B: Build directly in Minikube's Docker daemon
eval $(minikube docker-env)
docker build -t url-shortener:latest .
# Then set imagePullPolicy: Never in the Deployment manifest
```

### Step 2: Create Kubernetes namespace

```bash
kubectl create namespace url-shortener
kubectl config set-context --current --namespace=url-shortener
```

### Step 3: Apply Kubernetes manifests

```bash
# From the project root
kubectl apply -f k8s/

# Or apply specific files in order:
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/pvc/
kubectl apply -f k8s/zookeeper/
kubectl apply -f k8s/kafka/
kubectl apply -f k8s/redis/
kubectl apply -f k8s/dynamodb/
kubectl apply -f k8s/app/
kubectl apply -f k8s/prometheus/
kubectl apply -f k8s/grafana/
```

### Step 4: Wait for all pods to be ready

```bash
# Watch pod status
kubectl get pods -w

# Wait for all pods in Running state
kubectl wait --for=condition=Ready pod --all --timeout=300s
```

---

## Accessing Services Locally

### Using port-forward (easiest method)

```bash
# URL Shortener API (http://localhost:8083)
kubectl port-forward svc/url-shortener 8083:8080 &

# Prometheus (http://localhost:9090)
kubectl port-forward svc/prometheus 9090:9090 &

# Grafana (http://localhost:3000)
kubectl port-forward svc/grafana 3000:3000 &

# Kafka UI (if deployed)
kubectl port-forward svc/kafka-ui 8090:8080 &

# DynamoDB Admin (if deployed)
kubectl port-forward svc/dynamodb-admin 8001:8001 &

# Redis Commander (optional add-on)
kubectl port-forward svc/redis 6379:6379 &
```

### Using Minikube service command (Minikube only)

```bash
# Automatically opens service in browser
minikube service url-shortener -n url-shortener

# Or get the URL manually
minikube service url-shortener -n url-shortener --url
```

### Using Ingress (production-like, local testing)

```bash
# Enable ingress addon (Minikube)
minikube addons enable ingress

# Or install ingress controller (Docker Desktop K8s)
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml

# Then access via:
# http://url-shortener.local (requires /etc/hosts entry for local setup)
```

---

## Useful kubectl Commands

```bash
# View all resources
kubectl get all -n url-shortener

# View pod logs
kubectl logs deployment/url-shortener -n url-shortener -f

# View specific pod details
kubectl describe pod <pod-name> -n url-shortener

# Execute command in pod
kubectl exec -it <pod-name> -n url-shortener -- /bin/sh

# Delete entire deployment
kubectl delete namespace url-shortener

# Scale deployment
kubectl scale deployment url-shortener --replicas=3 -n url-shortener

# Get resource usage
kubectl top pods -n url-shortener
kubectl top nodes
```

---

## Testing the Application

```bash
# Shorten a URL
curl -X POST http://localhost:8083/shorten \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://www.example.com"}'

# Redirect to shortened URL
curl -L http://localhost:8083/abc123

# Health check
curl http://localhost:8083/health

# Prometheus metrics
curl http://localhost:9090/api/v1/query?query=url_shortener_urls_created_total
```

---

## Troubleshooting

### Pod stuck in `Pending` state
```bash
kubectl describe pod <pod-name> -n url-shortener
# Check for resource constraints, missing PVCs, or image pull errors
```

### ImagePullBackOff error
- Ensure image exists: `docker images | grep url-shortener`
- For Minikube, use `imagePullPolicy: Never` in manifest
- For Docker Desktop K8s, image should be available in local Docker

### Connection refused errors
- Verify services are running: `kubectl get svc -n url-shortener`
- Check port-forward is active: `ps aux | grep port-forward`
- Restart port-forward if needed

### DNS resolution issues within pods
```bash
# Test DNS from within a pod
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup redis.url-shortener.svc.cluster.local
```

### Scale DynamoDB/Kafka issues
- Ensure PVCs are provisioned: `kubectl get pvc -n url-shortener`
- Check storage class: `kubectl get storageclass`
- For local K8s, use `hostPath` storage or emptyDir

---

## Switching Back to Docker Compose

If you need to return to Docker Compose:

```bash
# Stop Kubernetes (Docker Desktop)
# Settings → Kubernetes → uncheck Enable Kubernetes

# Or stop Minikube
minikube stop
```

Then run:
```bash
cd projects/url-shortener
docker-compose up -d
```

---

## Next Steps

1. Create Kubernetes manifests in `k8s/` directory
2. Test deployment locally
3. Add Kustomize overlays for dev/staging/prod configurations
4. Set up CI/CD pipeline to build and deploy automatically
5. Document environment-specific configurations

---

## Resources

- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Docker Desktop Kubernetes Docs](https://docs.docker.com/desktop/kubernetes/)
- [Minikube Docs](https://minikube.sigs.k8s.io/)
- [Kubernetes Official Docs](https://kubernetes.io/docs/)
