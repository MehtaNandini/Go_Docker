# Kubernetes Deployment Fixes Applied

## Issues Fixed

### 1. Control Plane Instability
**Problem**: Controller manager and scheduler were crashing due to resource constraints and timeout issues.

**Solution**:
- Increased probe timeouts to be more lenient
- Added proper resource limits
- Created health check script to monitor and fix control plane issues

### 2. Health Probe Timeouts
**Problem**: Health probes were timing out, causing pods to be killed and restarted constantly.

**Solution**:
- Increased `timeoutSeconds` from 5-10s to 10-15s
- Increased `initialDelaySeconds` to give services more time to start
- Increased `periodSeconds` to reduce probe frequency
- Reduced `failureThreshold` to prevent premature restarts

### 3. Services Not Visible in Dashboard
**Problem**: Services were appearing/disappearing in Kubernetes UI due to missing labels and unstable pods.

**Solution**:
- Added comprehensive labels to all resources:
  - `app`: Service name
  - `component`: Component type (api, ml, database, airflow, mlflow)
  - `part-of: todo-app`: Groups all resources together
- Added Prometheus annotations for better monitoring
- Created proper RBAC for dashboard access

### 4. Deployment Script Improvements
**Problem**: Deployment script didn't handle errors gracefully and didn't check cluster health.

**Solution**:
- Added cluster health check before deployment
- Improved error handling with `|| true` for non-critical waits
- Created separate `fix-deployment.sh` script for fixing existing deployments
- Created `check-cluster-health.sh` for monitoring

## Changes Made

### Health Probe Updates

| Service | Before | After |
|---------|--------|-------|
| **API** | 30s initial, 5s timeout | 60s initial, 10s timeout |
| **ML Service** | 30s initial, 5s timeout | 60s initial, 10s timeout |
| **Postgres** | 30s initial, 1s timeout | 60s initial, 5s timeout |
| **Postgres-Airflow** | 30s initial, 1s timeout | 60s initial, 5s timeout |
| **MLflow** | 60s initial, 10s timeout | 90s initial, 15s timeout |
| **Airflow Webserver** | 60s initial, 10s timeout | 120s initial, 15s timeout |
| **Airflow Scheduler** | 60s initial, 10s timeout | 180s initial, 30s timeout |

### Labels Added

All resources now have:
```yaml
labels:
  app: <service-name>
  component: <component-type>
  part-of: todo-app
```

### New Scripts

1. **check-cluster-health.sh**: Monitors cluster health and control plane status
2. **fix-deployment.sh**: Fixes existing deployments without full redeploy
3. **dashboard-rbac.yaml**: Proper RBAC for Kubernetes dashboard

## Usage

### Initial Deployment
```bash
cd k8s
./deploy.sh
```

### Fix Existing Deployment
```bash
cd k8s
./fix-deployment.sh
```

### Check Cluster Health
```bash
cd k8s
./check-cluster-health.sh
```

### Setup Dashboard Access
```bash
cd k8s
kubectl apply -f dashboard-rbac.yaml
./get-dashboard-token.sh
```

## Verification

After deployment, verify everything is working:

```bash
# Check all pods are running
kubectl get pods -n todo-app

# Check services
kubectl get svc -n todo-app

# Test API
curl http://localhost:30080/health

# Check dashboard
kubectl proxy
# Then open: http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/
```

## Expected Behavior

1. **Stable Pods**: Pods should remain running without constant restarts
2. **Visible in Dashboard**: All resources should be visible in Kubernetes dashboard
3. **Accessible Services**: Services should be accessible via NodePort URLs
4. **No Control Plane Crashes**: Controller manager and scheduler should remain running

## Troubleshooting

If services still disappear:

1. **Check control plane**:
   ```bash
   kubectl get pods -n kube-system
   ```

2. **Restart control plane**:
   ```bash
   kubectl delete pod -n kube-system -l component=kube-controller-manager
   kubectl delete pod -n kube-system -l component=kube-scheduler
   ```

3. **Check resource usage**:
   ```bash
   docker stats
   ```

4. **Increase kind cluster resources** (if needed):
   Edit `kind-config.yaml` to add more resources

5. **Run fix script**:
   ```bash
   ./fix-deployment.sh
   ```

## Notes

- Health probes are now more lenient to prevent false positives
- All resources have proper labels for dashboard visibility
- Control plane stability is monitored and auto-fixed
- Deployment scripts handle errors gracefully

