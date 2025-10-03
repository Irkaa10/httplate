package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := log.New(os.Stdout, "[HTTP] ", log.LstdFlags|log.Lshortfile)

	config := &Config{
		Port:         getEnv("PORT", "8080"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	srv := NewServer(logger, config)

	httpServer := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      srv,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	go func() {
		logger.Printf("Starting server on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server stopped gracefully")
}

func NewServer(logger *log.Logger, config *Config) http.Handler {
	mux := http.NewServeMux()

	registerRoutes(mux, logger, config)

	var handler http.Handler = mux
	handler = loggingMiddleware(logger, handler)
	handler = recoveryMiddleware(logger, handler)

	return handler
}

type Config struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func registerRoutes(mux *http.ServeMux, logger *log.Logger, config *Config) {
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", homeHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Write([]byte("Welcome to the API"))
}
