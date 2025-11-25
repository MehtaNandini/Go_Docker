# Kubernetes Deployment for TODO App

This directory contains Kubernetes manifests for deploying the TODO application stack to a local kind cluster.

## 📁 Structure

```
k8s/
├── namespace.yaml            # Namespace definition
├── postgres.yaml             # Main PostgreSQL database
├── postgres-airflow.yaml     # Airflow PostgreSQL database
├── api.yaml                  # Go TODO API service
├── ml.yaml                   # Python ML scoring service
├── mlflow.yaml              # MLflow tracking server
├── airflow.yaml             # Airflow (webserver, scheduler, init job)
├── deploy.sh                # Automated deployment script
├── cleanup.sh               # Cleanup script
├── DEPLOYMENT.md            # Detailed deployment guide
├── QUICK_REFERENCE.md       # Quick command reference
└── README.md                # This file
```

## 🚀 Quick Start

### Prerequisites

- Docker
- kind (Kubernetes in Docker)
- kubectl

Install on macOS:
```bash
brew install kind kubectl
```

### Deploy

**Option 1: Automated (Recommended)**
```bash
cd k8s
./deploy.sh
```

**Option 2: Manual**
```bash
# Build images
docker build -t todoapp:latest .
docker build -t todo-ml:latest ./ml_service
docker build -t todo-mlflow:latest ./mlflow_server

# Create cluster
kind create cluster --name todo-cluster --config k8s/kind-config.yaml

# Load images
kind load docker-image todoapp:latest --name todo-cluster
kind load docker-image todo-ml:latest --name todo-cluster
kind load docker-image todo-mlflow:latest --name todo-cluster

# Apply manifests
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/postgres-airflow.yaml
kubectl apply -f k8s/ml.yaml
kubectl apply -f k8s/api.yaml
kubectl apply -f k8s/mlflow.yaml
kubectl apply -f k8s/airflow.yaml
```

### Access Services

- **TODO API**: http://localhost:30080
- **Airflow UI**: http://localhost:30082 (admin/admin)
- **MLflow UI**: http://localhost:30500

### Cleanup

```bash
./cleanup.sh
```

## 📊 Architecture

### Components

1. **Postgres** (ClusterIP)
   - Main database for TODO app
   - PVC: 1Gi
   - Service: `postgres:5432`

2. **Postgres-Airflow** (ClusterIP)
   - Database for Airflow metadata
   - PVC: 2Gi
   - Service: `postgres-airflow:5432`

3. **ML Service** (ClusterIP)
   - Python ML scoring service
   - Service: `ml:8081`

4. **TODO API** (ClusterIP + NodePort)
   - Go REST API
   - 2 replicas for HA
   - Internal: `todo-api:8080`
   - External: `localhost:30080`

5. **MLflow** (ClusterIP + NodePort)
   - ML experiment tracking
   - PVC: 5Gi
   - Internal: `mlflow:5000`
   - External: `localhost:30500`

6. **Airflow** (ClusterIP + NodePort)
   - Webserver: `airflow-webserver:8080` → `localhost:30082`
   - Scheduler: runs in background
   - Init Job: one-time DB setup

### Network Architecture

```
┌─────────────────────────────────────────────────────┐
│                  kind Cluster                        │
│                                                      │
│  ┌────────────────────────────────────────────┐    │
│  │         Namespace: todo-app                 │    │
│  │                                             │    │
│  │  ┌──────────┐      ┌──────────┐           │    │
│  │  │ Postgres │◄─────┤ TODO API │           │    │
│  │  └──────────┘      └─────┬────┘           │    │
│  │                           │                │    │
│  │  ┌──────────┐            │                │    │
│  │  │    ML    │◄───────────┘                │    │
│  │  └──────────┘                              │    │
│  │                                             │    │
│  │  ┌──────────┐      ┌─────────────┐        │    │
│  │  │ Postgres │◄─────┤   Airflow   │        │    │
│  │  │ Airflow  │      │ (Web+Sched) │        │    │
│  │  └──────────┘      └──────┬──────┘        │    │
│  │                           │                │    │
│  │  ┌──────────┐            │                │    │
│  │  │  MLflow  │◄───────────┘                │    │
│  │  └──────────┘                              │    │
│  └────────────────────────────────────────────┘    │
│                                                      │
│  NodePorts:                                         │
│  30080 → TODO API                                   │
│  30082 → Airflow UI                                 │
│  30500 → MLflow UI                                  │
└─────────────────────────────────────────────────────┘
         ▲         ▲         ▲
         │         │         │
    localhost:30080 │    localhost:30500
              localhost:30082
```

