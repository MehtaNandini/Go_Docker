#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

NAMESPACE="todo-app"
DAG_DIR="../airflow/dags"

echo -e "${GREEN}=== Syncing DAGs to Airflow ===${NC}\n"

# Get pod names
echo -e "${YELLOW}Getting Airflow pod names...${NC}"
WEBSERVER_POD=$(kubectl get pod -n ${NAMESPACE} -l app=airflow-webserver -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
SCHEDULER_POD=$(kubectl get pod -n ${NAMESPACE} -l app=airflow-scheduler -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -z "$WEBSERVER_POD" ]; then
    echo -e "${RED}Error: Airflow webserver pod not found${NC}"
    exit 1
fi

if [ -z "$SCHEDULER_POD" ]; then
    echo -e "${RED}Error: Airflow scheduler pod not found${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Found webserver: ${WEBSERVER_POD}${NC}"
echo -e "${GREEN}✓ Found scheduler: ${SCHEDULER_POD}${NC}\n"

# Copy DAGs to webserver
echo -e "${YELLOW}Copying DAGs to webserver...${NC}"
kubectl cp ${DAG_DIR}/. ${NAMESPACE}/${WEBSERVER_POD}:/opt/airflow/dags/
echo -e "${GREEN}✓ DAGs copied to webserver${NC}\n"

# Copy DAGs to scheduler
echo -e "${YELLOW}Copying DAGs to scheduler...${NC}"
kubectl cp ${DAG_DIR}/. ${NAMESPACE}/${SCHEDULER_POD}:/opt/airflow/dags/ -c airflow-scheduler
echo -e "${GREEN}✓ DAGs copied to scheduler${NC}\n"

# Verify DAGs
echo -e "${YELLOW}Verifying DAGs...${NC}"
kubectl exec -n ${NAMESPACE} ${SCHEDULER_POD} -c airflow-scheduler -- ls -la /opt/airflow/dags/

echo -e "\n${YELLOW}Listing DAGs (this may take a moment)...${NC}"
kubectl exec -n ${NAMESPACE} ${SCHEDULER_POD} -c airflow-scheduler -- airflow dags list 2>/dev/null || echo -e "${YELLOW}Note: DAG may take 30-60 seconds to appear${NC}"

echo -e "\n${GREEN}=== Sync Complete ===${NC}"
echo -e "\n${YELLOW}Access Airflow UI:${NC} http://localhost:30082 (admin/admin)"
echo -e "${YELLOW}Wait 30-60 seconds for DAGs to appear in the UI${NC}"

