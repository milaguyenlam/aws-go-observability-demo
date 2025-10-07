# Go Team Coffee Service - Observability Demo ☕

A comprehensive Go microservice demonstration showcasing production-ready observability practices in AWS. This project implements structured logging, custom metrics, and distributed tracing to monitor a coffee order management system.

## 🎯 Project Overview

The **Go Team Coffee Service** is a REST API that manages coffee orders for different team members, each with unique behaviors that demonstrate various types of issues that observability helps detect and debug.

### Key Features

- **REST API** for creating and retrieving coffee orders
- **PostgreSQL database** with pgx driver and automatic tracing
- **Person-specific endpoints** with different behaviors (slow queries, memory leaks, errors, etc.)
- **Comprehensive observability stack** with AWS CloudWatch, Zap logging, and OpenTelemetry tracing
- **AWS infrastructure** deployed with CDK (ECS Fargate, RDS, ALB, CloudWatch)

## 🛠️ Technology Stack

### Backend
- **Go 1.21** - Application language
- **Chi Router** - HTTP routing and middleware
- **pgx v5** - PostgreSQL driver with built-in tracing
- **Zap Logger** - Structured logging
- **OpenTelemetry** - Distributed tracing
- **AWS SDK** - CloudWatch metrics

### Infrastructure
- **AWS CDK** - Infrastructure as Code
- **ECS Fargate** - Container orchestration
- **RDS PostgreSQL** - Managed database
- **Application Load Balancer** - Traffic distribution
- **CloudWatch** - Metrics, logs, and dashboards
- **AWS X-Ray** - Distributed tracing
- **ECR** - Container registry

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+**
- **Docker**
- **AWS CLI** configured with appropriate permissions
- **AWS CDK** installed (`npm install -g aws-cdk`)
- **Node.js** (for CDK)

### 1. Clone and Setup

```bash
git clone <repository-url>
cd gomeetup-demo
```

### 2. Local Development

#### Database Setup
```bash
# Start PostgreSQL locally (using Docker)
docker run --name postgres-demo \
  -e POSTGRES_DB=observability_demo \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  -d postgres:15
```

#### Environment Variables
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=observability_demo
export DB_USER=postgres
export DB_PASSWORD=password
export AWS_REGION=eu-central-1
export PORT=8080
```

#### Run the Service
```bash
cd service
go mod tidy
go run .
```

The service will be available at `http://localhost:8080`

### 3. AWS Deployment

#### Deploy Infrastructure
```bash
cd infra
npm install
cdk bootstrap  # First time only
cdk deploy
```

#### Build and Push Docker Image
```bash
cd service
chmod +x build-image.sh
./build-image.sh
```

#### Update ECS Service
```bash
# Get the ECS service name from CDK output
aws ecs update-service \
  --cluster <cluster-name> \
  --service <service-name> \
  --force-new-deployment
```

## 📊 Observability Features

### 1. Structured Logging
- **Zap logger** with JSON output
- **Request correlation** with unique request IDs
- **Trace correlation** with OpenTelemetry trace IDs
- **Structured fields** for easy parsing and filtering

### 2. Custom Metrics
- **Request duration** and count metrics
- **Endpoint-specific** metrics with dimensions
- **Business metrics** (coffee orders by type, user)
- **CloudWatch integration** with custom namespaces

### 3. Distributed Tracing
- **OpenTelemetry** integration with AWS X-Ray
- **Automatic database tracing** with pgx
- **Request flow visualization** across services
- **Performance bottleneck identification**

## 🎭 Demo Endpoints

Each endpoint demonstrates different types of issues that observability helps detect:

| Endpoint | Behavior | Issue Type | Observability Impact |
|----------|----------|------------|---------------------|
| `/coffee-mila` | Normal operation (delayed by 1 hour) | Baseline | Normal metrics and traces |
| `/coffee-tom` | 3-second delay | Performance | High response time metrics |
| `/coffee-honza` | Always returns 500 error | Reliability | Error rate metrics, failed traces |
| `/coffee-marek` | Allocates 250MB memory | Resource leak | High memory usage metrics |
| `/coffee-viking` | Unnecessary database queries | Performance | High DB query count |
| `/coffee-matus` | Saves "beer" instead of coffee | Data integrity | Business logic monitoring |

### API Usage Examples

#### Create Coffee Order
```bash
curl -X POST http://localhost:8080/coffee-tom \
  -H "Content-Type: application/json" \
  -d '{"user_name": "Tom", "coffee_type": "espresso"}'
```

#### Get Coffee Order
```bash
curl http://localhost:8080/coffee/1
```

#### Health Check
```bash
curl http://localhost:8080/health
```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `observability_demo` | Database name |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | `password` | Database password |
| `AWS_REGION` | `eu-central-1` | AWS region |
| `PORT` | `8080` | Service port |

### AWS Permissions Required

The service requires the following AWS permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudwatch:PutMetricData",
        "logs:PutLogEvents",
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "xray:PutTraceSegments",
        "xray:PutTelemetryRecords"
      ],
      "Resource": "*"
    }
  ]
}
```

## 📈 Monitoring and Alerting

### CloudWatch Dashboards
- **Request metrics**: Duration, count, error rate
- **Infrastructure metrics**: CPU, memory, database connections
- **Custom business metrics**: Coffee orders by type and user

### Key Metrics to Monitor
- **Request Duration**: P50, P95, P99 percentiles
- **Error Rate**: 4xx and 5xx responses
- **Memory Usage**: Container memory utilization
- **Database Connections**: Active connections and query duration
- **Business Metrics**: Coffee orders per minute/hour

### Recommended Alerts
- High error rate (> 5%)
- High response time (P95 > 1s)
- High memory usage (> 80%)
- Database connection issues
- Service health check failures

## 🧪 Testing the Observability

### Load Testing
```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Test different endpoints
hey -n 100 -c 10 http://localhost:8080/coffee-tom
hey -n 100 -c 10 http://localhost:8080/coffee-marek
hey -n 100 -c 10 http://localhost:8080/coffee-honza
```

### Observability Verification
1. **Check CloudWatch metrics** for custom application metrics
2. **View X-Ray traces** for request flow visualization
3. **Monitor logs** in CloudWatch Logs for structured logging
4. **Verify correlation** between logs, metrics, and traces

## 🏃‍♂️ Development Workflow

### Local Development
```bash
# Start dependencies
docker-compose up -d postgres

# Run tests
go test ./...

# Run with hot reload (using air)
air

# Build Docker image
docker build -t go-coffee-service .
```

### CI/CD Pipeline
```bash
# Run tests
go test ./...

# Build and push image
./build-image.sh

# Deploy infrastructure
cdk deploy
```
