package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/lumina/gateway/internal/api"
	"github.com/lumina/gateway/internal/auth"
	"github.com/lumina/gateway/internal/cache"
	"github.com/lumina/gateway/internal/config"
	"github.com/lumina/gateway/internal/database"
	"github.com/lumina/gateway/internal/logging"
	"github.com/lumina/gateway/internal/proxy"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set up structured logging
	logLevel := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	slog.Info("starting Lumina Gateway", "port", cfg.Port)

	// Initialize database connection
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize Redis cache
	redisCache, err := cache.New(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisCache.Close()

	// Initialize OpenSearch logging
	logPipeline, err := logging.New(cfg.OpenSearchURL)
	if err != nil {
		slog.Error("failed to connect to OpenSearch", "error", err)
		os.Exit(1)
	}
	defer logPipeline.Close()

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret)

	// Initialize services
	keyService := auth.NewKeyService(db, redisCache, cfg.EncryptionKey)
	proxyHandler := proxy.NewHandler(keyService, logPipeline)
	apiHandler := api.NewHandler(db, keyService, jwtManager)
	apiHandler.SetLogPipeline(logPipeline)

	// Set up router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API routes (dashboard management)
	r.Route("/api", func(r chi.Router) {
		// Public routes
		r.Post("/auth/login", apiHandler.Login)
		r.Post("/auth/register", apiHandler.Register)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(auth.JWTMiddleware(jwtManager))

			r.Post("/auth/logout", apiHandler.Logout)
			r.Get("/auth/me", apiHandler.Me)

			// Key management
			r.Route("/keys", func(r chi.Router) {
				r.Get("/", apiHandler.ListKeys)
				r.Post("/", apiHandler.CreateKey)
				r.Get("/{id}", apiHandler.GetKey)
				r.Put("/{id}", apiHandler.UpdateKey)
				r.Delete("/{id}", apiHandler.RevokeKey)
			})

			// Provider management (account-level API keys)
			r.Route("/providers", func(r chi.Router) {
				r.Get("/", apiHandler.ListProviders)
				r.Post("/", apiHandler.SetProvider)
				r.Delete("/{provider}", apiHandler.RemoveProvider)
			})

			// Statistics
			r.Get("/stats/overview", apiHandler.GetOverview)
			r.Get("/stats/daily", apiHandler.GetDailyStats)

			// Logs
			r.Get("/logs", apiHandler.SearchLogs)
			r.Get("/logs/{id}", apiHandler.GetLog)
		})
	})

	// LLM Proxy routes (OpenAI compatible)
	r.Route("/v1", func(r chi.Router) {
		r.Post("/chat/completions", proxyHandler.ChatCompletions)
		r.Post("/completions", proxyHandler.Completions)
		r.Post("/embeddings", proxyHandler.Embeddings)
	})

	// Anthropic proxy routes
	r.Route("/anthropic", func(r chi.Router) {
		r.Post("/v1/messages", proxyHandler.AnthropicMessages)
	})

	// Create server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}
