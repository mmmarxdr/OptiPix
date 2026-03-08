package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/optipix/backend/internal/config"
	"github.com/optipix/backend/internal/handler"
	"github.com/optipix/backend/internal/middleware"
	"github.com/optipix/backend/internal/processor"
)

func main() {
	cfg := config.Load()

	processor.InitVips()
	defer processor.ShutdownVips()

	if err := os.MkdirAll(cfg.TempDir, 0755); err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}

	handler := handler.New(cfg)
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.MaxBodySize(cfg.MaxUploadSize))
	router.Use(middleware.IPRateLimiter(cfg.RateLimitPerMinute))

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.CorsOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding"},
		ExposedHeaders:   []string{"Content-Disposition", "X-Original-Size", "X-Output-Size", "X-Savings-Percent"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	router.Get("/api/health", handler.Health)
	router.Get("/api/formats", handler.Formats)
	router.Post("/api/optimize", handler.Optimize)
	router.Post("/api/optimize/svg", handler.OptimizeSVG)
	router.Post("/api/batch", handler.BatchOptimize)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")
		if err := server.Close(); err != nil {
			log.Printf("Server Close Error: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("OptiPix API starting on :%s", cfg.Port)
	log.Printf("Max upload size: %d bytes", cfg.MaxUploadSize)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server ListenAndServe Error: %v", err)
	}

	<-idleConnsClosed
	log.Println("Server gracefully stopped")
}
