package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

// App represents the application instance
type App struct {
	db      *Database
	logger  *slog.Logger
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

func (app *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	ctx := r.Context()

	var dbStatus string
	if err := app.db.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
		app.logger.Error("Database health check failed", "error", err)
	} else {
		dbStatus = "healthy"
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Database:  dbStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *App) getCoffeeOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.returnErrorResponse(w, r, "Invalid coffee order ID", err)
		return
	}

	order, err := app.db.GetCoffeeOrder(ctx, id)
	if err != nil {
		app.returnErrorResponse(w, r, "Failed to get coffee order", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (app *App) createCoffeeOrderTomHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var coffeeOrder CreateCoffeeOrder
	if err := json.NewDecoder(r.Body).Decode(&coffeeOrder); err != nil {
		app.returnErrorResponse(w, r, "Invalid JSON", err)
		return
	}

	query := "INSERT INTO coffee_orders (user_name, coffee_type) VALUES ($1, $2) RETURNING id, user_name, coffee_type, created_at"

	// Add sleep
	time.Sleep(3 * time.Second)

	var createdOrder CoffeeOrder
	err := app.db.pool.QueryRow(ctx, query, coffeeOrder.UserName, coffeeOrder.CoffeeType).Scan(
		&createdOrder.ID, &createdOrder.UserName, &createdOrder.CoffeeType, &createdOrder.CreatedAt)
	if err != nil {
		app.returnErrorResponse(w, r, "Failed to create coffee order", err)
		return
	}

	go app.metrics.sendCreatedCoffeeOrderMetrics(ctx, createdOrder.CoffeeType, createdOrder.UserName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdOrder)
}

func (app *App) createCoffeeOrderHonzaHandler(w http.ResponseWriter, r *http.Request) {
	var coffeeOrder CreateCoffeeOrder
	if err := json.NewDecoder(r.Body).Decode(&coffeeOrder); err != nil {
		app.returnErrorResponse(w, r, "Invalid JSON", err)
		return
	}

	app.returnErrorResponse(w, r, "Honza's endpoint is broken", errors.New("intentional failure"))
}

func (app *App) createCoffeeOrderMarekHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var coffeeOrder CreateCoffeeOrder
	if err := json.NewDecoder(r.Body).Decode(&coffeeOrder); err != nil {
		app.returnErrorResponse(w, r, "Invalid JSON", err)
		return
	}

	var slices [][]byte
	for i := 0; i < 250; i++ {
		slice := make([]byte, 1024*1024) // 1MB each
		slices = append(slices, slice)
	}

	time.Sleep(1 * time.Second)

	// Create the coffee order after memory allocation
	order, err := app.db.CreateCoffeeOrder(ctx, coffeeOrder)
	if err != nil {
		app.returnErrorResponse(w, r, "Failed to create coffee order", err)
		return
	}

	go app.metrics.sendCreatedCoffeeOrderMetrics(ctx, order.CoffeeType, order.UserName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (app *App) createCoffeeOrderJakubHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var coffeeOrder CreateCoffeeOrder
	if err := json.NewDecoder(r.Body).Decode(&coffeeOrder); err != nil {
		app.returnErrorResponse(w, r, "Invalid JSON", err)
		return
	}

	// Viking's unnecessary select queries
	for i := 0; i < 10; i++ {
		app.db.GetCoffeeOrder(ctx, i)
	}

	// Create the actual coffee order
	order, err := app.db.CreateCoffeeOrder(ctx, coffeeOrder)
	if err != nil {
		app.returnErrorResponse(w, r, "Failed to create coffee order", err)
		return
	}

	go app.metrics.sendCreatedCoffeeOrderMetrics(ctx, order.CoffeeType, order.UserName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (app *App) createCoffeeOrderMatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var coffeeOrder CreateCoffeeOrder
	if err := json.NewDecoder(r.Body).Decode(&coffeeOrder); err != nil {
		app.returnErrorResponse(w, r, "Invalid JSON", err)
		return
	}

	// Matus always saves "borovicka" instead of the requested coffee type
	modifiedOrder := CreateCoffeeOrder{
		UserName:   coffeeOrder.UserName,
		CoffeeType: "borovicka", // Always borovicka!
	}

	order, err := app.db.CreateCoffeeOrder(ctx, modifiedOrder)
	if err != nil {
		app.returnErrorResponse(w, r, "Failed to create coffee order", err)
		return
	}

	go app.metrics.sendCreatedCoffeeOrderMetrics(ctx, order.CoffeeType, order.UserName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (app *App) createCoffeeOrderMilaHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var coffeeOrder CreateCoffeeOrder
	if err := json.NewDecoder(r.Body).Decode(&coffeeOrder); err != nil {
		app.returnErrorResponse(w, r, "Invalid JSON", err)
		return
	}

	order, err := app.db.CreateCoffeeOrderInOneHour(ctx, coffeeOrder)
	if err != nil {
		app.returnErrorResponse(w, r, "Failed to create coffee order", err)
		return
	}

	go app.metrics.sendCreatedCoffeeOrderMetrics(ctx, order.CoffeeType, order.UserName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

// returnErrorResponse logs errors and sends error responses
func (app *App) returnErrorResponse(w http.ResponseWriter, r *http.Request, message string, err error) {
	requestID := getRequestID(r.Context())
	traceID := trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()

	app.logger.Error(message,
		"request_id", requestID,
		"trace_id", traceID,
		"error", err,
		"method", r.Method,
		"path", r.URL.Path,
	)

	errorResponse := ErrorResponse{
		Error: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(errorResponse)
}
