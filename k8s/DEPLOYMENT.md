# Kubernetes Deployment Guide

This guide walks you through deploying the TODO application stack to a local Kubernetes cluster using kind.

## Prerequisites

1. **Install kind (Kubernetes in Docker)**:
   ```bash
   # macOS
   brew install kind
   
   # Or using go
   go install sigs.k8s.io/kind@v0.20.0
   ```

2. **Install kubectl**:
   ```bash
   # macOS
   brew install kubectl
   ```

3. **Build Docker images** (before creating the cluster):
   ```bash
   docker build -t todoapp:latest .
   docker build -t todo-ml:latest ./ml_service
   docker build -t todo-mlflow:latest ./mlflow_server
   ```

## Step 1: Create kind Cluster

Create a kind cluster with port mappings for NodePort services:

```bash
kind create cluster --name todo-cluster --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: 30080
    protocol: TCP
  - containerPort: 30082
    hostPort: 30082
    protocol: TCP
  - containerPort: 30500
    hostPort: 30500
    protocol: TCP
EOF
```

Verify cluster is running:
```bash
kubectl cluster-info --context kind-todo-cluster
```

## Step 2: Load Docker Images into kind

Since we're using `imagePullPolicy: Never`, we need to load the images into kind:

```bash
kind load docker-image todoapp:latest --name todo-cluster
kind load docker-image todo-ml:latest --name todo-cluster
kind load docker-image todo-mlflow:latest --name todo-cluster
```

## Step 3: Deploy the Stack

Apply all Kubernetes manifests in the correct order:

```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Deploy databases first
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/postgres-airflow.yaml

# Wait for databases to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n todo-app --timeout=300s
kubectl wait --for=condition=ready pod -l app=postgres-airflow -n todo-app --timeout=300s

# Deploy application services
kubectl apply -f k8s/ml.yaml
kubectl apply -f k8s/api.yaml
kubectl apply -f k8s/mlflow.yaml

# Wait for services to be ready
kubectl wait --for=condition=ready pod -l app=ml-service -n todo-app --timeout=300s
kubectl wait --for=condition=ready pod -l app=todo-api -n todo-app --timeout=300s
kubectl wait --for=condition=ready pod -l app=mlflow -n todo-app --timeout=300s

# Initialize Airflow (run DB migrations and create admin user)
kubectl apply -f k8s/airflow.yaml
kubectl wait --for=condition=complete job/airflow-init -n todo-app --timeout=600s

# Copy DAGs and training code to persistent volumes
# Note: In production, you'd use a ConfigMap, GitSync, or init container
kubectl cp ./airflow/dags/. $(kubectl get pod -n todo-app -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}'):/opt/airflow/dags/ -n todo-app
kubectl cp ./ml_training/. $(kubectl get pod -n todo-app -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}'):/opt/airflow/ml_training/ -n todo-app
```

## Step 4: Verify Deployment

Check all pods are running:
```bash
kubectl get pods -n todo-app
```

Expected output should show all pods in `Running` or `Completed` state:
```
NAME                                   READY   STATUS      RESTARTS   AGE
airflow-init-xxxxx                     0/1     Completed   0          5m
airflow-scheduler-xxxxx                1/1     Running     0          5m
airflow-webserver-xxxxx                1/1     Running     0          5m
mlflow-xxxxx                           1/1     Running     0          7m
ml-service-xxxxx                       1/1     Running     0          8m
postgres-xxxxx                         1/1     Running     0          10m
postgres-airflow-xxxxx                 1/1     Running     0          10m
todo-api-xxxxx                         1/1     Running     0          8m
todo-api-yyyyy                         1/1     Running     0          8m
```

Check services:
```bash
kubectl get svc -n todo-app
```

Check logs if needed:
```bash
# Check API logs
kubectl logs -n todo-app -l app=todo-api --tail=50

# Check ML service logs
kubectl logs -n todo-app -l app=ml-service --tail=50

# Check Airflow webserver logs
kubectl logs -n todo-app -l app=airflow-webserver --tail=50
```

## Step 5: Access the Services

### Option A: Using NodePort (Recommended for kind)

Services are exposed on these ports:

