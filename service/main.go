package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Context key type for request ID
type contextKey string

const requestIDKey contextKey = "request_id"

// CloudWatchMetrics handles CloudWatch metrics operations
type CloudWatchMetrics struct {
	cw     *cloudwatch.CloudWatch
	logger *zap.Logger
}

// App represents the application instance
type App struct {
	db      *sql.DB
	logger  *zap.Logger
	metrics *CloudWatchMetrics
	region  string
	tracer  trace.Tracer
}

// CoffeeOrder model
type CoffeeOrder struct {
	ID         int       `json:"id"`
	UserName   string    `json:"user_name"`
	CoffeeType string    `json:"coffee_type"`
	CreatedAt  time.Time `json:"created_at"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database"`
	Region    string    `json:"region"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id"`
	Timestamp string `json:"timestamp"`
}

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Get configuration from environment
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "observability_demo")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "password")
	region := getEnv("AWS_REGION", "eu-central-1")

	// Initialize OpenTelemetry
	tracer, cleanup, err := initTracing(region)
	if err != nil {
		logger.Fatal("Failed to initialize tracing", zap.Error(err))
	}
	defer cleanup()

	// Initialize database connection
	db, err := initDB(dbHost, dbPort, dbName, dbUser, dbPassword)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize AWS CloudWatch
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		logger.Fatal("Failed to create AWS session", zap.Error(err))
	}
	cw := cloudwatch.New(sess)

	// Create metrics instance
	metrics := &CloudWatchMetrics{
		cw:     cw,
		logger: logger,
	}

	// Create app instance
	app := &App{
		db:      db,
		logger:  logger,
		metrics: metrics,
		region:  region,
		tracer:  tracer,
	}

	// Initialize database schema
	if err := app.initSchema(); err != nil {
		logger.Fatal("Failed to initialize database schema", zap.Error(err))
	}

	// Setup routes
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(app.loggingMiddleware)
	router.Use(app.metricsMiddleware)
	router.Use(app.tracingMiddleware)

	// Routes
	router.Get("/health", app.healthHandler)
	router.Route("/coffee", func(r chi.Router) {
		r.Get("/{id}", app.receiveCoffeeOrderHandler)
		r.Post("/", app.createCoffeeOrderHandler)
	})

	// Demo endpoints for observability
	router.Route("/demo", func(r chi.Router) {
		r.Get("/slow-query", app.slowQueryHandler)
		r.Get("/error", app.errorHandler)
		r.Get("/memory-leak", app.memoryLeakHandler)
		r.Get("/high-cpu", app.highCPUHandler)
	})

	// Start server
	port := getEnv("PORT", "8080")
	logger.Info("Starting server", zap.String("port", port))

	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

