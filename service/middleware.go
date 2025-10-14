package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Context key type for request ID
type contextKey string

const requestIDKey contextKey = "request_id"

// loggingMiddleware logs HTTP requests and responses and adds request ID to response headers
func (app *App) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := middleware.GetReqID(r.Context())
		traceID := trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r = r.WithContext(ctx)

		// Log request
		app.logger.Info("Request started",
			"request_id", requestID,
			"trace_id", traceID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		// Log response
		duration := time.Since(start)
		app.logger.Info("Request completed",
			"request_id", requestID,
			"trace_id", traceID,
			"status_code", wrapped.statusCode,
			"duration", duration,
		)
	})
}

// tracingMiddleware handles OpenTelemetry tracing and adds trace ID to response headers
func (app *App) tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from headers
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		// Create span
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := app.tracer.Start(ctx, spanName)
		defer span.End()

		// Add span attributes
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.user_agent", r.UserAgent()),
			attribute.String("http.remote_addr", r.RemoteAddr),
		)

		// Add request ID to span
		if requestID := getRequestID(ctx); requestID != "" {
			span.SetAttributes(attribute.String("request.id", requestID))
		}

		// Update request context
		r = r.WithContext(ctx)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		// Set span status and attributes
		span.SetAttributes(attribute.Int("http.status_code", wrapped.statusCode))
		if wrapped.statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", wrapped.statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	})
}

// metricsMiddleware handles CloudWatch metrics
func (app *App) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)

		// Send metrics to CloudWatch
		routePattern := chi.RouteContext(ctx).RoutePattern()
		go app.metrics.sendRouteMetrics(ctx, routePattern, duration)
	})
}

// responseHeadersMiddleware adds request ID and trace ID to response headers
func (app *App) responseHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract request ID from context
		requestID := getRequestID(r.Context())
		if requestID != "" {
			w.Header().Set("X-Request-ID", requestID)
		}

		// Extract trace ID from OpenTelemetry context
		span := trace.SpanFromContext(r.Context())
		if span.SpanContext().IsValid() {
			traceID := span.SpanContext().TraceID().String()
			w.Header().Set("X-Trace-ID", traceID)
		}

		next.ServeHTTP(w, r)
	})
}

// getRequestID extracts request ID from context
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}
