# Kubernetes Deployment Summary

## ✅ What's Been Created

A complete Kubernetes setup for your TODO app stack has been created in the `k8s/` directory.

### 📁 Files Created

```
k8s/
├── namespace.yaml              # Namespace: todo-app
├── postgres.yaml               # Main PostgreSQL (1Gi PVC)
├── postgres-airflow.yaml       # Airflow PostgreSQL (2Gi PVC)
├── api.yaml                    # Go TODO API (2 replicas, NodePort 30080)
├── ml.yaml                     # Python ML service
├── mlflow.yaml                 # MLflow server (5Gi PVC, NodePort 30500)
├── airflow.yaml                # Airflow stack (init, webserver, scheduler, NodePort 30082)
├── kind-config.yaml            # kind cluster configuration
├── deploy.sh                   # Automated deployment script ⭐
├── cleanup.sh                  # Cleanup script
├── README.md                   # Main documentation
├── DEPLOYMENT.md               # Detailed step-by-step guide
└── QUICK_REFERENCE.md          # Command cheat sheet
```

## 🚀 Quick Start Commands

### Deploy Everything (Easiest Method)

```bash
cd k8s
./deploy.sh
```

This script will:
1. ✅ Build Docker images
2. ✅ Create kind cluster with proper port mappings
3. ✅ Load images into kind
4. ✅ Deploy all services in correct order
5. ✅ Wait for services to be ready
6. ✅ Copy DAGs and training code to Airflow
7. ✅ Show access URLs

### Access Your Services

After deployment completes:

| Service | URL | Credentials |
|---------|-----|-------------|
| **TODO API** | http://localhost:30080 | - |
| **Airflow UI** | http://localhost:30082 | admin / admin |
| **MLflow UI** | http://localhost:30500 | - |

### Test the API

```bash
# Health check
curl http://localhost:30080/health

# Create a todo
curl -X POST http://localhost:30080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Test K8s","description":"Deployed on Kubernetes!"}'

# List todos
curl http://localhost:30080/todos
```

### Cleanup

```bash
cd k8s
./cleanup.sh
```

## 🏗️ Architecture Overview

### Components Deployed

1. **Namespace**: `todo-app`
   - All resources are isolated in this namespace

2. **Databases**:
   - `postgres`: Main app database (ClusterIP)
   - `postgres-airflow`: Airflow metadata DB (ClusterIP)

3. **Application Services**:
   - `todo-api`: Go REST API with 2 replicas (ClusterIP + NodePort)
   - `ml`: Python ML scoring service (ClusterIP)
   - `mlflow`: Experiment tracking (ClusterIP + NodePort)

4. **Airflow**:
   - `airflow-init`: One-time DB setup (Job)
   - `airflow-webserver`: Web UI (ClusterIP + NodePort)
   - `airflow-scheduler`: DAG scheduler (ClusterIP)

### Service Communication

```
┌─────────────────────────────────────────────────┐
│                  Kubernetes                      │
│                                                  │
│  TODO API ──────► Postgres (postgres:5432)     │
│     │                                            │
│     └──────────► ML Service (ml:8081)          │
│                                                  │
│  Airflow ───────► Postgres-Airflow              │
│     │                                            │
│     ├──────────► MLflow (mlflow:5000)          │
│     └──────────► Postgres (postgres:5432)      │
│                                                  │
│  External Access via NodePort:                  │
│  - localhost:30080 → TODO API                   │
│  - localhost:30082 → Airflow UI                 │
│  - localhost:30500 → MLflow UI                  │
└─────────────────────────────────────────────────┘
```

## 🔄 Key Changes from Docker Compose

| Aspect | Docker Compose | Kubernetes |
|--------|---------------|------------|
| **Service Discovery** | Service names (e.g., `postgres`) | DNS: `postgres.todo-app.svc.cluster.local` or just `postgres` |
| **Configuration** | Environment vars in compose file | ConfigMaps |
| **Storage** | Named volumes | PersistentVolumeClaims (PVCs) |
| **External Access** | Published ports | NodePort or Ingress |
| **Scaling** | Manual `--scale` flag | `replicas` in Deployment |
| **Health Checks** | `healthcheck` directive | Liveness/Readiness probes |
| **Secrets** | Plain env vars | Can use Secrets (not used here for simplicity) |

