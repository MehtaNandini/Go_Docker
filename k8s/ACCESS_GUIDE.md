# How to Access Kubernetes Services

## Service URLs

All services are accessible via NodePort mappings on localhost:

### 1. TODO API
```bash
# Health check
curl http://localhost:30080/health

# List todos
curl http://localhost:30080/api/todos

# Create a todo
curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"My Task","tags":["work"],"durationMinutes":30}'

# Web UI (open in browser)
open http://localhost:30080
```

### 2. Airflow UI
```bash
# Open in browser
open http://localhost:30082

# Login credentials:
# Username: admin
# Password: admin
```

### 3. MLflow UI
```bash
# Open in browser
open http://localhost:30500
```

## Kubectl Commands

### View Services
```bash
# List all services with ports
kubectl get svc -n todo-app

# Detailed service information
kubectl describe svc -n todo-app
```

### View Pods
```bash
# List all pods
kubectl get pods -n todo-app

# Watch pods in real-time
kubectl get pods -n todo-app -w

# Detailed pod information
kubectl describe pod <pod-name> -n todo-app
```

### View Logs
```bash
# Airflow webserver logs
kubectl logs -n todo-app -l app=airflow-webserver --tail=100

# Airflow scheduler logs
kubectl logs -n todo-app -l app=airflow-scheduler --tail=100

# TODO API logs
kubectl logs -n todo-app -l app=todo-api --tail=100

# ML service logs
kubectl logs -n todo-app -l app=ml-service --tail=100

# Follow logs in real-time
kubectl logs -n todo-app -l app=airflow-scheduler -f
```

### Execute Commands in Pods
```bash
# List DAGs
kubectl exec -n todo-app -l app=airflow-scheduler -- airflow dags list

# Trigger a DAG
kubectl exec -n todo-app -l app=airflow-scheduler -- airflow dags trigger todo_ml_pipeline

# Access pod shell
kubectl exec -it -n todo-app <pod-name> -- /bin/bash
```

### Port Forwarding (Alternative Access)
```bash
# Forward TODO API
kubectl port-forward -n todo-app svc/todo-api 8080:8080

# Forward Airflow
kubectl port-forward -n todo-app svc/airflow-webserver 8082:8080

# Forward MLflow
kubectl port-forward -n todo-app svc/mlflow 5000:5000
```

## Troubleshooting

### Check Resource Status
```bash
# All resources
kubectl get all -n todo-app

# Events
kubectl get events -n todo-app --sort-by='.lastTimestamp'

# Describe failing pod
kubectl describe pod <pod-name> -n todo-app
```

### Check Connectivity
```bash
# Test from within cluster
kubectl run -it --rm debug --image=busybox --restart=Never -n todo-app -- sh

# Then inside the pod:
wget -qO- http://todo-api:8080/health
wget -qO- http://mlflow:5000
```

## Service Endpoints Summary

| Service | Internal (ClusterIP) | External (NodePort) | Browser URL |
|---------|---------------------|---------------------|-------------|
| TODO API | `todo-api:8080` | `localhost:30080` | http://localhost:30080 |
| Airflow UI | `airflow-webserver:8080` | `localhost:30082` | http://localhost:30082 |
| MLflow UI | `mlflow:5000` | `localhost:30500` | http://localhost:30500 |
| Postgres | `postgres:5432` | N/A | Internal only |
| Postgres-Airflow | `postgres-airflow:5432` | N/A | Internal only |
| ML Service | `ml:8081` | N/A | Internal only |

