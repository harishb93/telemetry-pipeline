// Package api provides HTTP API server implementation for telemetry data access
package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/harishb93/telemetry-pipeline/internal/collector"
)

// Server represents the HTTP API server
type Server struct {
	collector  *collector.Collector
	httpServer *http.Server
	port       string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string
}

// NewServer creates a new API server instance
func NewServer(collector *collector.Collector, config ServerConfig) *Server {
	return &Server{
		collector: collector,
		port:      config.Port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	router := mux.NewRouter()

	// Create handlers
	handlers := NewHandlers(s.collector)

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/gpus", handlers.GetGPUs).Methods("GET")
	v1.HandleFunc("/gpus/{id}/telemetry", handlers.GetTelemetry).Methods("GET")
	v1.HandleFunc("/hosts", handlers.GetHosts).Methods("GET")
	v1.HandleFunc("/hosts/{hostname}/gpus", handlers.GetHostGPUs).Methods("GET")

	// Health endpoint
	router.HandleFunc("/health", handlers.Health).Methods("GET")

	// Swagger documentation endpoint
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// CORS middleware
	router.Use(s.corsMiddleware)

	// Request logging middleware
	router.Use(s.loggingMiddleware)

	s.httpServer = &http.Server{
		Addr:         ":" + s.port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("API server starting on port %s", s.port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/", s.port)

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("API server stopping...")
	return s.httpServer.Shutdown(ctx)
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapper.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
