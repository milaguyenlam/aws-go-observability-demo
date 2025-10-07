package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func setupRoutes(app *App) *chi.Mux {
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(app.tracingMiddleware)
	router.Use(app.loggingMiddleware)
	router.Use(app.metricsMiddleware)
	router.Use(app.responseHeadersMiddleware)

	// Routes
	router.Get("/health", app.healthHandler)

	// Standard coffee routes
	router.Route("/coffee", func(r chi.Router) {
		r.Get("/{id}", app.getCoffeeOrderHandler)
	})

	// Person-specific coffee order endpoints
	router.Post("/make-coffee-tom", app.createCoffeeOrderTomHandler)
	router.Post("/make-coffee-honza", app.createCoffeeOrderHonzaHandler)
	router.Post("/make-coffee-marek", app.createCoffeeOrderMarekHandler)
	router.Post("/make-coffee-viking", app.createCoffeeOrderVikingHandler)
	router.Post("/make-coffee-matus", app.createCoffeeOrderMatusHandler)
	router.Post("/make-coffee-mila", app.createCoffeeOrderMilaHandler)

	return router
}