### Environment Variables

All the same environment variables from docker-compose are preserved:

**API (api.yaml)**:
- `PORT=8080`
- `DATABASE_URL=postgres://todo:todo@postgres:5432/tododb?sslmode=disable`
- `ML_SERVICE_URL=http://ml:8081`

**Airflow (airflow.yaml)**:
- `MLFLOW_TRACKING_URI=http://mlflow:5000`
- `POSTGRES_HOST=postgres`
- All other Airflow configs

## 📋 Manual Deployment Steps

If you prefer to deploy manually instead of using `deploy.sh`:

### 1. Build Images
```bash
docker build -t todoapp:latest .
docker build -t todo-ml:latest ./ml_service
docker build -t todo-mlflow:latest ./mlflow_server
```

### 2. Create kind Cluster
```bash
kind create cluster --config k8s/kind-config.yaml
```

### 3. Load Images
```bash
kind load docker-image todoapp:latest --name todo-cluster
kind load docker-image todo-ml:latest --name todo-cluster
kind load docker-image todo-mlflow:latest --name todo-cluster
```

### 4. Deploy Services
```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/postgres-airflow.yaml

# Wait for DBs
kubectl wait --for=condition=ready pod -l app=postgres -n todo-app --timeout=300s
kubectl wait --for=condition=ready pod -l app=postgres-airflow -n todo-app --timeout=300s

# Deploy apps
kubectl apply -f k8s/ml.yaml
kubectl apply -f k8s/api.yaml
kubectl apply -f k8s/mlflow.yaml
kubectl apply -f k8s/airflow.yaml

# Wait for init
kubectl wait --for=condition=complete job/airflow-init -n todo-app --timeout=600s
```

### 5. Copy DAGs
```bash
WEBSERVER_POD=$(kubectl get pod -n todo-app -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}')
kubectl cp ./airflow/dags/. todo-app/${WEBSERVER_POD}:/opt/airflow/dags/
kubectl cp ./ml_training/. todo-app/${WEBSERVER_POD}:/opt/airflow/ml_training/
kubectl rollout restart deployment/airflow-scheduler -n todo-app
```

## 🛠️ Common Operations

### View Resources
```bash
# All pods
kubectl get pods -n todo-app

# All services
kubectl get svc -n todo-app

# All resources
kubectl get all -n todo-app
```

### View Logs
```bash
# API logs
kubectl logs -n todo-app -l app=todo-api --tail=50 -f

# Airflow logs
kubectl logs -n todo-app -l app=airflow-webserver --tail=50 -f
```

### Scale API
```bash
kubectl scale deployment/todo-api -n todo-app --replicas=5
```

### Update After Code Changes
```bash
# Rebuild and reload image
docker build -t todoapp:latest .
kind load docker-image todoapp:latest --name todo-cluster

# Restart deployment
kubectl rollout restart deployment/todo-api -n todo-app
```

### Port Forwarding (Alternative Access)
```bash
# If NodePort doesn't work
kubectl port-forward -n todo-app svc/todo-api 8080:8080
kubectl port-forward -n todo-app svc/airflow-webserver 8082:8080
kubectl port-forward -n todo-app svc/mlflow 5500:5000
```

## 🐛 Troubleshooting

### Check Pod Status
```bash
kubectl get pods -n todo-app
kubectl describe pod <pod-name> -n todo-app
kubectl logs <pod-name> -n todo-app
```

### Check Events
```bash
kubectl get events -n todo-app --sort-by='.lastTimestamp'
```

### Database Connection Test
```bash
POSTGRES_POD=$(kubectl get pod -n todo-app -l app=postgres -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it -n todo-app ${POSTGRES_POD} -- psql -U todo -d tododb -c "SELECT 1;"
```

