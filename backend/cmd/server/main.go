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

	h := handler.New(cfg)
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.MaxBodySize(cfg.MaxUploadSize))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.CorsOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding"},
		ExposedHeaders:   []string{"Content-Disposition", "X-Original-Size", "X-Output-Size", "X-Savings-Percent"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/health", h.Health)
	r.Get("/api/formats", h.Formats)
	r.Post("/api/optimize", h.Optimize)
	r.Post("/api/optimize/svg", h.OptimizeSVG)
	r.Post("/api/batch", h.BatchOptimize)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Graceful shutdown
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")
		if err := srv.Close(); err != nil {
			log.Printf("Server Close Error: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("OptiPix API starting on :%s", cfg.Port)
	log.Printf("Max upload size: %d bytes", cfg.MaxUploadSize)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server ListenAndServe Error: %v", err)
	}

	<-idleConnsClosed
	log.Println("Server gracefully stopped")
}
