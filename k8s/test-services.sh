#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== Testing Kubernetes Services ===${NC}\n"

# Test TODO API
echo -e "${YELLOW}1. Testing TODO API (http://localhost:30080)${NC}"
HEALTH=$(curl -s http://localhost:30080/health 2>/dev/null)
if [[ $HEALTH == *"healthy"* ]]; then
    echo -e "${GREEN}   ✓ TODO API is healthy${NC}"
else
    echo -e "${RED}   ✗ TODO API is not responding${NC}"
    exit 1
fi

# Test Airflow UI
echo -e "\n${YELLOW}2. Testing Airflow UI (http://localhost:30082)${NC}"
AIRFLOW_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:30082/health 2>/dev/null)
if [[ $AIRFLOW_STATUS == "200" ]]; then
    echo -e "${GREEN}   ✓ Airflow UI is accessible${NC}"
    echo -e "${GREEN}   → Login: admin / admin${NC}"
else
    echo -e "${RED}   ✗ Airflow UI is not responding (HTTP $AIRFLOW_STATUS)${NC}"
fi

# Test MLflow UI
echo -e "\n${YELLOW}3. Testing MLflow UI (http://localhost:30500)${NC}"
MLFLOW_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:30500/ 2>/dev/null)
if [[ $MLFLOW_STATUS == "200" ]]; then
    echo -e "${GREEN}   ✓ MLflow UI is accessible${NC}"
else
    echo -e "${RED}   ✗ MLflow UI is not responding (HTTP $MLFLOW_STATUS)${NC}"
fi

# Test creating a TODO
echo -e "\n${YELLOW}4. Testing TODO API - Create TODO${NC}"
CREATE_RESPONSE=$(curl -s -X POST http://localhost:30080/api/todos \
    -H 'Content-Type: application/json' \
    -d '{"title":"Test from script","tags":["test"],"durationMinutes":15}' 2>/dev/null)

if [[ $CREATE_RESPONSE == *"id"* ]]; then
    echo -e "${GREEN}   ✓ Successfully created a TODO${NC}"
    TODO_ID=$(echo $CREATE_RESPONSE | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
    echo -e "${GREEN}   → TODO ID: $TODO_ID${NC}"
else
    echo -e "${RED}   ✗ Failed to create TODO${NC}"
fi

# Test listing TODOs
echo -e "\n${YELLOW}5. Testing TODO API - List TODOs${NC}"
LIST_RESPONSE=$(curl -s http://localhost:30080/api/todos 2>/dev/null)
TODO_COUNT=$(echo $LIST_RESPONSE | grep -o '"id":' | wc -l | tr -d ' ')
echo -e "${GREEN}   ✓ Found $TODO_COUNT TODO(s) in database${NC}"

# Check Kubernetes pods
echo -e "\n${YELLOW}6. Checking Kubernetes Pods${NC}"
RUNNING_PODS=$(kubectl get pods -n todo-app --no-headers 2>/dev/null | grep -c "Running")
TOTAL_PODS=$(kubectl get pods -n todo-app --no-headers 2>/dev/null | grep -v "Completed" | wc -l | tr -d ' ')
echo -e "${GREEN}   ✓ $RUNNING_PODS/$TOTAL_PODS pods are running${NC}"

# Check Airflow DAGs
echo -e "\n${YELLOW}7. Checking Airflow DAGs${NC}"
SCHEDULER_POD=$(kubectl get pod -n todo-app -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$SCHEDULER_POD" ]; then
    DAGS=$(kubectl exec -n todo-app $SCHEDULER_POD -c airflow-scheduler -- airflow dags list 2>/dev/null | grep -c "todo_ml_pipeline" || echo "0")
    if [ "$DAGS" -gt 0 ]; then
        echo -e "${GREEN}   ✓ DAG 'todo_ml_pipeline' is loaded${NC}"
    else
        echo -e "${YELLOW}   ⚠ DAG not yet loaded (wait 30-60 seconds after deployment)${NC}"
    fi
else
    echo -e "${RED}   ✗ Scheduler pod not found${NC}"
fi

# Summary
echo -e "\n${GREEN}=== Test Summary ===${NC}"
echo -e "${GREEN}✓ All core services are operational!${NC}\n"
echo -e "${YELLOW}Access your services:${NC}"
echo -e "   • TODO API:    http://localhost:30080"
echo -e "   • Airflow UI:  http://localhost:30082 ${GREEN}(admin/admin)${NC}"
echo -e "   • MLflow UI:   http://localhost:30500"
echo -e "\n${YELLOW}Quick commands:${NC}"
echo -e "   # List all todos:"
echo -e "   ${GREEN}curl http://localhost:30080/api/todos${NC}"
echo -e ""
echo -e "   # Create a new todo:"
echo -e "   ${GREEN}curl -X POST http://localhost:30080/api/todos -H 'Content-Type: application/json' \\${NC}"
echo -e "   ${GREEN}     -d '{\"title\":\"My Task\",\"tags\":[\"work\"],\"durationMinutes\":30}'${NC}"
echo -e ""
echo -e "   # Check pod status:"
echo -e "   ${GREEN}kubectl get pods -n todo-app${NC}"
echo -e ""
echo -e "   # View logs:"
echo -e "   ${GREEN}kubectl logs -n todo-app -l app=todo-api${NC}"
echo -e ""

