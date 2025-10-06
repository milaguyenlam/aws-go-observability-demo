package main

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// CreateCoffeeOrder
type CreateCoffeeOrder struct {
	UserName   string `json:"user_name"`
	CoffeeType string `json:"coffee_type"`
}

// HealthResponse represents the health check response

// CoffeeOrder model
type CoffeeOrder struct {
	ID         int       `json:"id"`
	UserName   string    `json:"user_name"`
	CoffeeType string    `json:"coffee_type"`
	CreatedAt  time.Time `json:"created_at"`
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// Database wraps pgxpool.Pool with additional functionality
type Database struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// App represents the application instance
type App struct {
	db      *Database
	logger  *zap.Logger
	metrics *CloudWatchMetrics
	region  string
	tracer  trace.Tracer
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
