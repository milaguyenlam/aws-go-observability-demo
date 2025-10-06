#!/bin/bash

# CDK Cleanup script for Go Observability Demo
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🧹 Cleaning up Go Observability Demo CDK resources...${NC}"

# Check if CDK is installed
if ! command -v cdk &> /dev/null; then
    echo -e "${RED}❌ AWS CDK is not installed. Please install it first.${NC}"
    exit 1
fi

# Navigate to CDK directory
cd infrastructure-cdk

# Destroy the stack
echo -e "${YELLOW}🗑️  Destroying CDK stack...${NC}"
cdk destroy --force

echo -e "${GREEN}✅ CDK stack destroyed successfully${NC}"

# Clean up ECR repository
echo -e "${YELLOW}🗑️  Cleaning up ECR repository...${NC}"
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION="us-west-2"
ECR_REPOSITORY="go-observability-demo-app"

# Delete all images in the repository
aws ecr list-images --repository-name ${ECR_REPOSITORY} --region ${AWS_REGION} --query 'imageIds[*]' --output json | \
jq -r '.[] | .imageDigest' | \
while read digest; do
    aws ecr batch-delete-image --repository-name ${ECR_REPOSITORY} --region ${AWS_REGION} --image-ids imageDigest=${digest} 2>/dev/null || true
done

# Delete the repository
aws ecr delete-repository --repository-name ${ECR_REPOSITORY} --region ${AWS_REGION} --force 2>/dev/null || true

echo -e "${GREEN}✅ ECR repository cleaned up${NC}"

echo -e "${GREEN}🎉 Cleanup completed successfully!${NC}"
echo -e "${YELLOW}💡 All AWS resources have been removed${NC}"
