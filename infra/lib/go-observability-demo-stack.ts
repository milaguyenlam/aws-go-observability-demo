import * as cdk from "aws-cdk-lib";
import * as ec2 from "aws-cdk-lib/aws-ec2";
import * as ecs from "aws-cdk-lib/aws-ecs";
import * as ecr from "aws-cdk-lib/aws-ecr";
import * as ecs_patterns from "aws-cdk-lib/aws-ecs-patterns";
import * as rds from "aws-cdk-lib/aws-rds";
import * as logs from "aws-cdk-lib/aws-logs";
import * as iam from "aws-cdk-lib/aws-iam";
import { Construct } from "constructs";

const awsOtelCollectorConfig = `
extensions:
  health_check:

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch/traces:
    timeout: 1s
    send_batch_size: 50
  batch/metrics:
    timeout: 60s
  resourcedetection:
    detectors:
      - env
      - ecs
      - ec2
  resource:
    attributes:
      - key: TaskDefinitionFamily
        from_attribute: aws.ecs.task.family
        action: insert
      - key: aws.ecs.task.family
        action: delete
      - key: InstanceId
        from_attribute: host.id
        action: insert
      - key: host.id
        action: delete
      - key: TaskARN
        from_attribute: aws.ecs.task.arn
        action: insert
      - key: aws.ecs.task.arn
        action: delete
      - key: TaskDefinitionRevision
        from_attribute: aws.ecs.task.revision
        action: insert
      - key: aws.ecs.task.revision
        action: delete
      - key: LaunchType
        from_attribute: aws.ecs.launchtype
        action: insert
      - key: aws.ecs.launchtype
        action: delete
      - key: ClusterARN
        from_attribute: aws.ecs.cluster.arn
        action: insert
      - key: aws.ecs.cluster.arn
        action: delete
      - key: cloud.provider
        action: delete
      - key: cloud.platform
        action: delete
      - key: cloud.account.id
        action: delete
      - key: cloud.region
        action: delete
      - key: cloud.availability_zone
        action: delete
      - key: aws.log.group.names
        action: delete
      - key: aws.log.group.arns
        action: delete
      - key: aws.log.stream.names
        action: delete
      - key: host.image.id
        action: delete
      - key: host.name
        action: delete
      - key: host.type
        action: delete

exporters:
  awsxray:
    indexed_attributes: ["otel.resource.aws.ecs.cluster.arn"]

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resourcedetection, batch/traces]
      exporters: [awsxray]

  extensions: [health_check]

`;

