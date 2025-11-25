# Troubleshooting Guide

## Current Status

### Working Services ✅
- **TODO API**: http://localhost:30080 - Fully functional
- **MLflow UI**: http://localhost:30500 - Fully functional  
- **Airflow UI**: http://localhost:30082 - UI accessible (admin/admin)
- **Postgres databases**: Both running correctly

### Known Issue ⚠️
- **Airflow DAG Processing**: Scheduler experiencing intermittent DNS resolution failures for `postgres-airflow`

## Quick Fixes

### 1. Restart Airflow Scheduler (Recommended)
```bash
kubectl delete pod -n todo-app -l app=airflow-scheduler
```
This will create a fresh scheduler pod with proper DNS resolution.

### 2. Sync DAGs After Restart
```bash
cd k8s
./sync-dags.sh
```
Wait 60 seconds, then check Airflow UI.

### 3. Verify DAGs are Loaded
```bash
kubectl exec -n todo-app -l app=airflow-scheduler -c airflow-scheduler -- airflow dags list
```

Expected output:
```
dag_id           | filepath            | owner   | paused
=================+=====================+=========+=======
todo_ml_pipeline | todo_ml_pipeline.py | airflow | True  
```

### 4. Unpause and Trigger DAG
Via Airflow UI (http://localhost:30082):
1. Login with admin/admin
2. Find `todo_ml_pipeline` DAG
3. Toggle the switch to unpause it
4. Click the play button ▶ to trigger manually

Or via CLI:
```bash
kubectl exec -n todo-app -l app=airflow-scheduler -c airflow-scheduler -- \
  airflow dags unpause todo_ml_pipeline

kubectl exec -n todo-app -l app=airflow-scheduler -c airflow-scheduler -- \
  airflow dags trigger todo_ml_pipeline
```

## Common Issues

### Issue: DAGs Not Showing Up

**Symptoms**: `airflow dags list` shows "No data found"

**Solutions**:
1. Verify DAG files exist:
```bash
kubectl exec -n todo-app -l app=airflow-scheduler -c airflow-scheduler -- \
  ls -la /opt/airflow/dags/
```

2. Check for import errors:
```bash
kubectl exec -n todo-app -l app=airflow-scheduler -c airflow-scheduler -- \
  airflow dags list-import-errors
```

3. Check scheduler logs:
```bash
kubectl logs -n todo-app -l app=airflow-scheduler -c airflow-scheduler --tail=100
```

4. Restart and resync:
```bash
kubectl delete pod -n todo-app -l app=airflow-scheduler
sleep 20
cd k8s && ./sync-dags.sh
```

### Issue: DNS Resolution Failures

**Symptoms**: Error message: "could not translate host name postgres-airflow"

**Solutions**:
1. Restart affected pods:
```bash
kubectl delete pod -n todo-app -l app=airflow-scheduler
kubectl delete pod -n todo-app -l app=airflow-webserver
```

2. Verify service exists:
```bash
kubectl get svc -n todo-app postgres-airflow
```

3. Test DNS from a pod:
```bash
kubectl run -it --rm debug --image=busybox --restart=Never -n todo-app -- \
  nslookup postgres-airflow
```

### Issue: Tasks Failing in Pipeline

**Check Task Logs**:
1. Via Airflow UI:
   - Go to http://localhost:30082
   - Click on the DAG
   - Click on the failed task
   - Click "Log" button

2. Via CLI:
```bash
# Get task instance logs
kubectl logs -n todo-app -l app=airflow-scheduler -c airflow-scheduler | grep -A 20 "extract_tasks\|prepare_dataset\|train_model"
```

**Common Task Failures**:

| Task | Error | Solution |
|------|-------|----------|
| `extract_tasks` | Cannot connect to Postgres | Verify postgres service is running: `kubectl get pods -n todo-app -l app=postgres` |
| `extract_tasks` | No data in table | Create some todos via API first |
| `train_model` | MLflow connection error | Verify mlflow is running: `curl http://localhost:30500` |

### Issue: Pod Crashes or Restarts

**Check Pod Status**:
```bash
kubectl get pods -n todo-app
```

**Check Events**:
```bash
kubectl get events -n todo-app --sort-by='.lastTimestamp' | tail -20
```

**Check Resource Usage**:
```bash
kubectl top pods -n todo-app
```

**If OOMKilled** (out of memory):
1. Increase memory limits in deployment YAML
2. Or reduce number of replicas

### Issue: Services Not Accessible

**Verify NodePort Mappings**:
```bash
kubectl get svc -n todo-app
```

Should show:
- `todo-api-nodeport`: Port 8080 → NodePort 30080
- `airflow-webserver-nodeport`: Port 8080 → NodePort 30082
- `mlflow-nodeport`: Port 5000 → NodePort 30500

**Test Connectivity**:
```bash
# Test each service
curl http://localhost:30080/health
curl http://localhost:30082/health
curl http://localhost:30500/
```

**If not accessible**:
1. Verify kind cluster is running:
```bash
kind get clusters
```

2. Verify port mappings in kind config:
```bash
docker ps | grep kind-control-plane
```

3. Recreate cluster if needed:
```bash
cd k8s
./cleanup.sh
./deploy.sh
```

## Checking Service Health

### All Services Status
```bash
kubectl get all -n todo-app
```

### Individual Service Health

**TODO API**:
```bash
curl http://localhost:30080/health
# Expected: {"status":"healthy"}
```

**MLflow**:
```bash
curl -I http://localhost:30500/
# Expected: HTTP/1.1 200 OK
```

**Airflow**:
```bash
curl -I http://localhost:30082/health
# Expected: HTTP/1.1 200 OK
```

**Postgres**:
```bash
kubectl exec -it -n todo-app -l app=postgres -- \
  psql -U todo -d tododb -c "SELECT COUNT(*) FROM todos;"
```

**Postgres-Airflow**:
```bash
kubectl exec -it -n todo-app -l app=postgres-airflow -- \
  psql -U airflow -d airflow -c "SELECT COUNT(*) FROM dag;"
```

## Complete Reset

If all else fails, completely reset the deployment:

```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s

# 1. Clean up everything
./cleanup.sh

# 2. Wait for cleanup
sleep 10

# 3. Redeploy
./deploy.sh

# 4. Sync DAGs
./sync-dags.sh

# 5. Access services
echo "TODO API: http://localhost:30080"
echo "Airflow UI: http://localhost:30082 (admin/admin)"
echo "MLflow UI: http://localhost:30500"
```

## Monitoring Commands

### Watch Pods
```bash
watch -n 2 kubectl get pods -n todo-app
```

### Follow Logs
```bash
# Airflow scheduler
kubectl logs -n todo-app -l app=airflow-scheduler -c airflow-scheduler -f

# TODO API
kubectl logs -n todo-app -l app=todo-api -f

# ML Service
kubectl logs -n todo-app -l app=ml-service -f
```

### Check Resource Usage
```bash
kubectl top nodes
kubectl top pods -n todo-app
```

## Getting Help

### Collect Diagnostic Info
```bash
# Save all diagnostics
kubectl get all -n todo-app > diagnostics.txt
kubectl describe pods -n todo-app >> diagnostics.txt
kubectl get events -n todo-app --sort-by='.lastTimestamp' >> diagnostics.txt
kubectl logs -n todo-app -l app=airflow-scheduler --tail=200 >> diagnostics.txt
```

### Check Airflow Configuration
```bash
kubectl exec -n todo-app -l app=airflow-webserver -- airflow config list
```

### Export Database Data
```bash
kubectl exec -n todo-app -l app=postgres -- \
  pg_dump -U todo tododb > backup.sql
```

