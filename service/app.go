package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// healthHandler handles health check requests
func (app *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	ctx := r.Context()

	var dbStatus string
	if err := app.db.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
		app.logger.Error("Database health check failed", zap.Error(err))
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

// getCoffeeOrderHandler handles GET requests for coffee orders
func (app *App) getCoffeeOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := app.tracer.Start(r.Context(), "receiveCoffeeOrderHandler")
	defer span.End()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid coffee order ID")
		app.returnErrorResponse(w, r, "Invalid coffee order ID", err)
		return
	}

	order, err := app.db.GetCoffeeOrder(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		app.returnErrorResponse(w, r, "Failed to get coffee order", err)
		return
	}

	span.SetAttributes(
		attribute.Int("coffee_order.id", order.ID),
		attribute.String("coffee_order.user_name", order.UserName),
		attribute.String("coffee_order.coffee_type", order.CoffeeType),
	)
	span.SetStatus(codes.Ok, "Coffee order retrieved successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// createCoffeeOrderHandler handles POST requests for creating coffee orders
func (app *App) createCoffeeOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := app.tracer.Start(r.Context(), "createCoffeeOrderHandler")
	defer span.End()

	var coffeeOrder CreateCoffeeOrder
	if err := json.NewDecoder(r.Body).Decode(&coffeeOrder); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		app.returnErrorResponse(w, r, "Invalid JSON", err)
		return
	}

	span.SetAttributes(
		attribute.String("coffee_order.user_name", coffeeOrder.UserName),
		attribute.String("coffee_order.coffee_type", coffeeOrder.CoffeeType),
	)

	order, err := app.db.CreateCoffeeOrder(ctx, coffeeOrder)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		app.returnErrorResponse(w, r, "Failed to create coffee order", err)
		return
	}
	span.SetStatus(codes.Ok, "Coffee order created successfully")

	go app.metrics.sendCreatedCoffeeOrderMetrics(
		order.UserName,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

// logError logs errors and sends error responses
func (app *App) returnErrorResponse(w http.ResponseWriter, r *http.Request, message string, err error) {
	requestID := getRequestID(r.Context())
	traceID := trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()

	app.logger.Error(message,
		zap.String("request_id", requestID),
		zap.String("trace_id", traceID),
		zap.Error(err),
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
	)

	errorResponse := ErrorResponse{
		Error: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(errorResponse)
}
