#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

CLUSTER_NAME="todo-cluster"
NAMESPACE="todo-app"

echo -e "${YELLOW}=== Cleanup TODO App Kubernetes Deployment ===${NC}\n"

# Ask for confirmation
read -p "This will delete the namespace and optionally the cluster. Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled"
    exit 1
fi

# Delete namespace (this removes all resources)
echo -e "${YELLOW}Deleting namespace '${NAMESPACE}'...${NC}"
kubectl delete namespace ${NAMESPACE} --ignore-not-found=true
echo -e "${GREEN}✓ Namespace deleted${NC}\n"

# Ask about cluster deletion
read -p "Do you want to delete the kind cluster '${CLUSTER_NAME}'? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Deleting kind cluster...${NC}"
    kind delete cluster --name ${CLUSTER_NAME}
    echo -e "${GREEN}✓ Cluster deleted${NC}"
else
    echo -e "${YELLOW}Cluster preserved${NC}"
fi

echo -e "\n${GREEN}Cleanup complete!${NC}"

