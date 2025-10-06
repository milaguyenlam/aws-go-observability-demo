package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Load configuration
	config := LoadConfig()

	// Initialize OpenTelemetry
	tracer, cleanup, err := initTracing()
	if err != nil {
		logger.Fatal("Failed to initialize tracing", zap.Error(err))
	}
	defer cleanup()

	// Initialize database connection with pgx
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := Open(context.Background(), dsn, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize AWS CloudWatch
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region),
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
		region:  config.Region,
		tracer:  tracer,
	}

	// Initialize database schema
	if err := app.initSchema(); err != nil {
		logger.Fatal("Failed to initialize database schema", zap.Error(err))
	}

	router := setupRoutes(app)

	// Start server
	logger.Info("Starting server", zap.String("port", config.Port))

	if err := http.ListenAndServe(":"+config.Port, router); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}
