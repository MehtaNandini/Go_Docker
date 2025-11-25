#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=== Kubernetes Dashboard Access ===${NC}\n"

# Create service account and role binding
echo -e "${YELLOW}Creating service account...${NC}"
kubectl create serviceaccount dashboard-admin -n kube-system 2>/dev/null || echo "Service account already exists"

kubectl create clusterrolebinding dashboard-admin \
  --clusterrole=cluster-admin \
  --serviceaccount=kube-system:dashboard-admin 2>/dev/null || echo "Role binding already exists"

echo -e "${GREEN}✓ Service account created${NC}\n"

# Create token
echo -e "${YELLOW}Generating access token...${NC}\n"
TOKEN=$(kubectl create token dashboard-admin -n kube-system --duration=24h)

echo -e "${GREEN}=== Your Dashboard Token ===${NC}"
echo -e "${YELLOW}(Copy this entire token)${NC}\n"
echo "$TOKEN"
echo ""

# Also show kubeconfig location
echo -e "\n${GREEN}=== Kubeconfig File Location ===${NC}"
echo -e "${YELLOW}Your kubeconfig file is at:${NC}"
echo "$HOME/.kube/config"
echo ""

echo -e "${GREEN}=== Access Methods ===${NC}"
echo -e "\n${YELLOW}Method 1: Use Token (Recommended)${NC}"
echo "1. Copy the token above"
echo "2. Select 'Token' option in the dashboard"
echo "3. Paste the token"
echo ""

echo -e "${YELLOW}Method 2: Use Kubeconfig File${NC}"
echo "1. Select 'Kubeconfig' option"
echo "2. Choose file: $HOME/.kube/config"
echo ""

echo -e "${GREEN}Token is valid for 24 hours${NC}\n"

