#!/bin/bash
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

CLUSTER_NAME="todo-cluster"
NAMESPACE="todo-app"

echo -e "${GREEN}=== Kubernetes Cluster Health Check ===${NC}\n"

# Check if cluster exists
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo -e "${RED}✗ Cluster '${CLUSTER_NAME}' does not exist${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Cluster exists${NC}"

# Check control plane components
echo -e "\n${YELLOW}Checking control plane components...${NC}"
CONTROLLER_STATUS=$(kubectl get pods -n kube-system -l component=kube-controller-manager --no-headers 2>/dev/null | awk '{print $3}' || echo "Unknown")
SCHEDULER_STATUS=$(kubectl get pods -n kube-system -l component=kube-scheduler --no-headers 2>/dev/null | awk '{print $3}' || echo "Unknown")

if [[ "$CONTROLLER_STATUS" == "Running" ]]; then
    echo -e "${GREEN}✓ Controller manager: Running${NC}"
else
    echo -e "${RED}✗ Controller manager: $CONTROLLER_STATUS${NC}"
    echo -e "${YELLOW}  Attempting to fix...${NC}"
    kubectl delete pod -n kube-system -l component=kube-controller-manager 2>/dev/null || true
    sleep 5
fi

if [[ "$SCHEDULER_STATUS" == "Running" ]]; then
    echo -e "${GREEN}✓ Scheduler: Running${NC}"
else
    echo -e "${RED}✗ Scheduler: $SCHEDULER_STATUS${NC}"
    echo -e "${YELLOW}  Attempting to fix...${NC}"
    kubectl delete pod -n kube-system -l component=kube-scheduler 2>/dev/null || true
    sleep 5
fi

# Check namespace
echo -e "\n${YELLOW}Checking namespace...${NC}"
if kubectl get namespace ${NAMESPACE} &>/dev/null; then
    echo -e "${GREEN}✓ Namespace '${NAMESPACE}' exists${NC}"
else
    echo -e "${YELLOW}Creating namespace '${NAMESPACE}'...${NC}"
    kubectl create namespace ${NAMESPACE}
fi

# Check pods
echo -e "\n${YELLOW}Checking pods in ${NAMESPACE}...${NC}"
PODS=$(kubectl get pods -n ${NAMESPACE} --no-headers 2>/dev/null | wc -l | tr -d ' ')
RUNNING_PODS=$(kubectl get pods -n ${NAMESPACE} --no-headers 2>/dev/null | grep -c "Running" || echo "0")
CRASHING_PODS=$(kubectl get pods -n ${NAMESPACE} --no-headers 2>/dev/null | grep -E "CrashLoopBackOff|Error" | wc -l | tr -d ' ')

echo -e "Total pods: $PODS"
echo -e "Running: $RUNNING_PODS"
if [ "$CRASHING_PODS" -gt 0 ]; then
    echo -e "${RED}Crashing pods: $CRASHING_PODS${NC}"
    kubectl get pods -n ${NAMESPACE} | grep -E "CrashLoopBackOff|Error"
else
    echo -e "${GREEN}✓ No crashing pods${NC}"
fi

# Check services
echo -e "\n${YELLOW}Checking services...${NC}"
SERVICES=$(kubectl get svc -n ${NAMESPACE} --no-headers 2>/dev/null | wc -l | tr -d ' ')
echo -e "${GREEN}✓ Found $SERVICES services${NC}"

# Check node resources
echo -e "\n${YELLOW}Checking node resources...${NC}"
if kubectl top nodes &>/dev/null; then
    kubectl top nodes
else
    echo -e "${YELLOW}⚠ Metrics server not available (this is normal for kind)${NC}"
fi

# Check recent events
echo -e "\n${YELLOW}Recent events (last 5)...${NC}"
kubectl get events -n ${NAMESPACE} --sort-by='.lastTimestamp' 2>/dev/null | tail -5 || echo "No events"

echo -e "\n${GREEN}=== Health Check Complete ===${NC}\n"

