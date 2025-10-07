package main

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// initTracing initializes OpenTelemetry tracing for distributed tracing and observability.
// This function sets up the complete tracing infrastructure including resource identification,
// OTLP exporter configuration, trace provider setup, and global propagators.
// Returns a tracer instance, cleanup function, and any initialization errors.
func initTracing() (trace.Tracer, func(), error) {
	// Create resource with service metadata for trace identification
	// The resource provides context about the service generating traces, including
	// service name, version, and deployment environment. This metadata helps
	// distinguish traces from different service instances and versions.
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("go-observability-demo"), // Identifies this service in traces
			semconv.ServiceVersion("1.0.0"),              // Version for deployment tracking
			semconv.DeploymentEnvironment("prod"),        // Environment context (dev/staging/prod)
		),
	)
	if err != nil {
		return nil, nil, err
	}

	// Create OTLP HTTP exporter for sending traces to AWS X-Ray
	// OTLP (OpenTelemetry Protocol) is the standard protocol for telemetry data.
	// This exporter sends traces over HTTP to AWS X-Ray, which provides
	// distributed tracing visualization and analysis capabilities.
	// Note: WithInsecure() is used for demo purposes - use HTTPS in production
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithInsecure(), // Use HTTPS in production for security
	)
	if err != nil {
		return nil, nil, err
	}

	// Create trace provider with batching and sampling configuration
	// The trace provider manages the lifecycle of traces and controls how they're
	// processed and exported. Batching improves performance by grouping multiple
	// spans together, while sampling controls which traces are recorded.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),                // Batch spans for efficient export
		sdktrace.WithResource(res),                    // Associate service metadata
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Record all traces (adjust for production)
	)

	// Set global tracer provider for the application
	// This makes the tracer provider available throughout the application
	// without needing to pass it around explicitly
	otel.SetTracerProvider(tp)

	// Set global text map propagator for trace context
	// Propagators handle the serialization/deserialization of trace context
	// across service boundaries (HTTP headers, gRPC metadata, etc.)
	// TraceContext handles W3C trace context, Baggage handles additional metadata
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C trace context standard
		propagation.Baggage{},      // Additional trace metadata
	))

	// Get tracer instance for creating spans
	// The tracer is used throughout the application to create spans,
	// which represent individual operations within a trace
	tracer := otel.Tracer("go-observability-demo")

	// Cleanup function to properly shutdown tracing infrastructure
	// This ensures all pending traces are flushed before the application exits
	// and resources are properly cleaned up
	cleanup := func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}

	return tracer, cleanup, nil
}