export class GoObservabilityDemoStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // VPC
    const vpc = new ec2.Vpc(this, "VPC", {
      maxAzs: 2,
      natGateways: 1, // Minimal setup - single NAT gateway
    });

    // RDS Database
    const database = new rds.DatabaseInstance(this, "Database", {
      engine: rds.DatabaseInstanceEngine.postgres({
        version: rds.PostgresEngineVersion.VER_15,
      }),
      instanceType: ec2.InstanceType.of(
        ec2.InstanceClass.T3,
        ec2.InstanceSize.MICRO
      ),
      vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS,
      },
      databaseName: "observability_demo",
      credentials: rds.Credentials.fromGeneratedSecret("postgres"),
      deletionProtection: false,
      backupRetention: cdk.Duration.days(7),
      deleteAutomatedBackups: true,
    });

    // ECS Cluster
    const cluster = new ecs.Cluster(this, "Cluster", {
      vpc,
      containerInsights: true,
    });

    // CloudWatch Log Group
    const logGroup = new logs.LogGroup(this, "LogGroup", {
      logGroupName: "/ecs/go-observability-demo",
      retention: logs.RetentionDays.ONE_WEEK,
    });

    // IAM Role for ECS Task
    const taskRole = new iam.Role(this, "TaskRole", {
      assumedBy: new iam.ServicePrincipal("ecs-tasks.amazonaws.com"),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName(
          "CloudWatchAgentServerPolicy"
        ),
      ],
    });

    // Add custom policy for CloudWatch metrics
    taskRole.addToPolicy(
      new iam.PolicyStatement({
        effect: iam.Effect.ALLOW,
        actions: [
          "cloudwatch:PutMetricData",
          "cloudwatch:GetMetricStatistics",
          "cloudwatch:ListMetrics",
          "logs:PutLogEvents",
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:DescribeLogStreams",
          "logs:DescribeLogGroups",
          "xray:PutTraceSegments",
          "xray:PutTelemetryRecords",
          "xray:GetSamplingRules",
          "xray:GetSamplingTargets",
          "xray:GetSamplingStatisticSummaries",
          "ecr:GetAuthorizationToken",
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
        ],
        resources: ["*"],
      })
    );

    // ECS Service with Application Load Balancer

    const service = new ecs_patterns.ApplicationLoadBalancedFargateService(
      this,
      "Service",
      {
        cluster,
        taskImageOptions: {
          image: ecs.ContainerImage.fromEcrRepository(
            ecr.Repository.fromRepositoryName(
              this,
              "GoObservabilityDemoRepository",
              "go-observability-demo"
            ),
            "0.0.1"
          ), // Placeholder - will be updated
          containerPort: 8080,
          environment: {
            DB_HOST: database.instanceEndpoint.hostname,
            DB_PORT: "5432",
            DB_NAME: "observability_demo",
            DB_USER: "postgres",
            AWS_REGION: this.region,
            OTEL_EXPORTER_OTLP_ENDPOINT: "http://localhost:4318",
            OTEL_TRACES_SAMPLER: "parentbased_traceidratio",
            OTEL_TRACES_SAMPLER_ARG: "1",
          },
          secrets: {
            DB_PASSWORD: ecs.Secret.fromSecretsManager(
              database.secret!,
              "password"
            ),
          },
          logDriver: ecs.LogDrivers.awsLogs({
            streamPrefix: "go-observability-demo",
            logGroup,
          }),
          taskRole,
        },
        desiredCount: 1,
        cpu: 256,
        memoryLimitMiB: 512,
        publicLoadBalancer: true,
        healthCheckGracePeriod: cdk.Duration.seconds(60),
      }
    );

    service.targetGroup.configureHealthCheck({
      path: "/health",
      healthyHttpCodes: "200",
      interval: cdk.Duration.seconds(30),
      timeout: cdk.Duration.seconds(5),
      healthyThresholdCount: 2,
      unhealthyThresholdCount: 3,
    });

    service.taskDefinition.addContainer("aws-otel-collector", {
      image: ecs.ContainerImage.fromRegistry(
        "public.ecr.aws/aws-observability/aws-otel-collector:v0.42.0"
      ),
      environment: {
        AOT_CONFIG_CONTENT: awsOtelCollectorConfig,
      },
      essential: true,
      portMappings: [
        {
          containerPort: 4317,
          hostPort: 4317,
          protocol: ecs.Protocol.TCP,
        },
        {
          containerPort: 4318,
          hostPort: 4318,
          protocol: ecs.Protocol.TCP,
        },
      ],
      logging: ecs.LogDriver.awsLogs({
        logRetention: logs.RetentionDays.ONE_WEEK,
        streamPrefix: "/aws-otel-collector",
      }),
    });

    // Allow ECS service to connect to RDS
    database.connections.allowFrom(service.service, ec2.Port.tcp(5432));
    database.secret?.grantRead(service.taskDefinition.taskRole);

    // CloudWatch Dashboard
    const dashboard = new cdk.aws_cloudwatch.Dashboard(this, "Dashboard", {
      dashboardName: "GoObservabilityDemo-Dashboard",
    });

    // Add widgets to dashboard
    dashboard.addWidgets(
      new cdk.aws_cloudwatch.GraphWidget({
        title: "ALB Request Count",
        left: [
          new cdk.aws_cloudwatch.Metric({
            namespace: "AWS/ApplicationELB",
            metricName: "RequestCount",
            dimensionsMap: {
              LoadBalancer: service.loadBalancer.loadBalancerFullName,
            },
            statistic: "Sum",
            period: cdk.Duration.seconds(30),
          }),
        ],
        width: 12,
        height: 6,
      }),
      new cdk.aws_cloudwatch.GraphWidget({
        title: "ALB Target Response Time",
        left: [
          new cdk.aws_cloudwatch.Metric({
            namespace: "AWS/ApplicationELB",
            metricName: "TargetResponseTime",
            dimensionsMap: {
              LoadBalancer: service.loadBalancer.loadBalancerFullName,
            },
            statistic: "Average",
            period: cdk.Duration.seconds(30),
          }),
        ],
        width: 12,
        height: 6,
      }),
      new cdk.aws_cloudwatch.GraphWidget({
        title: "ECS Service Metrics",
        left: [
          new cdk.aws_cloudwatch.Metric({
            namespace: "AWS/ECS",
            metricName: "CPUUtilization",
            dimensionsMap: {
              ServiceName: service.service.serviceName,
              ClusterName: cluster.clusterName,
            },
          }),
          new cdk.aws_cloudwatch.Metric({
            namespace: "AWS/ECS",
            metricName: "MemoryUtilization",
            dimensionsMap: {
              ServiceName: service.service.serviceName,
              ClusterName: cluster.clusterName,
            },
          }),
        ],
        width: 12,
        height: 6,
      }),
      new cdk.aws_cloudwatch.GraphWidget({
        title: "RDS Database Metrics",
        left: [
          new cdk.aws_cloudwatch.Metric({
            namespace: "AWS/RDS",
            metricName: "CPUUtilization",
            dimensionsMap: {
              DBInstanceIdentifier: database.instanceIdentifier,
            },
          }),
        ],
        width: 12,
        height: 6,
      }),
      new cdk.aws_cloudwatch.GraphWidget({
        title: "Created Coffee Orders Metrics",
        left: [
          new cdk.aws_cloudwatch.Metric({
            namespace: "GoObservabilityDemo/Application",
            metricName: "CreatedCoffeeOrders_Total",
            statistic: "Sum",
            period: cdk.Duration.seconds(30),
          }),
        ],
        width: 12,
        height: 6,
      })
    );

    // CloudWatch Alarms
    new cdk.aws_cloudwatch.Alarm(this, "HighErrorRate", {
      metric: new cdk.aws_cloudwatch.Metric({
        namespace: "AWS/ApplicationELB",
        metricName: "HTTPCode_Target_5XX_Count",
        dimensionsMap: {
          LoadBalancer: service.loadBalancer.loadBalancerFullName,
        },
        statistic: "Sum",
        period: cdk.Duration.minutes(10),
      }),
      threshold: 5,
      evaluationPeriods: 1,
      alarmDescription: "High error rate detected",
    });

    new cdk.aws_cloudwatch.Alarm(this, "HighResponseTime", {
      metric: new cdk.aws_cloudwatch.Metric({
        namespace: "AWS/ApplicationELB",
        metricName: "TargetResponseTime",
        dimensionsMap: {
          LoadBalancer: service.loadBalancer.loadBalancerFullName,
        },
        statistic: "Maximum",
        period: cdk.Duration.seconds(30),
      }),
      threshold: 2,
      evaluationPeriods: 1,
      alarmDescription: "High response time detected",
    });

    new cdk.aws_cloudwatch.Alarm(this, "HighCPU", {
      metric: new cdk.aws_cloudwatch.Metric({
        namespace: "AWS/ECS",
        metricName: "CPUUtilization",
        dimensionsMap: {
          ServiceName: service.service.serviceName,
          ClusterName: cluster.clusterName,
        },
        statistic: "Maximum",
        period: cdk.Duration.seconds(30),
      }),
      threshold: 80,
      evaluationPeriods: 1,
      alarmDescription: "High CPU utilization detected",
    });

    new cdk.aws_cloudwatch.Alarm(this, "TooManyCreatedCoffeeOrders", {
      metric: new cdk.aws_cloudwatch.Metric({
        namespace: "GoObservabilityDemo/Application",
        metricName: "CreatedCoffeeOrders_Total",
        statistic: "Sum",
        period: cdk.Duration.minutes(10),
      }),
      threshold: 5,
      evaluationPeriods: 1,
      alarmDescription: "Too many created coffee orders detected",
    });

    // Outputs
    new cdk.CfnOutput(this, "LoadBalancerDNS", {
      value: service.loadBalancer.loadBalancerDnsName,
      description: "Load Balancer DNS Name",
    });

    new cdk.CfnOutput(this, "DatabaseEndpoint", {
      value: database.instanceEndpoint.hostname,
      description: "RDS Instance Endpoint",
    });

    new cdk.CfnOutput(this, "ECSClusterName", {
      value: cluster.clusterName,
      description: "ECS Cluster Name",
    });

    new cdk.CfnOutput(this, "ECSServiceName", {
      value: service.service.serviceName,
      description: "ECS Service Name",
    });

    new cdk.CfnOutput(this, "CloudWatchDashboard", {
      value: `https://${this.region}.console.aws.amazon.com/cloudwatch/home?region=${this.region}#dashboards:name=GoObservabilityDemo-Dashboard`,
      description: "CloudWatch Dashboard URL",
    });
  }
}
