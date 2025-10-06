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

// initTracing initializes OpenTelemetry tracing
func initTracing() (trace.Tracer, func(), error) {
	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("go-observability-demo"),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment("demo"),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	// Create OTLP exporter (for AWS X-Ray)
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithInsecure(), // Use HTTPS in production
	)
	if err != nil {
		return nil, nil, err
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Get tracer
	tracer := otel.Tracer("go-observability-demo")

	// Cleanup function
	cleanup := func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}

	return tracer, cleanup, nil
}
