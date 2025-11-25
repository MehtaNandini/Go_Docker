#!/bin/bash

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}=== Kubernetes Prerequisites Check ===${NC}\n"

MISSING=0

# Check Docker
echo -n "Checking Docker... "
if command -v docker >/dev/null 2>&1; then
    VERSION=$(docker --version | cut -d' ' -f3 | tr -d ',')
    echo -e "${GREEN}✓ Installed${NC} (version: $VERSION)"
else
    echo -e "${RED}✗ Not installed${NC}"
    echo "  Install from: https://www.docker.com/products/docker-desktop"
    MISSING=1
fi

# Check kind
echo -n "Checking kind... "
if command -v kind >/dev/null 2>&1; then
    VERSION=$(kind version | cut -d' ' -f2)
    echo -e "${GREEN}✓ Installed${NC} (version: $VERSION)"
else
    echo -e "${RED}✗ Not installed${NC}"
    echo "  Install with: ${YELLOW}brew install kind${NC}"
    MISSING=1
fi

# Check kubectl
echo -n "Checking kubectl... "
if command -v kubectl >/dev/null 2>&1; then
    VERSION=$(kubectl version --client -o json 2>/dev/null | grep -o '"gitVersion":"[^"]*"' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Installed${NC} (version: $VERSION)"
else
    echo -e "${RED}✗ Not installed${NC}"
    echo "  Install with: ${YELLOW}brew install kubectl${NC}"
    MISSING=1
fi

echo ""

if [ $MISSING -eq 0 ]; then
    echo -e "${GREEN}✓ All prerequisites are installed!${NC}"
    echo -e "\nYou're ready to deploy! Run:"
    echo -e "  ${YELLOW}./deploy.sh${NC}"
    exit 0
else
    echo -e "${YELLOW}Please install the missing tools above.${NC}"
    echo -e "\nQuick installation commands:"
    echo -e "  ${YELLOW}brew install kind kubectl${NC}"
    echo -e "\nAfter installation, run this script again to verify."
    exit 1
fi

