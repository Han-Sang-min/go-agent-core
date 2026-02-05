#!/bin/bash
set -e

# Configuration
REGISTRY="${REGISTRY:-your-registry}"
TAG="${TAG:-latest}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Pushing Docker images...${NC}"
echo "Registry: $REGISTRY"
echo "Tag: $TAG"
echo ""

# Push Collector
echo -e "${GREEN}Pushing Collector image...${NC}"
docker push ${REGISTRY}/collector:${TAG}

# Push Agent
echo -e "${GREEN}Pushing Agent image...${NC}"
docker push ${REGISTRY}/agent:${TAG}

echo ""
echo -e "${GREEN}Push completed successfully!${NC}"
