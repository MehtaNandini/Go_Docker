#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

NAMESPACE="todo-app"

echo -e "${GREEN}=== Fixing Airflow Errors ===${NC}\n"

# Step 1: Apply updated configuration
echo -e "${YELLOW}Step 1: Applying updated Airflow configuration...${NC}"
kubectl apply -f airflow.yaml

# Step 2: Restart scheduler to pick up changes
echo -e "\n${YELLOW}Step 2: Restarting Airflow scheduler...${NC}"
kubectl rollout restart deployment/airflow-scheduler -n ${NAMESPACE}

# Step 3: Restart webserver
echo -e "\n${YELLOW}Step 3: Restarting Airflow webserver...${NC}"
kubectl rollout restart deployment/airflow-webserver -n ${NAMESPACE}

# Step 4: Wait for pods to be ready
echo -e "\n${YELLOW}Step 4: Waiting for pods to be ready...${NC}"
kubectl wait --for=condition=ready pod -l app=airflow-scheduler -n ${NAMESPACE} --timeout=300s || echo "Scheduler may still be starting..."
kubectl wait --for=condition=ready pod -l app=airflow-webserver -n ${NAMESPACE} --timeout=300s || echo "Webserver may still be starting..."

# Step 5: Clear failed task instances
echo -e "\n${YELLOW}Step 5: Clearing failed task instances...${NC}"
SCHEDULER_POD=$(kubectl get pod -n ${NAMESPACE} -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$SCHEDULER_POD" ]; then
    echo "Clearing failed tasks for todo_ml_pipeline..."
    kubectl exec -n ${NAMESPACE} ${SCHEDULER_POD} -- airflow tasks clear todo_ml_pipeline --yes --only-failed 2>/dev/null || echo "No failed tasks to clear or command not available"
fi

# Step 6: Show status
echo -e "\n${YELLOW}Step 6: Current status...${NC}"
kubectl get pods -n ${NAMESPACE} -l 'app in (airflow-scheduler,airflow-webserver)'

echo -e "\n${GREEN}=== Fix Complete ===${NC}"
echo -e "\n${YELLOW}Next steps:${NC}"
echo -e "1. Wait 1-2 minutes for services to fully start"
echo -e "2. Access Airflow UI: http://localhost:30082 (admin/admin)"
echo -e "3. Unpause and trigger the DAG: todo_ml_pipeline"
echo -e "4. Check logs if issues persist:"
echo -e "   kubectl logs -n ${NAMESPACE} -l app=airflow-scheduler --tail=50"

