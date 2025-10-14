#!/bin/bash

IMAGE_VERSION="0.0.3"
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION="eu-central-1"
ECR_REPOSITORY="go-observability-demo"

# Get ECR login token
docker login -u AWS -p $(aws ecr get-login-password --region ${AWS_REGION}) ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Build image
docker build -t ${ECR_REPOSITORY}:${IMAGE_VERSION} .

# Tag image
docker tag ${ECR_REPOSITORY}:${IMAGE_VERSION} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:${IMAGE_VERSION}

# Push image
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:${IMAGE_VERSION}

echo -e "${GREEN}✅ Docker image pushed successfully${NC}"
