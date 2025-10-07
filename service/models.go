package main

import (
	"time"
)

// CreateCoffeeOrder
type CreateCoffeeOrder struct {
	UserName   string `json:"user_name"`
	CoffeeType string `json:"coffee_type"`
}

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
