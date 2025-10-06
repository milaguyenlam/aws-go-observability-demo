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
	router.Route("/coffee", func(r chi.Router) {
		r.Get("/{id}", app.getCoffeeOrderHandler)
		r.Post("/", app.createCoffeeOrderHandler)
	})

	return router
}
