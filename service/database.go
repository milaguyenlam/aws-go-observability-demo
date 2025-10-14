package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/codes"
)

// Database wraps pgxpool.Pool with additional functionality
type Database struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// Open initializes a new database connection with pgxpool and tracing
func Open(ctx context.Context, dsn string, logger *slog.Logger) (*Database, error) {
	parsedConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Configure connection pool
	parsedConfig.MaxConns = 25
	parsedConfig.MinConns = 5
	parsedConfig.MaxConnLifetime = 5 * time.Minute
	parsedConfig.MaxConnIdleTime = 1 * time.Minute

	// Enable OpenTelemetry tracing
	parsedConfig.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, parsedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{
		pool:   pool,
		logger: logger,
	}, nil
}

// Close closes the database connection pool
func (db *Database) Close() {
	db.pool.Close()
}

// Ping checks the database connection
func (db *Database) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// initSchema initializes the database schema
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

	_, err := app.db.pool.Exec(ctx, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "Schema initialized successfully")
	return nil
}

// GetCoffeeOrder retrieves a coffee order by ID
func (db *Database) GetCoffeeOrder(ctx context.Context, id int) (*CoffeeOrder, error) {
	query := "SELECT id, user_name, coffee_type, created_at FROM coffee_orders WHERE id = $1"

	var order CoffeeOrder
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.UserName,
		&order.CoffeeType,
		&order.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &order, nil
}

// CreateCoffeeOrder creates a new coffee order
func (db *Database) CreateCoffeeOrder(ctx context.Context, order CreateCoffeeOrder) (*CoffeeOrder, error) {
	query := "INSERT INTO coffee_orders (user_name, coffee_type) VALUES ($1, $2) RETURNING id, user_name, coffee_type, created_at"

	var createdOrder CoffeeOrder

	err := db.pool.QueryRow(ctx, query, order.UserName, order.CoffeeType).Scan(&createdOrder.ID, &createdOrder.UserName, &createdOrder.CoffeeType, &createdOrder.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &createdOrder, nil
}

// CreateCoffeeOrder creates a new coffee order
func (db *Database) CreateCoffeeOrderInOneHour(ctx context.Context, order CreateCoffeeOrder) (*CoffeeOrder, error) {
	query := "INSERT INTO coffee_orders (user_name, coffee_type, created_at) VALUES ($1, $2, $3) RETURNING id, user_name, coffee_type, created_at"

	var createdOrder CoffeeOrder
	err := db.pool.QueryRow(ctx, query, order.UserName, order.CoffeeType, time.Now().Add(2*time.Hour)).Scan(&createdOrder.ID, &createdOrder.UserName, &createdOrder.CoffeeType, &createdOrder.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &createdOrder, nil
}