// Initialize OpenTelemetry tracing
func initTracing(region string) (trace.Tracer, func(), error) {
	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("go-observability-demo"),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment("demo"),
			attribute.String("aws.region", region),
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

// Database initialization
func initDB(host, port, dbname, user, password string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// Initialize database schema
func (app *App) initSchema() error {
	ctx, span := app.tracer.Start(context.Background(), "initSchema")
	defer span.End()

	query := `
	CREATE TABLE IF NOT EXISTS coffee_orders (
		id SERIAL PRIMARY KEY,
		user_name VARCHAR(255) NOT NULL,
		coffee_type VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := app.db.ExecContext(ctx, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "Schema initialized successfully")
	return nil
}

// Middleware
func (app *App) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := middleware.GetReqID(r.Context())

		// Add request ID to context if not already set
		if requestID == "" {
			requestID = generateRequestID()
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			r = r.WithContext(ctx)
		} else {
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			r = r.WithContext(ctx)
		}

		// Log request
		app.logger.Info("Request started",
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		// Log response
		duration := time.Since(start)
		app.logger.Info("Request completed",
			zap.String("request_id", requestID),
			zap.Int("status_code", wrapped.statusCode),
			zap.Duration("duration", duration),
		)
	})
}

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
		if requestID := getRequestID(ctx); requestID != "unknown" {
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

func (app *App) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		// Send metrics to CloudWatch
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		go app.metrics.sendMetrics(routePattern, wrapped.statusCode, duration)
	})
}

// Handlers
func (app *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := app.tracer.Start(r.Context(), "healthHandler")
	defer span.End()

	// Check database connectivity
	var dbStatus string
	if err := app.db.PingContext(ctx); err != nil {
		dbStatus = "unhealthy"
		app.logger.Error("Database health check failed", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		dbStatus = "healthy"
		span.SetStatus(codes.Ok, "Health check passed")
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Database:  dbStatus,
		Region:    app.region,
	}

	span.SetAttributes(
		attribute.String("health.status", response.Status),
		attribute.String("health.database", response.Database),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// / coffee order handlers
func (app *App) receiveCoffeeOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := app.tracer.Start(r.Context(), "receiveCoffeeOrderHandler")
	defer span.End()

	start := time.Now()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid coffee order ID")
		app.logError(w, r, "Invalid coffee order ID", err)
		return
	}

	query := "SELECT id, user_name, coffee_type, created_at FROM coffee_orders WHERE id = $1"
	rows, err := app.db.QueryContext(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		app.logError(w, r, "Failed to query coffee orders", err)
		return
	}
	defer rows.Close()

	var coffeeOrders []CoffeeOrder
	for rows.Next() {
		var coffeeOrder CoffeeOrder
		if err := rows.Scan(&coffeeOrder.ID, &coffeeOrder.UserName, &coffeeOrder.CoffeeType, &coffeeOrder.CreatedAt); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			app.logError(w, r, "Failed to scan coffee order", err)
			return
		}
		coffeeOrders = append(coffeeOrders, coffeeOrder)
	}

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int("coffee_orders.count", len(coffeeOrders)),
		attribute.Float64("db.query.duration", duration.Seconds()),
	)
	span.SetStatus(codes.Ok, "Coffee orders retrieved successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(coffeeOrders)
}

func (app *App) createCoffeeOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := app.tracer.Start(r.Context(), "createCoffeeOrderHandler")
	defer span.End()

	var coffeeOrder CoffeeOrder
	if err := json.NewDecoder(r.Body).Decode(&coffeeOrder); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		app.logError(w, r, "Invalid JSON", err)
		return
	}

	span.SetAttributes(
		attribute.String("coffee_order.user_name", coffeeOrder.UserName),
		attribute.String("coffee_order.coffee_type", coffeeOrder.CoffeeType),
	)

	start := time.Now()
	query := "INSERT INTO coffee_orders (user_name, coffee_type) VALUES ($1, $2) RETURNING id, created_at"
	err := app.db.QueryRowContext(ctx, query, coffeeOrder.UserName, coffeeOrder.CoffeeType).Scan(&coffeeOrder.ID, &coffeeOrder.CreatedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Coffee order already exists")
			app.logError(w, r, "Coffee order already exists", err)
			return
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		app.logError(w, r, "Failed to create coffee order", err)
		return
	}

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int("coffee_order.id", coffeeOrder.ID),
		attribute.Float64("db.query.duration", duration.Seconds()),
	)
	span.SetStatus(codes.Ok, "Coffee order created successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(coffeeOrder)

	app.metrics.sendCreatedCoffeeOrderMetrics(
		coffeeOrder.UserName,
		coffeeOrder.CoffeeType,
	)
}

// Demo handlers for observability
func (app *App) slowQueryHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := app.tracer.Start(r.Context(), "slowQueryHandler")
	defer span.End()

	start := time.Now()

	// Simulate a slow query
	query := "SELECT pg_sleep(2), COUNT(*) FROM users"
	var count int
	err := app.db.QueryRowContext(ctx, query).Scan(nil, &count)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		app.logError(w, r, "Slow query failed", err)
		return
	}

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int("db.query.count", count),
		attribute.Float64("db.query.duration", duration.Seconds()),
	)
	span.SetStatus(codes.Ok, "Slow query completed")

	response := map[string]interface{}{
		"message":  "Slow query completed",
		"count":    count,
		"duration": duration.Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *App) errorHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := app.tracer.Start(r.Context(), "errorHandler")
	defer span.End()

	span.SetStatus(codes.Error, "Demo error triggered")
	span.SetAttributes(attribute.String("error.type", "demo_error"))

	// Send error metric to CloudWatch
	go app.metrics.sendErrorMetric("demo_error")

	app.logger.Error("Demo error triggered",
		zap.String("request_id", getRequestID(ctx)),
		zap.String("error_type", "demo_error"),
	)

	http.Error(w, "Demo error: This is intentional for testing", http.StatusInternalServerError)
}

func (app *App) memoryLeakHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := app.tracer.Start(r.Context(), "memoryLeakHandler")
	defer span.End()

	// Simulate memory leak by creating large slices
	var slices [][]byte
	for i := 0; i < 1000; i++ {
		slice := make([]byte, 1024*1024) // 1MB
		slices = append(slices, slice)
	}

	span.SetAttributes(attribute.Int("memory.slices_created", len(slices)))
	span.SetStatus(codes.Ok, "Memory leak simulation completed")

	app.logger.Warn("Memory leak simulation",
		zap.String("request_id", getRequestID(ctx)),
		zap.Int("slices_created", len(slices)),
	)

	response := map[string]interface{}{
		"message": "Memory leak simulation completed",
		"slices":  len(slices),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *App) highCPUHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := app.tracer.Start(r.Context(), "highCPUHandler")
	defer span.End()

	start := time.Now()

	// Simulate high CPU usage
	for i := 0; i < 10000000; i++ {
		_ = rand.Intn(1000)
	}

	duration := time.Since(start)
	span.SetAttributes(attribute.Float64("cpu.simulation.duration", duration.Seconds()))
	span.SetStatus(codes.Ok, "High CPU simulation completed")

	app.logger.Warn("High CPU simulation",
		zap.String("request_id", getRequestID(ctx)),
		zap.Duration("duration", duration),
	)

	response := map[string]interface{}{
		"message":  "High CPU simulation completed",
		"duration": duration.Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions
func (app *App) logError(w http.ResponseWriter, r *http.Request, message string, err error) {
	requestID := getRequestID(r.Context())

	app.logger.Error(message,
		zap.String("request_id", requestID),
		zap.Error(err),
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
	)

	// Send error metric to CloudWatch
	go app.metrics.sendErrorMetric("application_error")

	errorResponse := ErrorResponse{
		Error:     message,
		RequestID: requestID,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(errorResponse)
}

// CloudWatch metrics methods
func (m *CloudWatchMetrics) sendMetrics(endpoint string, statusCode int, duration time.Duration) {
	namespace := "GoObservabilityDemo/Application"

	metrics := []*cloudwatch.MetricDatum{
		{
			MetricName: aws.String("RequestDuration"),
			Value:      aws.Float64(duration.Seconds()),
			Unit:       aws.String("Seconds"),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("Endpoint"),
					Value: aws.String(endpoint),
				},
			},
			Timestamp: aws.Time(time.Now()),
		},
		{
			MetricName: aws.String("RequestCount"),
			Value:      aws.Float64(1),
			Unit:       aws.String("Count"),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("Endpoint"),
					Value: aws.String(endpoint),
				},
				{
					Name:  aws.String("StatusCode"),
					Value: aws.String(strconv.Itoa(statusCode)),
				},
			},
			Timestamp: aws.Time(time.Now()),
		},
	}

	_, err := m.cw.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(namespace),
		MetricData: metrics,
	})

	if err != nil {
		m.logger.Error("Failed to send CloudWatch metrics", zap.Error(err))
	}
}

func (m *CloudWatchMetrics) sendCreatedCoffeeOrderMetrics(userName, coffeeType string) {
	namespace := "GoObservabilityDemo/Application"

	metric := &cloudwatch.MetricDatum{
		MetricName: aws.String("CreatedCoffeeOrders"),
		Value:      aws.Float64(1),
		Unit:       aws.String("Count"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("UserName"),
				Value: aws.String(userName),
			},
			{
				Name:  aws.String("CoffeeType"),
				Value: aws.String(coffeeType),
			},
		},
		Timestamp: aws.Time(time.Now()),
	}

	_, err := m.cw.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(namespace),
		MetricData: []*cloudwatch.MetricDatum{metric},
	})

	if err != nil {
		m.logger.Error("Failed to send created coffee order metric to CloudWatch", zap.Error(err))
	}
}
func (m *CloudWatchMetrics) sendErrorMetric(errorType string) {
	namespace := "GoObservabilityDemo/Application"

	metric := &cloudwatch.MetricDatum{
		MetricName: aws.String("ErrorCount"),
		Value:      aws.Float64(1),
		Unit:       aws.String("Count"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("ErrorType"),
				Value: aws.String(errorType),
			},
		},
		Timestamp: aws.Time(time.Now()),
	}

	_, err := m.cw.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(namespace),
		MetricData: []*cloudwatch.MetricDatum{metric},
	})

	if err != nil {
		m.logger.Error("Failed to send error metric to CloudWatch", zap.Error(err))
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return "unknown"
}

// Response writer wrapper
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
