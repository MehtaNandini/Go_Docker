# Airflow Error Fixes Applied

## Issues Fixed

### 1. Log Serving Error
**Error**: "Request URL is missing an 'http://' or 'https://' protocol"

**Fix**: 
- Disabled remote logging: `AIRFLOW__LOGGING__REMOTE_LOGGING: "false"`
- Cleared remote log connection ID
- Set proper base URL for webserver

### 2. Executor State Mismatch
**Error**: "Executor reports task instance finished (failed) although the task says it's queued"

**Fix**:
- Added proper executor configuration
- Set parallelism and concurrency limits
- Improved task state synchronization

### 3. DNS Resolution Issues
**Error**: "could not translate host name 'postgres-airflow' to address"

**Fix**:
- Added explicit wait for postgres-airflow in initContainers
- Added wait in container startup commands
- Improved DNS resolution timing

### 4. Database Connection Issues
**Error**: Tasks failing to connect to Postgres

**Fix**:
- Added proper wait loops for postgres-airflow
- Improved connection timeout handling
- Better error handling in DAG code

## Configuration Changes

### Airflow ConfigMap Updates

Added/Updated:
```yaml
AIRFLOW__LOGGING__REMOTE_LOGGING: "false"
AIRFLOW__LOGGING__REMOTE_LOG_CONN_ID: ""
AIRFLOW__LOGGING__REMOTE_BASE_LOG_FOLDER: ""
AIRFLOW__WEBSERVER__BASE_URL: "http://localhost:30082"
AIRFLOW__CORE__DAGBAG_IMPORT_TIMEOUT: "30"
AIRFLOW__CORE__DAG_FILE_PROCESSOR_TIMEOUT: "50"
AIRFLOW__SCHEDULER__DAG_DIR_LIST_INTERVAL: "10"
AIRFLOW__CORE__EXECUTOR: "LocalExecutor"
AIRFLOW__CORE__PARALLELISM: "4"
AIRFLOW__CORE__DAG_CONCURRENCY: "16"
AIRFLOW__CORE__MAX_ACTIVE_RUNS_PER_DAG: "1"
```

### Deployment Updates

1. **InitContainers**: Added explicit wait for postgres-airflow
2. **Startup Commands**: Added wait loops before starting services
3. **Health Probes**: Improved timeout values

## How to Apply Fixes

### Option 1: Use Fix Script (Recommended)
```bash
cd k8s
./fix-airflow-errors.sh
```

### Option 2: Manual Fix
```bash
cd k8s

# Apply updated configuration
kubectl apply -f airflow.yaml

# Restart services
kubectl rollout restart deployment/airflow-scheduler -n todo-app
kubectl rollout restart deployment/airflow-webserver -n todo-app

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=airflow-scheduler -n todo-app --timeout=300s
kubectl wait --for=condition=ready pod -l app=airflow-webserver -n todo-app --timeout=300s

# Clear failed tasks
SCHEDULER_POD=$(kubectl get pod -n todo-app -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n todo-app ${SCHEDULER_POD} -- airflow tasks clear todo_ml_pipeline --yes --only-failed
```

## Verification

After applying fixes:

1. **Check pods are running**:
   ```bash
   kubectl get pods -n todo-app -l 'app in (airflow-scheduler,airflow-webserver)'
   ```

2. **Check logs**:
   ```bash
   kubectl logs -n todo-app -l app=airflow-scheduler --tail=50
   kubectl logs -n todo-app -l app=airflow-webserver --tail=50
   ```

3. **Access Airflow UI**: http://localhost:30082 (admin/admin)

4. **Test DAG**:
   - Unpause `todo_ml_pipeline` DAG
   - Trigger a manual run
   - Check task logs (should now work without protocol errors)

## Expected Behavior After Fix

✅ Logs should be viewable in Airflow UI without protocol errors
✅ Tasks should execute successfully
✅ No DNS resolution errors
✅ No executor state mismatches
✅ Database connections should work reliably

## Troubleshooting

If issues persist:

1. **Check postgres-airflow is running**:
   ```bash
   kubectl get pods -n todo-app -l app=postgres-airflow
   ```

2. **Test database connection**:
   ```bash
   SCHEDULER_POD=$(kubectl get pod -n todo-app -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}')
   kubectl exec -n todo-app ${SCHEDULER_POD} -- nc -z postgres-airflow 5432 && echo "Connection OK" || echo "Connection failed"
   ```

3. **Check DAG import errors**:
   ```bash
   SCHEDULER_POD=$(kubectl get pod -n todo-app -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}')
   kubectl exec -n todo-app ${SCHEDULER_POD} -- airflow dags list-import-errors
   ```

4. **View scheduler logs**:
   ```bash
   kubectl logs -n todo-app -l app=airflow-scheduler --tail=100 -f
   ```

## Notes

- Logs are now stored locally (not remote)
- LocalExecutor is used (single node execution)
- All fixes maintain backward compatibility
- No data loss during fixes

