#!/bin/bash

IMAGE_VERSION="0.1.2"
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION="eu-central-1"
ECR_REPOSITORY="go-observability-demo"

# Create ECR repository if it doesn't exist
aws ecr describe-repositories --repository-names ${ECR_REPOSITORY} --region ${AWS_REGION} 2>/dev/null || \
aws ecr create-repository --repository-name ${ECR_REPOSITORY} --region ${AWS_REGION}

# Get ECR login token
docker login -u AWS -p $(aws ecr get-login-password --region ${AWS_REGION}) ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Build image
docker build -t ${ECR_REPOSITORY}:latest .

# Tag image
docker tag ${ECR_REPOSITORY}:latest ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:${IMAGE_VERSION}

# Push image
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:${IMAGE_VERSION}

echo -e "${GREEN}✅ Docker image pushed successfully${NC}"
