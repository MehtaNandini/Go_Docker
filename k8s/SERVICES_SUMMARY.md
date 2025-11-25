# Kubernetes Services - Quick Reference

## ✅ All Services Are Running

### Access URLs

| Service | URL | Login | Description |
|---------|-----|-------|-------------|
| **TODO API** | http://localhost:30080 | N/A | REST API for managing TODOs |
| **Airflow UI** | http://localhost:30082 | admin/admin | ML pipeline orchestration |
| **MLflow UI** | http://localhost:30500 | N/A | ML experiment tracking |

## Service Status

Check all services:
```bash
kubectl get pods -n todo-app
kubectl get svc -n todo-app
```

Expected output: All pods should show `READY 1/1` or `2/2` with status `Running`

## Using the Services

### 1. TODO API (http://localhost:30080)

**Web Interface**: Open http://localhost:30080 in browser

**API Endpoints**:
```bash
# Health check
curl http://localhost:30080/health

# List all todos
curl http://localhost:30080/api/todos

# Create a todo
curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Complete project documentation",
    "tags": ["work", "documentation"],
    "durationMinutes": 120
  }'

# Update a todo
curl -X PUT http://localhost:30080/api/todos/1 \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Updated title",
    "completed": true,
    "tags": ["work"],
    "durationMinutes": 90
  }'

# Delete a todo
curl -X DELETE http://localhost:30080/api/todos/1
```

### 2. Airflow UI (http://localhost:30082)

**Login**:
- Username: `admin`
- Password: `admin`

**The `todo_ml_pipeline` DAG**:
- **Status**: ✅ Running and loaded
- **Schedule**: Daily at midnight (`@daily`)
- **Can trigger manually**: Yes

**How to Trigger the Pipeline**:

Option 1 - Via Airflow UI:
1. Go to http://localhost:30082
2. Login with admin/admin
3. Find `todo_ml_pipeline` in the DAGs list
4. Click the play button ▶ on the right side
5. Confirm the trigger

