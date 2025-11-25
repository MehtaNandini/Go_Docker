# Kubernetes Quick Reference

Common commands for managing the TODO app on Kubernetes.

## Quick Start

```bash
# Deploy everything
./deploy.sh

# Cleanup everything
./cleanup.sh
```

## Cluster Management

```bash
# Create cluster
kind create cluster --name todo-cluster

# List clusters
kind get clusters

# Delete cluster
kind delete cluster --name todo-cluster

# Get cluster info
kubectl cluster-info --context kind-todo-cluster
```

## Image Management

```bash
# Build images
docker build -t todoapp:latest .
docker build -t todo-ml:latest ./ml_service
docker build -t todo-mlflow:latest ./mlflow_server

# Load images into kind
kind load docker-image todoapp:latest --name todo-cluster
kind load docker-image todo-ml:latest --name todo-cluster
kind load docker-image todo-mlflow:latest --name todo-cluster
```

## Deployment Commands

```bash
# Apply all manifests
kubectl apply -f k8s/

# Apply specific manifest
kubectl apply -f k8s/api.yaml

# Delete resources
kubectl delete -f k8s/api.yaml

# Delete namespace (removes everything)
kubectl delete namespace todo-app
```

## Viewing Resources

```bash
# List all pods
kubectl get pods -n todo-app

# List all pods with more details
kubectl get pods -n todo-app -o wide

# List services
kubectl get svc -n todo-app

# List deployments
kubectl get deployments -n todo-app

# List persistent volumes
kubectl get pvc -n todo-app

# List all resources
kubectl get all -n todo-app
```

## Inspecting Resources

```bash
# Describe a pod
kubectl describe pod <pod-name> -n todo-app

# Get pod logs
kubectl logs <pod-name> -n todo-app

# Get logs with follow
kubectl logs -f <pod-name> -n todo-app

# Get logs from all pods of a deployment
kubectl logs -n todo-app -l app=todo-api --tail=100

# Get previous pod logs (if pod crashed)
kubectl logs <pod-name> -n todo-app --previous

# Get events
kubectl get events -n todo-app --sort-by='.lastTimestamp'
```

## Accessing Pods

```bash
# Execute command in pod
kubectl exec -it <pod-name> -n todo-app -- /bin/sh

# Execute specific command
kubectl exec -it <pod-name> -n todo-app -- ls -la /app

# Access postgres
kubectl exec -it -n todo-app <postgres-pod-name> -- psql -U todo -d tododb

# Copy files to pod
kubectl cp ./local-file.txt todo-app/<pod-name>:/path/in/pod/

# Copy files from pod
kubectl cp todo-app/<pod-name>:/path/in/pod/file.txt ./local-file.txt
```

## Port Forwarding

```bash
# Forward API port
kubectl port-forward -n todo-app svc/todo-api 8080:8080

# Forward Airflow UI
kubectl port-forward -n todo-app svc/airflow-webserver 8082:8080

# Forward MLflow UI
kubectl port-forward -n todo-app svc/mlflow 5500:5000

# Forward to specific pod
kubectl port-forward -n todo-app <pod-name> 8080:8080
```

## Scaling

```bash
# Scale deployment
kubectl scale deployment/todo-api -n todo-app --replicas=3

# Autoscale (requires metrics-server)
kubectl autoscale deployment/todo-api -n todo-app --cpu-percent=80 --min=2 --max=10
```

## Updates and Rollouts

```bash
# Update image
kubectl set image deployment/todo-api todo-api=todoapp:v2 -n todo-app

# Check rollout status
kubectl rollout status deployment/todo-api -n todo-app

# View rollout history
kubectl rollout history deployment/todo-api -n todo-app

# Rollback to previous version
kubectl rollout undo deployment/todo-api -n todo-app

# Rollback to specific revision
kubectl rollout undo deployment/todo-api -n todo-app --to-revision=2

# Restart deployment (recreate pods)
kubectl rollout restart deployment/todo-api -n todo-app

# Pause rollout
kubectl rollout pause deployment/todo-api -n todo-app

# Resume rollout
kubectl rollout resume deployment/todo-api -n todo-app
```

## Configuration

```bash
# View ConfigMap
kubectl get configmap -n todo-app
kubectl describe configmap api-config -n todo-app

# Edit ConfigMap
kubectl edit configmap api-config -n todo-app

# Create ConfigMap from file
kubectl create configmap my-config --from-file=config.yaml -n todo-app

# View Secrets
kubectl get secrets -n todo-app
kubectl describe secret <secret-name> -n todo-app
```

## Troubleshooting

