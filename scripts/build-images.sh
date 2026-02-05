#!/bin/bash
set -e

# Configuration
REGISTRY="${REGISTRY:-your-registry}"
TAG="${TAG:-latest}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Building Docker images...${NC}"
echo "Registry: $REGISTRY"
echo "Tag: $TAG"
echo ""

# Build Collector
echo -e "${GREEN}Building Collector image...${NC}"
docker build -f deploy/docker/Dockerfile.collector \
  -t ${REGISTRY}/collector:${TAG} \
  .

echo -e "${GREEN}Building Agent image...${NC}"
docker build -f deploy/docker/Dockerfile.agent \
  -t ${REGISTRY}/agent:${TAG} \
  .

echo ""
echo -e "${GREEN}Build completed successfully!${NC}"
echo ""
echo "Images built:"
echo "  - ${REGISTRY}/collector:${TAG}"
echo "  - ${REGISTRY}/agent:${TAG}"
echo ""
echo "To push images, run:"
echo "  docker push ${REGISTRY}/collector:${TAG}"
echo "  docker push ${REGISTRY}/agent:${TAG}"
