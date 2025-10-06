# Go Observability Demo

A comprehensive demonstration of observability features for Go services running on AWS, including monitoring, logging, metrics, and error tracking.

## 🏗️ Architecture

```
Internet → ALB → ECS Fargate → Go Service → RDS PostgreSQL
                ↓
            CloudWatch (Logs, Metrics, Alarms)
                ↓
            AWS X-Ray (Distributed Tracing)
```

## 📋 Prerequisites

- AWS CLI configured with appropriate permissions
- Docker installed
- Terraform installed
- Go 1.21+ (for local development)
- curl (for testing)

## 🚀 Quick Start

### 1. Deploy Infrastructure and Service

```bash
# Clone and navigate to the project
cd go-observability-demo

# Deploy everything (infrastructure + service)
./deploy.sh
```

This script will:
- Deploy AWS infrastructure using Terraform
- Build and push Docker image to ECR
- Deploy Go service to ECS Fargate
- Set up CloudWatch dashboards and alarms
- Test the deployment

### 2. Run Demo Scripts

```bash
# Get the ALB DNS name from deployment output
ALB_DNS="your-alb-dns-name"

# Quick demo of all features
./demo-quick.sh http://$ALB_DNS

# Load test with multiple users
./demo-load-test.sh http://$ALB_DNS -u 20 -d 120
```

## 🔍 Observability Features

### 1. **Application Performance Monitoring (APM)**
- **CloudWatch Custom Metrics**: Request duration, error rates, custom business metrics
- **OpenTelemetry Tracing**: Distributed tracing with AWS X-Ray integration
- **Request Correlation**: Each request gets a unique ID for tracing across services

### 2. **Structured Logging**
- **Zap Logger**: High-performance structured logging
- **Request Correlation**: Each request gets a unique ID for tracing
- **Log Levels**: Info, Warn, Error with appropriate context
- **CloudWatch Logs**: Centralized log aggregation

### 3. **Health Monitoring**
- **Health Endpoint**: `/health` with database connectivity check
- **Load Balancer Health Checks**: ALB monitors service health
- **ECS Health Checks**: Container-level health monitoring

### 4. **Error Tracking**
- **Panic Recovery**: Middleware catches and logs panics
- **Error Classification**: Different error types tracked separately
- **Stack Traces**: Detailed error information in logs
- **Error Metrics**: CloudWatch counters for error rates

### 5. **Performance Monitoring**
- **Request Duration**: Histogram of response times
- **Database Query Performance**: Separate metrics for different query types
- **Resource Utilization**: CPU and memory monitoring via ECS metrics
- **Slow Query Detection**: Demo endpoint for testing slow queries

### 6. **Custom Metrics**
- **Business Metrics**: User creation, API usage patterns
- **Database Metrics**: Connection pool status, query performance
- **Application Metrics**: Custom counters and gauges
- **CloudWatch Integration**: Native AWS metrics with custom dimensions

## 📊 Monitoring Dashboards

### CloudWatch Dashboard
- **ALB Metrics**: Request count, response time, error rates
- **ECS Metrics**: CPU utilization, memory usage
- **RDS Metrics**: Database performance, connections
- **Custom Metrics**: Application-specific metrics
- **X-Ray Traces**: Distributed tracing and performance analysis

## 🧪 Demo Scenarios

### 1. **Normal Operations**
- Health checks
- User CRUD operations
- Standard API usage

### 2. **Error Scenarios**
- Demo error endpoint (`/demo/error`)
- Database connection issues
- Invalid input handling

### 3. **Performance Issues**
- Slow query simulation (`/demo/slow-query`)
- Memory leak simulation (`/demo/memory-leak`)
- High CPU usage (`/demo/high-cpu`)

### 4. **Load Testing**
- Concurrent user simulation
- High request volume
- Mixed workload patterns

## 🔧 API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Service health check |
| `/users` | GET | List all users |
| `/users` | POST | Create new user |
| `/users/{id}` | GET | Get user by ID |
| `/users/{id}` | PUT | Update user |
| `/users/{id}` | DELETE | Delete user |
| `/demo/slow-query` | GET | Simulate slow database query |
| `/demo/error` | GET | Generate demo error |
| `/demo/memory-leak` | GET | Simulate memory leak |
| `/demo/high-cpu` | GET | Simulate high CPU usage |

## 📈 Key Metrics to Monitor

### Application Metrics
- `RequestDuration`: Request duration in CloudWatch
- `RequestCount`: Total HTTP requests by endpoint and status
- `ErrorCount`: Error counts by type
- `db.query.duration`: Database query performance (in traces)

### Infrastructure Metrics
- `CPUUtilization`: ECS service CPU usage
- `MemoryUtilization`: ECS service memory usage
- `RequestCount`: ALB request count
- `TargetResponseTime`: ALB response time

### Business Metrics
- User creation rate
- API usage patterns
- Error rates by endpoint
- Performance trends

## 🚨 Alerts and Thresholds

### CloudWatch Alarms
- **High Error Rate**: >5 5xx errors in 5 minutes
- **High Response Time**: >2 seconds average response time
- **High CPU**: >80% CPU utilization
- **Database Issues**: Connection failures or slow queries

## 🛠️ Development

### Local Development
```bash
# Install dependencies
cd go-service
go mod download

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=observability_demo
export DB_USER=postgres
export DB_PASSWORD=password
export AWS_REGION=us-west-2

# Run locally
go run main.go
```

### Building Docker Image
```bash
cd go-service
docker build -t go-observability-demo:latest .
```

## 🧹 Cleanup

To avoid AWS charges, clean up resources:

```bash
cd infrastructure
terraform destroy
```

## 📚 Learning Objectives

After running this demo, you should understand:

1. **How to implement comprehensive observability** in Go services
2. **AWS monitoring best practices** for containerized applications
3. **Structured logging** with correlation IDs
4. **Custom metrics** and CloudWatch integration
5. **Error tracking** and alerting strategies
6. **Performance monitoring** and optimization
7. **Load testing** and capacity planning

## 🔗 Useful Links

- [AWS CloudWatch Documentation](https://docs.aws.amazon.com/cloudwatch/)
- [AWS X-Ray Documentation](https://docs.aws.amazon.com/xray/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)
- [Go Zap Logger](https://github.com/uber-go/zap)
- [ECS Monitoring](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/monitoring.html)

## 📝 Notes for Your Talk

1. **Start with the architecture** - show the complete stack
2. **Demonstrate normal operations** - health checks, basic API calls
3. **Generate errors** - show how they're captured and tracked
4. **Show performance issues** - slow queries, resource utilization
5. **Explain the monitoring** - dashboards, metrics, alerts
6. **Discuss debugging** - how to use logs and metrics together
7. **Cover best practices** - what to monitor, how to set thresholds

This demo provides a complete, production-ready example of observability in Go services on AWS!
