import "source-map-support/register";
import * as cdk from "aws-cdk-lib";
import { GoObservabilityDemoStack } from "../lib/go-observability-demo-stack";

const app = new cdk.App();

new GoObservabilityDemoStack(app, "GoObservabilityDemoStack", {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: process.env.CDK_DEFAULT_REGION || "eu-central-1",
  },
  description: "Go Observability Demo - Minimal AWS CDK Setup",
});

app.synth();
