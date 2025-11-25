# Quick Fix Guide - Kubernetes Deployment Issues

## Problem Summary
- Services disappearing from Kubernetes UI
- Services only visible for a short time
- Connection issues
- Control plane components crashing

## Quick Fix (Run This First)

```bash
cd k8s

# 1. Fix the deployment with updated configurations
./fix-deployment.sh

# 2. Wait 2-3 minutes for everything to stabilize

# 3. Verify everything is working
kubectl get pods -n todo-app
kubectl get svc -n todo-app

# 4. Test services
curl http://localhost:30080/health
```

## What Was Fixed

### 1. Health Probe Timeouts
- Increased timeout values to prevent false failures
- Services now have more time to respond to health checks

### 2. Resource Labels
- Added proper labels so services appear in Kubernetes dashboard
- All resources are now properly grouped

### 3. Control Plane Stability
- Fixed controller manager and scheduler crashes
- Added health monitoring

### 4. Deployment Scripts
- Improved error handling
- Added health checks before deployment

## Access Your Services

After running `./fix-deployment.sh`, access:

- **TODO API**: http://localhost:30080
- **Airflow UI**: http://localhost:30082 (admin/admin)
- **MLflow UI**: http://localhost:30500

## Kubernetes Dashboard

To access Kubernetes dashboard:

```bash
# 1. Apply RBAC
kubectl apply -f dashboard-rbac.yaml

# 2. Get token
./get-dashboard-token.sh

# 3. Start proxy (in separate terminal)
kubectl proxy

# 4. Open browser
open http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/
```

## If Issues Persist

1. **Check cluster health**:
   ```bash
   ./check-cluster-health.sh
   ```

2. **Restart control plane**:
   ```bash
   kubectl delete pod -n kube-system -l component=kube-controller-manager
   kubectl delete pod -n kube-system -l component=kube-scheduler
   sleep 10
   ```

3. **Full redeploy** (if needed):
   ```bash
   ./cleanup.sh
   ./deploy.sh
   ```

## Monitoring

Watch pods in real-time:
```bash
kubectl get pods -n todo-app -w
```

Check recent events:
```bash
kubectl get events -n todo-app --sort-by='.lastTimestamp' | tail -20
```

View logs:
```bash
kubectl logs -n todo-app -l app=todo-api --tail=50
```

## Expected Results

After fix:
- ✅ All pods remain running (no constant restarts)
- ✅ Services visible in Kubernetes dashboard
- ✅ Services accessible via NodePort URLs
- ✅ Control plane components stable
- ✅ No health probe timeouts

## Files Changed

- `api.yaml` - Updated health probes and labels
- `ml.yaml` - Updated health probes and labels
- `postgres.yaml` - Updated health probes and labels
- `postgres-airflow.yaml` - Updated health probes and labels
- `mlflow.yaml` - Updated health probes and labels
- `airflow.yaml` - Updated health probes and labels
- `namespace.yaml` - Added labels
- `deploy.sh` - Improved error handling
- New: `check-cluster-health.sh` - Health monitoring
- New: `fix-deployment.sh` - Fix existing deployments
- New: `dashboard-rbac.yaml` - Dashboard access

See `FIXES_APPLIED.md` for detailed changes.

