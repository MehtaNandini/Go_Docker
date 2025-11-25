#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

CLUSTER_NAME="todo-cluster"
NAMESPACE="todo-app"

echo -e "${GREEN}=== Fixing Kubernetes Deployment ===${NC}\n"

# Step 1: Check cluster health
echo -e "${YELLOW}Step 1: Checking cluster health...${NC}"
./check-cluster-health.sh

# Step 2: Restart control plane if needed
echo -e "\n${YELLOW}Step 2: Ensuring control plane stability...${NC}"
kubectl delete pod -n kube-system -l component=kube-controller-manager 2>/dev/null || true
kubectl delete pod -n kube-system -l component=kube-scheduler 2>/dev/null || true
sleep 10

# Step 3: Apply updated manifests
echo -e "\n${YELLOW}Step 3: Applying updated manifests...${NC}"
kubectl apply -f namespace.yaml
kubectl apply -f postgres.yaml
kubectl apply -f postgres-airflow.yaml
kubectl apply -f ml.yaml
kubectl apply -f api.yaml
kubectl apply -f mlflow.yaml
kubectl apply -f airflow.yaml

# Step 4: Wait for databases
echo -e "\n${YELLOW}Step 4: Waiting for databases to be ready...${NC}"
kubectl wait --for=condition=ready pod -l app=postgres -n ${NAMESPACE} --timeout=300s || true
kubectl wait --for=condition=ready pod -l app=postgres-airflow -n ${NAMESPACE} --timeout=300s || true

# Step 5: Wait for services
echo -e "\n${YELLOW}Step 5: Waiting for services to be ready...${NC}"
kubectl wait --for=condition=ready pod -l app=ml-service -n ${NAMESPACE} --timeout=300s || true
kubectl wait --for=condition=ready pod -l app=todo-api -n ${NAMESPACE} --timeout=300s || true
kubectl wait --for=condition=ready pod -l app=mlflow -n ${NAMESPACE} --timeout=300s || true

# Step 6: Wait for Airflow init
echo -e "\n${YELLOW}Step 6: Waiting for Airflow initialization...${NC}"
kubectl wait --for=condition=complete job/airflow-init -n ${NAMESPACE} --timeout=600s || true

# Step 7: Copy DAGs
echo -e "\n${YELLOW}Step 7: Copying DAGs...${NC}"
WEBSERVER_POD=$(kubectl get pod -n ${NAMESPACE} -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$WEBSERVER_POD" ]; then
    kubectl cp ../airflow/dags/. ${NAMESPACE}/${WEBSERVER_POD}:/opt/airflow/dags/ 2>/dev/null || true
    kubectl cp ../ml_training/. ${NAMESPACE}/${WEBSERVER_POD}:/opt/airflow/ml_training/ 2>/dev/null || true
    kubectl rollout restart deployment/airflow-scheduler -n ${NAMESPACE} 2>/dev/null || true
fi

# Step 8: Show status
echo -e "\n${YELLOW}Step 8: Final status...${NC}"
kubectl get pods -n ${NAMESPACE}
echo ""
kubectl get svc -n ${NAMESPACE}

echo -e "\n${GREEN}=== Fix Complete ===${NC}"
echo -e "\n${GREEN}Access URLs:${NC}"
echo -e "  TODO API:    http://localhost:30080"
echo -e "  Airflow UI:  http://localhost:30082 (admin/admin)"
echo -e "  MLflow UI:   http://localhost:30500"

