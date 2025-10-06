#!/bin/bash

# CDK Deployment script for Go Observability Demo
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="go-observability-demo"
AWS_REGION="eu-central-1"
ECR_REPOSITORY="${PROJECT_NAME}-app"

echo -e "${GREEN}🚀 Starting Go Observability Demo CDK Deployment${NC}"

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo -e "${RED}❌ AWS CLI is not installed. Please install it first.${NC}"
    exit 1
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed. Please install it first.${NC}"
    exit 1
fi

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ Node.js is not installed. Please install it first.${NC}"
    exit 1
fi

# Check if CDK is installed
if ! command -v cdk &> /dev/null; then
    echo -e "${YELLOW}📦 Installing AWS CDK...${NC}"
    npm install -g aws-cdk
fi

echo -e "${YELLOW}📋 Prerequisites check passed${NC}"

# Step 1: Deploy infrastructure with CDK
echo -e "${YELLOW}🏗️  Deploying AWS infrastructure with CDK...${NC}"

# Install dependencies
npm install

# Bootstrap CDK (if needed)
# cdk bootstrap

# Deploy the stack
cdk deploy --require-approval never

# Get outputs
ALB_DNS=$(aws cloudformation describe-stacks \
    --stack-name GoObservabilityDemoStack \
    --query 'Stacks[0].Outputs[?OutputKey==`LoadBalancerDNS`].OutputValue' \
    --output text \
    --region ${AWS_REGION})

RDS_ENDPOINT=$(aws cloudformation describe-stacks \
    --stack-name GoObservabilityDemoStack \
    --query 'Stacks[0].Outputs[?OutputKey==`DatabaseEndpoint`].OutputValue' \
    --output text \
    --region ${AWS_REGION})

ECS_CLUSTER=$(aws cloudformation describe-stacks \
    --stack-name GoObservabilityDemoStack \
    --query 'Stacks[0].Outputs[?OutputKey==`ECSClusterName`].OutputValue' \
    --output text \
    --region ${AWS_REGION})

ECS_SERVICE=$(aws cloudformation describe-stacks \
    --stack-name GoObservabilityDemoStack \
    --query 'Stacks[0].Outputs[?OutputKey==`ECSServiceName`].OutputValue' \
    --output text \
    --region ${AWS_REGION})

echo -e "${GREEN}✅ Infrastructure deployed successfully${NC}"
echo -e "${GREEN}   ALB DNS: ${ALB_DNS}${NC}"
echo -e "${GREEN}   RDS Endpoint: ${RDS_ENDPOINT}${NC}"

# # Step 2: Build and push Docker image
# echo -e "${YELLOW}🐳 Building and pushing Docker image...${NC}"

# # Get AWS account ID
# AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# # Create ECR repository if it doesn't exist
# aws ecr describe-repositories --repository-names ${ECR_REPOSITORY} --region ${AWS_REGION} 2>/dev/null || \
# aws ecr create-repository --repository-name ${ECR_REPOSITORY} --region ${AWS_REGION}

# # Get ECR login token
# aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# # Build image
# cd service
# docker build -t ${ECR_REPOSITORY}:latest .

# # Tag image
# docker tag ${ECR_REPOSITORY}:latest ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:latest

# # Push image
# docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:latest

# echo -e "${GREEN}✅ Docker image pushed successfully${NC}"

# # Step 3: Update ECS service with new image
# echo -e "${YELLOW}🔄 Updating ECS service...${NC}"

# # Update task definition with new image
# aws ecs update-service \
#     --cluster ${ECS_CLUSTER} \
#     --service ${ECS_SERVICE} \
#     --force-new-deployment \
#     --region ${AWS_REGION}

# # Wait for service to be stable
# echo -e "${YELLOW}⏳ Waiting for service to be stable...${NC}"
# aws ecs wait services-stable \
#     --cluster ${ECS_CLUSTER} \
#     --services ${ECS_SERVICE} \
#     --region ${AWS_REGION}

# echo -e "${GREEN}✅ ECS service updated successfully${NC}"

# # Step 4: Test the deployment
# echo -e "${YELLOW}🧪 Testing the deployment...${NC}"

# # Wait a bit for the service to be fully ready
# sleep 30

# # Test health endpoint
# echo -e "${YELLOW}Testing health endpoint...${NC}"
# curl -f "http://${ALB_DNS}/health" || echo -e "${RED}❌ Health check failed${NC}"

# # Test CloudWatch metrics
# echo -e "${YELLOW}Testing CloudWatch metrics...${NC}"
# echo "Check AWS CloudWatch Console for metrics"

# echo -e "${GREEN}🎉 Deployment completed successfully!${NC}"
# echo -e "${GREEN}📱 Your Go service is now accessible at: http://${ALB_DNS}${NC}"
# echo -e "${GREEN}📊 CloudWatch Dashboard: https://${AWS_REGION}.console.aws.amazon.com/cloudwatch/home?region=${AWS_REGION}#dashboards:name=GoObservabilityDemo-Dashboard${NC}"

# echo -e "${YELLOW}📝 Next steps:${NC}"
# echo -e "${YELLOW}   1. Visit http://${ALB_DNS}/health to check service health${NC}"
# echo -e "${YELLOW}   2. Check AWS CloudWatch Console for metrics${NC}"
# echo -e "${YELLOW}   3. Check CloudWatch dashboard for monitoring${NC}"
# echo -e "${YELLOW}   4. Run demo scripts to generate load and errors${NC}"
