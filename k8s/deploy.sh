#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

CLUSTER_NAME="todo-cluster"
NAMESPACE="todo-app"

echo -e "${GREEN}=== TODO App Kubernetes Deployment ===${NC}\n"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"
if ! command_exists kind; then
    echo -e "${RED}Error: kind is not installed${NC}"
    echo "Install with: brew install kind"
    exit 1
fi

if ! command_exists kubectl; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    echo "Install with: brew install kubectl"
    exit 1
fi

if ! command_exists docker; then
    echo -e "${RED}Error: docker is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites check passed${NC}\n"

# Step 1: Build Docker images
echo -e "${YELLOW}Step 1: Building Docker images...${NC}"
cd ..
docker build -t todoapp:latest .
docker build -t todo-ml:latest ./ml_service
docker build -t todo-mlflow:latest ./mlflow_server
echo -e "${GREEN}✓ Images built${NC}\n"

# Step 2: Create kind cluster if it doesn't exist
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo -e "${YELLOW}Cluster '${CLUSTER_NAME}' already exists${NC}"
else
    echo -e "${YELLOW}Step 2: Creating kind cluster...${NC}"
    cat <<EOF | kind create cluster --name ${CLUSTER_NAME} --config -
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
    echo -e "${GREEN}✓ Cluster created${NC}\n"
fi

# Step 3: Load images into kind
echo -e "${YELLOW}Step 3: Loading images into kind...${NC}"
kind load docker-image todoapp:latest --name ${CLUSTER_NAME}
kind load docker-image todo-ml:latest --name ${CLUSTER_NAME}
kind load docker-image todo-mlflow:latest --name ${CLUSTER_NAME}
echo -e "${GREEN}✓ Images loaded${NC}\n"

# Step 4: Check cluster health
cd k8s
echo -e "${YELLOW}Step 4: Checking cluster health...${NC}"
if [ -f "./check-cluster-health.sh" ]; then
    chmod +x ./check-cluster-health.sh
    ./check-cluster-health.sh || true
fi

# Step 5: Deploy to Kubernetes
echo -e "\n${YELLOW}Step 5: Deploying to Kubernetes...${NC}"

# Create namespace
kubectl apply -f namespace.yaml

# Deploy databases
echo "Deploying databases..."
kubectl apply -f postgres.yaml
kubectl apply -f postgres-airflow.yaml

echo "Waiting for databases to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres -n ${NAMESPACE} --timeout=300s || echo "Postgres may still be starting..."
kubectl wait --for=condition=ready pod -l app=postgres-airflow -n ${NAMESPACE} --timeout=300s || echo "Postgres-airflow may still be starting..."

# Deploy application services
echo "Deploying application services..."
kubectl apply -f ml.yaml
kubectl apply -f api.yaml
kubectl apply -f mlflow.yaml

echo "Waiting for services to be ready..."
kubectl wait --for=condition=ready pod -l app=ml-service -n ${NAMESPACE} --timeout=300s || echo "ML service may still be starting..."
kubectl wait --for=condition=ready pod -l app=todo-api -n ${NAMESPACE} --timeout=300s || echo "API may still be starting..."
kubectl wait --for=condition=ready pod -l app=mlflow -n ${NAMESPACE} --timeout=300s || echo "MLflow may still be starting..."

# Deploy Airflow
echo "Deploying Airflow..."
kubectl apply -f airflow.yaml

echo "Waiting for Airflow init to complete..."
kubectl wait --for=condition=complete job/airflow-init -n ${NAMESPACE} --timeout=600s || echo "Airflow init may still be running..."

echo "Waiting for Airflow webserver to be ready..."
sleep 15
kubectl wait --for=condition=ready pod -l app=airflow-webserver -n ${NAMESPACE} --timeout=300s || echo "Airflow webserver may still be starting..."

# Copy DAGs and training code
echo "Copying DAGs and training code..."
WEBSERVER_POD=$(kubectl get pod -n ${NAMESPACE} -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$WEBSERVER_POD" ]; then
    kubectl cp ../airflow/dags/. ${NAMESPACE}/${WEBSERVER_POD}:/opt/airflow/dags/ 2>/dev/null || echo "Failed to copy DAGs, will retry later"
    kubectl cp ../ml_training/. ${NAMESPACE}/${WEBSERVER_POD}:/opt/airflow/ml_training/ 2>/dev/null || echo "Failed to copy training code, will retry later"
    
    # Restart scheduler to pick up DAGs
    echo "Restarting Airflow scheduler..."
    kubectl rollout restart deployment/airflow-scheduler -n ${NAMESPACE} 2>/dev/null || true
else
    echo "Airflow webserver pod not ready yet, skipping DAG copy"
fi

echo -e "${GREEN}✓ Deployment complete${NC}\n"

# Step 6: Show status
echo -e "\n${YELLOW}Step 6: Checking deployment status...${NC}"
kubectl get pods -n ${NAMESPACE}
echo ""
kubectl get svc -n ${NAMESPACE}
echo ""
kubectl get deployments -n ${NAMESPACE}

# Show access information
echo -e "\n${GREEN}=== Access Information ===${NC}"
echo -e "${GREEN}TODO API:${NC}      http://localhost:30080"
echo -e "${GREEN}Airflow UI:${NC}    http://localhost:30082 (admin/admin)"
echo -e "${GREEN}MLflow UI:${NC}     http://localhost:30500"

echo -e "\n${GREEN}=== Quick Test ===${NC}"
echo -e "Test API health:"
echo -e "  ${YELLOW}curl http://localhost:30080/health${NC}"
echo -e "\nCreate a todo:"
echo -e "  ${YELLOW}curl -X POST http://localhost:30080/api/todos -H 'Content-Type: application/json' -d '{\"title\":\"Test\",\"tags\":[\"test\"],\"durationMinutes\":30}'${NC}"

echo -e "\n${GREEN}Deployment successful!${NC}"

