package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	config := LoadConfig()

	// Initialize OpenTelemetry
	tracer, cleanup, err := initTracing()
	if err != nil {
		logger.Error("Failed to initialize tracing", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	// Initialize database connection with pgx
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := Open(context.Background(), dsn, logger)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize AWS CloudWatch
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region),
	})
	if err != nil {
		logger.Error("Failed to create AWS session", "error", err)
		os.Exit(1)
	}
	cw := cloudwatch.New(sess)

	// Create metrics instance
	metrics := &CloudWatchMetrics{
		cw:     cw,
		logger: logger,
		tracer: tracer,
	}

	// Create app instance
	app := &App{
		db:      db,
		logger:  logger,
		metrics: metrics,
		region:  config.Region,
		tracer:  tracer,
	}

	// Initialize database schema
	if err := app.initSchema(); err != nil {
		logger.Error("Failed to initialize database schema", "error", err)
		os.Exit(1)
	}

	router := setupRoutes(app)

	// Start server
	logger.Info("Starting server", "port", config.Port)

	if err := http.ListenAndServe(":"+config.Port, router); err != nil {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