1. **TODO API**: http://localhost:30080
   ```bash
   curl http://localhost:30080/health
   ```

2. **Airflow UI**: http://localhost:30082
   - Username: `admin`
   - Password: `admin`

3. **MLflow UI**: http://localhost:30500

### Option B: Using Port-Forward (Alternative)

If NodePort doesn't work, use port-forward:

```bash
# TODO API
kubectl port-forward -n todo-app svc/todo-api 8080:8080

# Airflow UI
kubectl port-forward -n todo-app svc/airflow-webserver 8082:8080

# MLflow UI
kubectl port-forward -n todo-app svc/mlflow 5500:5000
```

Then access:
- TODO API: http://localhost:8080
- Airflow UI: http://localhost:8082
- MLflow UI: http://localhost:5500

## Testing the Application

### Test TODO API

Create a todo:
```bash
curl -X POST http://localhost:30080/todos \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Kubernetes Todo",
    "description": "Testing on k8s cluster"
  }'
```

List todos:
```bash
curl http://localhost:30080/todos
```

### Test ML Integration

The ML service should automatically score new todos. Check the priority field in the response.

### Test Airflow

1. Open http://localhost:30082 (or 8082 if using port-forward)
2. Login with admin/admin
3. Find the `todo_ml_pipeline` DAG
4. Trigger a manual run
5. Monitor the execution

## Key Differences from Docker Compose

### Service Discovery
- **Docker Compose**: Uses service names directly (e.g., `postgres`, `ml`)
- **Kubernetes**: Uses DNS with format `<service-name>.<namespace>.svc.cluster.local`
  - For same namespace, just `<service-name>` works (e.g., `postgres`, `ml`)

### Environment Variables
- Same environment variables are used
- Managed via ConfigMaps for better separation of config from code

### Networking
- **Internal communication**: ClusterIP services (postgres, ml, mlflow)
- **External access**: NodePort services (todo-api, airflow, mlflow)
- Services communicate using DNS names, not localhost

### Storage
- **Docker Compose**: Named volumes
- **Kubernetes**: PersistentVolumeClaims (PVCs)
- kind automatically provisions local storage

### Scaling
- API runs with 2 replicas for high availability
- Can scale with: `kubectl scale deployment/todo-api -n todo-app --replicas=3`

## Troubleshooting

### Pods not starting
```bash
# Check pod status
kubectl describe pod <pod-name> -n todo-app

# Check events
kubectl get events -n todo-app --sort-by='.lastTimestamp'
```

### Database connection issues
```bash
# Check if postgres is ready
kubectl exec -it -n todo-app <postgres-pod-name> -- psql -U todo -d tododb -c "SELECT 1;"
```

### Image pull errors
```bash
# Re-load images into kind
kind load docker-image todoapp:latest --name todo-cluster
kind load docker-image todo-ml:latest --name todo-cluster
kind load docker-image todo-mlflow:latest --name todo-cluster
```

### Airflow DAGs not showing
```bash
# Copy DAGs again
kubectl cp ./airflow/dags/. $(kubectl get pod -n todo-app -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}'):/opt/airflow/dags/ -n todo-app

# Restart scheduler
kubectl rollout restart deployment/airflow-scheduler -n todo-app
```

### Port conflicts
If ports 30080, 30082, or 30500 are in use, edit the NodePort values in:
- `k8s/api.yaml` (nodePort: 30080)
- `k8s/airflow.yaml` (nodePort: 30082)
- `k8s/mlflow.yaml` (nodePort: 30500)

## Cleanup

Delete the entire deployment:
```bash
kubectl delete namespace todo-app
```

Delete the kind cluster:
```bash
kind delete cluster --name todo-cluster
```

## Production Considerations

For production deployment, consider:

1. **Use proper image registry** (not `imagePullPolicy: Never`)
2. **Add resource quotas** and limits
3. **Use Secrets** for sensitive data (passwords, keys)
4. **Add Ingress** instead of NodePort
5. **Use StatefulSets** for databases with proper backup strategy
6. **Add monitoring** (Prometheus, Grafana)
7. **Configure autoscaling** (HPA)
8. **Use Helm** for easier management
9. **Add network policies** for security
10. **Use external managed databases** (RDS, Cloud SQL)