Option 2 - Via Command Line:
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
SCHEDULER_POD=$(kubectl get pod -n todo-app -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n todo-app ${SCHEDULER_POD} -c airflow-scheduler -- \
  airflow dags trigger todo_ml_pipeline
```

**View Pipeline Results**:
1. Click on the DAG name `todo_ml_pipeline`
2. Click on the run date to see details
3. Click on each task to view logs
4. Green = Success, Red = Failed, Yellow = Running

**Pipeline Tasks**:
1. `extract_tasks` - Extracts TODOs from Postgres
2. `prepare_dataset` - Prepares features for ML
3. `train_model` - Trains model and logs to MLflow

### 3. MLflow UI (http://localhost:30500)

**Purpose**: Track all ML training runs

**What You'll See**:
- Experiment: `todo-priority-model`
- Runs: Each pipeline execution creates a new run
- Metrics: MAE, F1 score, dataset statistics
- Parameters: Model configuration (keyword_bonus, tag_bonus, etc.)

**After Running the Pipeline**:
1. Open http://localhost:30500
2. Click on "todo-priority-model" experiment
3. View all training runs with timestamps
4. Compare metrics across runs
5. Click on a run to see detailed parameters and logs

## DAG Management

### Sync DAGs to Airflow

If DAGs are not showing up or after making changes:
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./sync-dags.sh
```

This script:
- Copies DAG files to both webserver and scheduler pods
- Verifies the files are in place
- Lists all DAGs

### Check DAG Status
```bash
SCHEDULER_POD=$(kubectl get pod -n todo-app -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}')

# List all DAGs
kubectl exec -n todo-app ${SCHEDULER_POD} -c airflow-scheduler -- airflow dags list

# Check if paused
kubectl exec -n todo-app ${SCHEDULER_POD} -c airflow-scheduler -- airflow dags show todo_ml_pipeline

# Unpause a DAG
kubectl exec -n todo-app ${SCHEDULER_POD} -c airflow-scheduler -- airflow dags unpause todo_ml_pipeline
```

## Complete Workflow Example

Here's a complete workflow from creating TODOs to training the model:

```bash
# 1. Create some TODOs
curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"Review pull requests","tags":["code-review","urgent"],"durationMinutes":45}'

curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"Write documentation","tags":["documentation"],"durationMinutes":120}'

curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"Team meeting","tags":["meeting"],"durationMinutes":60}'

# 2. Verify TODOs were created
curl http://localhost:30080/api/todos | json_pp

# 3. Trigger the ML pipeline
cd /Users/nandini/Documents/Project/Go_Docker/k8s
SCHEDULER_POD=$(kubectl get pod -n todo-app -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n todo-app ${SCHEDULER_POD} -c airflow-scheduler -- \
  airflow dags trigger todo_ml_pipeline

# 4. Monitor the pipeline
# - Open Airflow UI: http://localhost:30082
# - Click on todo_ml_pipeline
# - Watch the tasks turn green

# 5. View results in MLflow
# - Open MLflow UI: http://localhost:30500
# - Click on "todo-priority-model"
# - View the latest run with metrics and parameters
```

## Monitoring

### View Logs

**Airflow Scheduler**:
```bash
kubectl logs -n todo-app -l app=airflow-scheduler -c airflow-scheduler --tail=100 -f
```

**Airflow Webserver**:
```bash
kubectl logs -n todo-app -l app=airflow-webserver --tail=100 -f
```

**TODO API**:
```bash
kubectl logs -n todo-app -l app=todo-api --tail=100 -f
```

**ML Service**:
```bash
kubectl logs -n todo-app -l app=ml-service --tail=100 -f
```

### Check Pod Status
```bash
# All pods
kubectl get pods -n todo-app

# Watch in real-time
watch -n 2 'kubectl get pods -n todo-app'

# Detailed pod info
kubectl describe pod <pod-name> -n todo-app
```

### Check Events
```bash
# Recent events
kubectl get events -n todo-app --sort-by='.lastTimestamp' | tail -20

# All events
kubectl get events -n todo-app --sort-by='.lastTimestamp'
```

## Common Operations

### Restart a Service
```bash
# Restart TODO API
kubectl rollout restart deployment/todo-api -n todo-app

# Restart Airflow scheduler
kubectl delete pod -n todo-app -l app=airflow-scheduler

# Restart Airflow webserver
kubectl delete pod -n todo-app -l app=airflow-webserver
```

### Scale Services
```bash
# Scale TODO API to 3 replicas
kubectl scale deployment/todo-api -n todo-app --replicas=3

# Check replicas
kubectl get deployment -n todo-app
```

### Access Database
```bash
# Access main Postgres
kubectl exec -it -n todo-app -l app=postgres -- psql -U todo -d tododb

# Query TODOs
kubectl exec -it -n todo-app -l app=postgres -- \
  psql -U todo -d tododb -c "SELECT id, title, completed, tags FROM todos;"

# Access Airflow Postgres
kubectl exec -it -n todo-app -l app=postgres-airflow -- psql -U airflow -d airflow
```

## Cleanup

### Remove Everything
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./cleanup.sh
```

### Redeploy from Scratch
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./cleanup.sh
sleep 10
./deploy.sh
./sync-dags.sh
```

## Troubleshooting

If you encounter issues, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed solutions.

Quick fixes:
```bash
# 1. Restart problematic services
kubectl delete pod -n todo-app -l app=airflow-scheduler

# 2. Resync DAGs
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./sync-dags.sh

# 3. Check logs
kubectl logs -n todo-app -l app=airflow-scheduler -c airflow-scheduler --tail=100
```

## Additional Documentation

- **[ACCESS_GUIDE.md](ACCESS_GUIDE.md)** - Detailed access instructions
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** - Common issues and solutions
- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Deployment details and architecture
- **[README.md](README.md)** - Main documentation

## Quick Health Check

Run this to verify everything is working:

```bash
#!/bin/bash
echo "=== Kubernetes Services Health Check ==="
echo ""
echo "Pods:"
kubectl get pods -n todo-app
echo ""
echo "Services:"
kubectl get svc -n todo-app
echo ""
echo "TODO API:"
curl -s http://localhost:30080/health
echo ""
echo "Airflow UI:"
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" http://localhost:30082/health
echo ""
echo "MLflow UI:"
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" http://localhost:30500/
echo ""
echo "DAGs:"
SCHEDULER_POD=$(kubectl get pod -n todo-app -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n todo-app ${SCHEDULER_POD} -c airflow-scheduler -- airflow dags list 2>/dev/null || echo "Checking DAGs..."
```

Save this as `health-check.sh` and run it anytime!