### Image Not Found
```bash
# Make sure images are loaded into kind
kind load docker-image todoapp:latest --name todo-cluster
kind load docker-image todo-ml:latest --name todo-cluster
kind load docker-image todo-mlflow:latest --name todo-cluster
```

### Airflow DAGs Not Showing
```bash
# Recopy DAGs and restart
WEBSERVER_POD=$(kubectl get pod -n todo-app -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}')
kubectl cp ./airflow/dags/. todo-app/${WEBSERVER_POD}:/opt/airflow/dags/
kubectl rollout restart deployment/airflow-scheduler -n todo-app
```

## 📚 Documentation Reference

- **[README.md](k8s/README.md)**: Main overview and quick start
- **[DEPLOYMENT.md](k8s/DEPLOYMENT.md)**: Detailed step-by-step deployment guide
- **[QUICK_REFERENCE.md](k8s/QUICK_REFERENCE.md)**: kubectl command cheat sheet

## ✨ Features

### High Availability
- TODO API runs with 2 replicas
- Automatic pod restart on failure
- Liveness and readiness probes

### Persistent Storage
- Postgres: 1Gi PVC
- Postgres-Airflow: 2Gi PVC
- MLflow: 5Gi PVC
- Airflow DAGs & Logs: Shared PVCs

### Resource Management
- CPU and memory requests/limits on all pods
- Prevents resource exhaustion

### Health Monitoring
- Liveness probes: Restart unhealthy pods
- Readiness probes: Remove from service when not ready

### Security
- Non-root containers where possible
- Namespace isolation
- Service-to-service communication via ClusterIP

## ⚠️ Important Notes

1. **imagePullPolicy: Never**
   - Images are loaded directly into kind
   - No external registry needed for local development

2. **NodePort Range**
   - 30080: TODO API
   - 30082: Airflow UI
   - 30500: MLflow UI
   - Change these if ports conflict

3. **Airflow DAGs**
   - Must be copied manually after deployment
   - Use `kubectl cp` or mount from ConfigMap in production

4. **Database Persistence**
   - Data persists across pod restarts
   - Stored in kind node (lost if cluster deleted)
   - Use external DB for production

5. **Resource Limits**
   - Set conservatively for local development
   - Adjust based on your machine's capacity

## 🎯 Next Steps

### For Local Development
1. ✅ Run `./deploy.sh` to get started
2. ✅ Access services via NodePort URLs
3. ✅ Make code changes and update images
4. ✅ Test end-to-end workflows

### For Production
Consider:
- External container registry (ECR, GCR, DockerHub)
- Managed databases (RDS, Cloud SQL)
- Ingress controller for HTTPS
- Cert-manager for TLS certificates
- Secrets management (Sealed Secrets, External Secrets)
- Monitoring (Prometheus, Grafana)
- Log aggregation (ELK, Loki)
- Helm charts for easier management
- GitOps (ArgoCD, Flux)
- Service mesh (Istio, Linkerd)

## 💡 Tips

1. **Set default namespace** to avoid typing `-n todo-app`:
   ```bash
   kubectl config set-context --current --namespace=todo-app
   ```

2. **Use aliases** for common commands:
   ```bash
   alias k='kubectl'
   alias kgp='kubectl get pods'
   alias kl='kubectl logs'
   ```

3. **Watch resources** with `-w` flag:
   ```bash
   kubectl get pods -n todo-app -w
   ```

4. **Use stern** for better log viewing:
   ```bash
   brew install stern
   stern -n todo-app todo-api
   ```

## 🎉 Summary

You now have:
- ✅ Complete Kubernetes manifests for all services
- ✅ Automated deployment script (`deploy.sh`)
- ✅ Cleanup script (`cleanup.sh`)
- ✅ Comprehensive documentation
- ✅ Working service communication
- ✅ Persistent storage for databases
- ✅ External access via NodePort
- ✅ Resource limits and health checks
- ✅ High availability for the API (2 replicas)

**Ready to deploy!** Run `cd k8s && ./deploy.sh` to get started! 🚀

