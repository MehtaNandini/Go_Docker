# How to Start the Project - Complete Guide

## 🚀 Quick Start (3 Steps)

### Step 1: Check if Already Running
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
kubectl get pods -n todo-app
```

If you see pods running, skip to Step 3. If not, continue to Step 2.

### Step 2: Deploy the Project

**Option A: First Time Deployment (Recommended)**
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./deploy.sh
```

**Option B: Fix Existing Deployment**
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./fix-deployment.sh
```

Wait 3-5 minutes for all services to start.

### Step 3: Verify Everything is Running
```bash
kubectl get pods -n todo-app
kubectl get svc -n todo-app
```

All pods should show `Running` status.

---

## 🌐 URLs to Access

### 1. TODO API & Web UI
**URL**: http://localhost:30080

**What it is**: Main TODO application with web interface

**Test it**:
```bash
# Health check
curl http://localhost:30080/health

# Create a todo
curl -X POST http://localhost:30080/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"My First Task","tags":["work"],"durationMinutes":30}'

# List todos
curl http://localhost:30080/api/todos
```

**Open in browser**: Just go to http://localhost:30080

---

### 2. Airflow UI
**URL**: http://localhost:30082

**Login Credentials**:
- Username: `admin`
- Password: `admin`

**What it is**: Workflow orchestration tool for ML pipeline

**Features**:
- View and manage DAGs (workflows)
- Monitor task execution
- View logs
- Trigger manual runs

**Open in browser**: http://localhost:30082

---

### 3. MLflow UI
**URL**: http://localhost:30500

**What it is**: ML experiment tracking and model registry

**Features**:
- View ML experiment runs
- Compare model metrics
- Download trained models
- View artifacts

**Open in browser**: http://localhost:30500

---

### 4. Kubernetes Dashboard (Optional)
**URL**: http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/

**Setup**:
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s

# 1. Apply RBAC
kubectl apply -f dashboard-rbac.yaml

# 2. Get access token
./get-dashboard-token.sh

# 3. Start proxy (in a separate terminal)
kubectl proxy

# 4. Open browser to the URL above
# 5. Select "Token" and paste the token from step 2
```

---

## 📋 Complete Start Checklist

### Prerequisites Check
```bash
# Check if Docker is running
docker ps

# Check if kind is installed
kind version

# Check if kubectl is installed
kubectl version --client
```

### Start the Project
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s

# Check cluster exists
kind get clusters

# Deploy (if cluster exists, this will update it)
./deploy.sh

# OR if you just want to fix existing deployment
./fix-deployment.sh
```

### Wait for Services
```bash
# Watch pods starting (press Ctrl+C to exit)
kubectl get pods -n todo-app -w

# Or check status
kubectl get pods -n todo-app
```

### Verify Services
```bash
# Check all pods are running
kubectl get pods -n todo-app

# Check services
kubectl get svc -n todo-app

# Test API
curl http://localhost:30080/health
```

---

## 🔍 Troubleshooting

### Services Not Accessible?

1. **Check if pods are running**:
   ```bash
   kubectl get pods -n todo-app
   ```

2. **Check pod logs**:
   ```bash
   kubectl logs -n todo-app -l app=todo-api --tail=50
   ```

3. **Check if NodePorts are correct**:
   ```bash
   kubectl get svc -n todo-app
   ```
   Should show:
   - `todo-api-nodeport` → `30080`
   - `airflow-webserver-nodeport` → `30082`
   - `mlflow-nodeport` → `30500`

4. **Restart services**:
   ```bash
   ./fix-deployment.sh
   ```

### Pods Not Starting?

1. **Check cluster health**:
   ```bash
   ./check-cluster-health.sh
   ```

2. **Check events**:
   ```bash
   kubectl get events -n todo-app --sort-by='.lastTimestamp' | tail -20
   ```

3. **Describe failing pod**:
   ```bash
   kubectl describe pod <pod-name> -n todo-app
   ```

### Port Already in Use?

If ports 30080, 30082, or 30500 are already in use:

1. **Find what's using the port**:
   ```bash
   lsof -i :30080
   lsof -i :30082
   lsof -i :30500
   ```

2. **Kill the process or change NodePort in manifests**

---

## 📊 Service Status Commands

### View All Resources
```bash
kubectl get all -n todo-app
```

### View Pods
```bash
kubectl get pods -n todo-app
```

### View Services
```bash
kubectl get svc -n todo-app
```

### View Deployments
```bash
kubectl get deployments -n todo-app
```

### View Logs
```bash
# API logs
kubectl logs -n todo-app -l app=todo-api --tail=50 -f

# ML service logs
kubectl logs -n todo-app -l app=ml-service --tail=50 -f

# Airflow webserver logs
kubectl logs -n todo-app -l app=airflow-webserver --tail=50 -f

# MLflow logs
kubectl logs -n todo-app -l app=mlflow --tail=50 -f
```

---

## 🛑 Stop the Project

### Stop All Services (Keep Data)
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
kubectl delete deployment --all -n todo-app
kubectl delete svc --all -n todo-app
```

### Complete Cleanup (Remove Everything)
```bash
cd /Users/nandini/Documents/Project/Go_Docker/k8s
./cleanup.sh
```

This will:
- Delete all deployments
- Delete all services
- Delete all PVCs (data will be lost)
- Optionally delete the cluster

---

## 📝 Quick Reference

| Service | URL | Purpose |
|---------|-----|---------|
| **TODO App** | http://localhost:30080 | Main application UI and API |
| **Airflow** | http://localhost:30082 | Workflow orchestration (admin/admin) |
| **MLflow** | http://localhost:30500 | ML experiment tracking |
| **K8s Dashboard** | http://localhost:8001/... | Cluster management UI |

---

## ✅ Success Indicators

You'll know everything is working when:

1. ✅ All pods show `Running` status:
   ```bash
   kubectl get pods -n todo-app
   ```

2. ✅ API responds:
   ```bash
   curl http://localhost:30080/health
   # Should return: {"status":"healthy"}
   ```

3. ✅ You can access web UIs in browser:
   - TODO app loads at http://localhost:30080
   - Airflow login page at http://localhost:30082
   - MLflow UI at http://localhost:30500

4. ✅ No errors in logs:
   ```bash
   kubectl logs -n todo-app -l app=todo-api --tail=20
   ```

---

## 🎯 Next Steps After Starting

1. **Create some todos** via the web UI at http://localhost:30080
2. **Check Airflow DAGs** at http://localhost:30082 (login: admin/admin)
3. **View ML experiments** at http://localhost:30500
4. **Monitor services** using `kubectl get pods -n todo-app -w`

---

## 💡 Tips

- Keep a terminal open with `kubectl get pods -n todo-app -w` to monitor status
- Use `kubectl logs -f` to follow logs in real-time
- Check `QUICK_FIX.md` if you encounter issues
- See `FIXES_APPLIED.md` for details on what was fixed

