# Quick Start Guide

## ✅ Everything is Running!

All services are deployed and operational. Here's how to use them:

## Access URLs

| Service | URL | Credentials |
|---------|-----|-------------|
| **TODO API** | http://localhost:30080 | N/A |
| **Airflow UI** | http://localhost:30082 | admin / admin |
| **MLflow UI** | http://localhost:30500 | N/A |

## Quick Test

Run this command to test all services:

```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./test-services.sh
```

## Basic Usage

### 1. Using TODO API

**Open in browser:**
```bash
open http://localhost:30080
```

**Create a TODO:**
```bash
curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"My Task","tags":["work"],"durationMinutes":30}'
```

**List all TODOs:**
```bash
curl http://localhost:30080/api/todos
```

**Health check:**
```bash
curl http://localhost:30080/health
```

### 2. Using Airflow UI

**Open in browser:**
```bash
open http://localhost:30082
```

**Login:** 
- Username: `admin`
- Password: `admin`

**The DAG `todo_ml_pipeline` will appear in 30-60 seconds after deployment**

**To trigger the ML pipeline manually:**
1. Go to http://localhost:30082
2. Login
3. Find `todo_ml_pipeline` in the list
4. Click the play button ▶ on the right
5. Confirm

### 3. Using MLflow UI

**Open in browser:**
```bash
open http://localhost:30500
```

You'll see all ML training runs from the Airflow pipeline here.

## Helpful Commands

### Check Status
```bash
# View all pods
kubectl get pods -n todo-app

# View all services
kubectl get svc -n todo-app

# Watch pods in real-time
watch -n 2 'kubectl get pods -n todo-app'
```

### View Logs
```bash
# TODO API logs
kubectl logs -n todo-app -l app=todo-api --tail=50

# Airflow scheduler logs
kubectl logs -n todo-app -l app=airflow-scheduler -c airflow-scheduler --tail=50

# Airflow webserver logs
kubectl logs -n todo-app -l app=airflow-webserver --tail=50
```

### Sync DAGs to Airflow
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./sync-dags.sh
```

Use this whenever you update DAG files or if DAGs don't appear in Airflow UI.

### Restart Services
```bash
# Restart TODO API
kubectl rollout restart deployment/todo-api -n todo-app

# Restart Airflow scheduler
kubectl delete pod -n todo-app -l app=airflow-scheduler

# Restart Airflow webserver  
kubectl delete pod -n todo-app -l app=airflow-webserver
```

## Complete Workflow Example

Here's a complete workflow from creating TODOs to running the ML pipeline:

```bash
# 1. Create some TODOs
curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"Review code","tags":["urgent","code-review"],"durationMinutes":45}'

curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"Write tests","tags":["testing"],"durationMinutes":90}'

curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"Deploy to production","tags":["deployment","urgent"],"durationMinutes":30}'

# 2. Verify TODOs
curl http://localhost:30080/api/todos

# 3. Open Airflow and trigger the pipeline
open http://localhost:30082
# Login with admin/admin
# Click on todo_ml_pipeline
# Click the play button ▶

# 4. Monitor the pipeline
# Watch the tasks turn green in Airflow UI
# Green = Success
# Yellow = Running
# Red = Failed

# 5. View ML results in MLflow
open http://localhost:30500
# Click on "todo-priority-model" experiment
# View the latest run with metrics
```

## Troubleshooting

### Services not accessible?

1. Check if pods are running:
```bash
kubectl get pods -n todo-app
```
All should show `READY 1/1` or `2/2` with status `Running`

2. Check if kind cluster is running:
```bash
kind get clusters
```
Should show: `todo-cluster`

3. If issues persist, restart:
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
kind delete cluster --name todo-cluster
./deploy.sh
./sync-dags.sh
```

### DAGs not showing in Airflow?

1. Wait 60 seconds after deployment
2. If still not showing, sync DAGs:
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./sync-dags.sh
```

3. Check scheduler logs:
```bash
kubectl logs -n todo-app -l app=airflow-scheduler -c airflow-scheduler --tail=50
```

### Pipeline tasks failing?

1. Check logs in Airflow UI (click on failed task → Log button)
2. Or check via command line:
```bash
kubectl logs -n todo-app -l app=airflow-scheduler -c airflow-scheduler --tail=100
```

## Clean Up

To remove everything:

```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
kind delete cluster --name todo-cluster
```

To redeploy from scratch:

```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
kind delete cluster --name todo-cluster
sleep 10
./deploy.sh
./sync-dags.sh
./test-services.sh
```

## Additional Documentation

- **[SERVICES_SUMMARY.md](SERVICES_SUMMARY.md)** - Complete service reference
- **[ACCESS_GUIDE.md](ACCESS_GUIDE.md)** - Detailed access instructions
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** - Common issues and solutions
- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Architecture and deployment details
- **[README.md](README.md)** - Main documentation

## Quick Reference

| Task | Command |
|------|---------|
| Test all services | `./test-services.sh` |
| Sync DAGs | `./sync-dags.sh` |
| View pods | `kubectl get pods -n todo-app` |
| View logs | `kubectl logs -n todo-app -l app=<service-name>` |
| Restart service | `kubectl rollout restart deployment/<name> -n todo-app` |
| Delete cluster | `kind delete cluster --name todo-cluster` |
| Redeploy | `./deploy.sh` |

## Need Help?

1. Run the test script: `./test-services.sh`
2. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
3. View logs: `kubectl logs -n todo-app -l app=<service-name>`
4. Check events: `kubectl get events -n todo-app --sort-by='.lastTimestamp'`

---

**Everything is working!** 🎉

Start by opening the services in your browser:
- TODO API: http://localhost:30080
- Airflow UI: http://localhost:30082 (admin/admin)
- MLflow UI: http://localhost:30500