## 🔄 Key Differences from Docker Compose

### Service Discovery
- **Docker Compose**: Uses service names directly
- **Kubernetes**: Uses DNS `<service>.<namespace>.svc.cluster.local`
  - Within same namespace: just `<service>` (e.g., `postgres`)

### Configuration
- **Docker Compose**: Environment variables in compose file
- **Kubernetes**: ConfigMaps for configuration

### Storage
- **Docker Compose**: Named volumes
- **Kubernetes**: PersistentVolumeClaims (PVCs)

### Networking
- **Docker Compose**: Bridge network
- **Kubernetes**: ClusterIP (internal) + NodePort (external)

### Scaling
- **Docker Compose**: Single instance by default
- **Kubernetes**: Easy scaling with replicas

## 🛠️ Common Operations

### View Status
```bash
kubectl get pods -n todo-app
kubectl get svc -n todo-app
```

### View Logs
```bash
kubectl logs -n todo-app -l app=todo-api --tail=50
kubectl logs -n todo-app -l app=airflow-webserver --tail=50
```

### Scale API
```bash
kubectl scale deployment/todo-api -n todo-app --replicas=3
```

### Update Image
```bash
docker build -t todoapp:latest .
kind load docker-image todoapp:latest --name todo-cluster
kubectl rollout restart deployment/todo-api -n todo-app
```

### Copy New DAGs to Airflow
```bash
WEBSERVER_POD=$(kubectl get pod -n todo-app -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}')
kubectl cp ./airflow/dags/. todo-app/${WEBSERVER_POD}:/opt/airflow/dags/
kubectl rollout restart deployment/airflow-scheduler -n todo-app
```

### Port Forward (Alternative to NodePort)
```bash
kubectl port-forward -n todo-app svc/todo-api 8080:8080
kubectl port-forward -n todo-app svc/airflow-webserver 8082:8080
kubectl port-forward -n todo-app svc/mlflow 5500:5000
```

## 📚 Documentation

- **[DEPLOYMENT.md](DEPLOYMENT.md)**: Detailed deployment instructions
- **[QUICK_REFERENCE.md](QUICK_REFERENCE.md)**: Quick command reference

## 🔍 Troubleshooting

### Pods Not Starting
```bash
kubectl describe pod <pod-name> -n todo-app
kubectl logs <pod-name> -n todo-app
kubectl get events -n todo-app --sort-by='.lastTimestamp'
```

### Image Pull Errors
```bash
kind load docker-image todoapp:latest --name todo-cluster
```

### Database Connection Issues
```bash
kubectl exec -it -n todo-app <postgres-pod> -- psql -U todo -d tododb -c "SELECT 1;"
```

### Airflow DAGs Not Showing
```bash
# Copy DAGs again
WEBSERVER_POD=$(kubectl get pod -n todo-app -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}')
kubectl cp ./airflow/dags/. todo-app/${WEBSERVER_POD}:/opt/airflow/dags/
kubectl rollout restart deployment/airflow-scheduler -n todo-app
```

## ⚠️ Production Considerations

This setup is for **local development** with kind. For production:

1. ✅ Use proper container registry (not `imagePullPolicy: Never`)
2. ✅ Store sensitive data in Secrets (not ConfigMaps)
3. ✅ Add resource quotas and limits
4. ✅ Use Ingress instead of NodePort
5. ✅ Use StatefulSets for databases
6. ✅ Add backup and restore procedures
7. ✅ Add monitoring (Prometheus/Grafana)
8. ✅ Configure autoscaling (HPA)
9. ✅ Add network policies
10. ✅ Use managed databases (RDS, Cloud SQL)
11. ✅ Use Helm for easier management
12. ✅ Add proper health checks and liveness probes
13. ✅ Implement GitOps (ArgoCD, Flux)
14. ✅ Add log aggregation (ELK, Loki)

## 📝 Notes

- All services run in the `todo-app` namespace
- Images use `imagePullPolicy: Never` for kind compatibility
- NodePorts are fixed for easy access (30080, 30082, 30500)
- Resource requests/limits are minimal for local development
- PVCs use default storage class (hostPath in kind)