```bash
# Check pod status
kubectl get pods -n todo-app

# Describe pod for events and issues
kubectl describe pod <pod-name> -n todo-app

# Check logs
kubectl logs <pod-name> -n todo-app

# Check previous logs (if crashed)
kubectl logs <pod-name> -n todo-app --previous

# Get events
kubectl get events -n todo-app --sort-by='.lastTimestamp'

# Check resource usage
kubectl top pods -n todo-app
kubectl top nodes

# Debug with temporary pod
kubectl run debug --image=busybox:1.36 --rm -it --restart=Never -n todo-app -- sh

# Test DNS resolution
kubectl run -it --rm debug --image=busybox:1.36 --restart=Never -n todo-app -- nslookup postgres

# Test connectivity
kubectl run -it --rm debug --image=busybox:1.36 --restart=Never -n todo-app -- wget -O- http://todo-api:8080/health
```

## Application-Specific Commands

### TODO API

```bash
# Test API health
curl http://localhost:30080/health

# Create a todo
curl -X POST http://localhost:30080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Test","description":"Testing k8s"}'

# List todos
curl http://localhost:30080/todos

# Get specific todo
curl http://localhost:30080/todos/1

# Update todo
curl -X PUT http://localhost:30080/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated","description":"Updated on k8s","completed":true}'

# Delete todo
curl -X DELETE http://localhost:30080/todos/1
```

### Airflow

```bash
# Access Airflow UI
open http://localhost:30082
# Login: admin / admin

# Copy new DAGs
WEBSERVER_POD=$(kubectl get pod -n todo-app -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}')
kubectl cp ./airflow/dags/. todo-app/${WEBSERVER_POD}:/opt/airflow/dags/

# Restart scheduler to pick up changes
kubectl rollout restart deployment/airflow-scheduler -n todo-app

# Check Airflow logs
kubectl logs -n todo-app -l app=airflow-webserver --tail=100
kubectl logs -n todo-app -l app=airflow-scheduler --tail=100

# Run Airflow CLI command
kubectl exec -it -n todo-app ${WEBSERVER_POD} -- airflow dags list
kubectl exec -it -n todo-app ${WEBSERVER_POD} -- airflow dags trigger todo_ml_pipeline
```

### MLflow

```bash
# Access MLflow UI
open http://localhost:30500

# Check MLflow logs
kubectl logs -n todo-app -l app=mlflow --tail=100
```

### Database

```bash
# Access Postgres
POSTGRES_POD=$(kubectl get pod -n todo-app -l app=postgres -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it -n todo-app ${POSTGRES_POD} -- psql -U todo -d tododb

# Run SQL query
kubectl exec -it -n todo-app ${POSTGRES_POD} -- psql -U todo -d tododb -c "SELECT * FROM todos LIMIT 10;"

# Backup database
kubectl exec -n todo-app ${POSTGRES_POD} -- pg_dump -U todo tododb > backup.sql

# Restore database
cat backup.sql | kubectl exec -i -n todo-app ${POSTGRES_POD} -- psql -U todo -d tododb
```

## Cleanup

```bash
# Delete specific deployment
kubectl delete deployment todo-api -n todo-app

# Delete namespace (removes everything)
kubectl delete namespace todo-app

# Delete cluster
kind delete cluster --name todo-cluster

# Force delete stuck resources
kubectl delete pod <pod-name> -n todo-app --grace-period=0 --force
```

## Monitoring

```bash
# Watch pods
kubectl get pods -n todo-app -w

# Watch events
kubectl get events -n todo-app -w

# Get resource usage (requires metrics-server)
kubectl top pods -n todo-app
kubectl top nodes

# Get pod resource requests/limits
kubectl describe pods -n todo-app | grep -A 5 "Limits:\|Requests:"
```

## Useful Aliases

Add these to your `~/.zshrc` or `~/.bashrc`:

```bash
alias k='kubectl'
alias kgp='kubectl get pods -n todo-app'
alias kgs='kubectl get svc -n todo-app'
alias kgd='kubectl get deployments -n todo-app'
alias kl='kubectl logs -n todo-app'
alias kd='kubectl describe -n todo-app'
alias ke='kubectl exec -it -n todo-app'
alias kpf='kubectl port-forward -n todo-app'
```

## Tips

1. **Always specify namespace**: Use `-n todo-app` or set default namespace:
   ```bash
   kubectl config set-context --current --namespace=todo-app
   ```

2. **Use labels for filtering**:
   ```bash
   kubectl get pods -n todo-app -l app=todo-api
   ```

3. **Watch for changes**:
   ```bash
   kubectl get pods -n todo-app -w
   ```

4. **Get YAML of existing resources**:
   ```bash
   kubectl get deployment todo-api -n todo-app -o yaml
   ```

5. **Dry run before applying**:
   ```bash
   kubectl apply -f k8s/api.yaml --dry-run=client
   ```

6. **Validate YAML**:
   ```bash
   kubectl apply -f k8s/api.yaml --dry-run=server
   ```

7. **Use stern for better log viewing** (install: `brew install stern`):
   ```bash
   stern -n todo-app todo-api
   ```

