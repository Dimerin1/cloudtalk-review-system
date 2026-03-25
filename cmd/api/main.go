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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/dimerin1/cloudtalk-review-system/internal/cache"
	"github.com/dimerin1/cloudtalk-review-system/internal/config"
	"github.com/dimerin1/cloudtalk-review-system/internal/events"
	"github.com/dimerin1/cloudtalk-review-system/internal/handler"
	"github.com/dimerin1/cloudtalk-review-system/internal/repository"
	"github.com/dimerin1/cloudtalk-review-system/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	// --- Database ---
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// --- Redis ---
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Error("redis url parse failed", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	// --- Kafka ---
	producer := events.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	// --- Wire dependencies ---
	productRepo := repository.NewProductRepository(pool)
	reviewRepo := repository.NewReviewRepository(pool)
	appCache := cache.New(rdb)

	productSvc := service.NewProductService(productRepo)
	reviewSvc := service.NewReviewService(reviewRepo, appCache, producer, logger)

	productH := handler.NewProductHandler(productSvc)
	reviewH := handler.NewReviewHandler(reviewSvc)

	// --- Router ---
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/products", func(r chi.Router) {
			r.Post("/", productH.Create)
			r.Get("/", productH.List)
			r.Get("/{id}", productH.GetByID)
			r.Put("/{id}", productH.Update)
			r.Delete("/{id}", productH.Delete)

			// Reviews scoped to a product.
			r.Get("/{id}/reviews", reviewH.GetByProductID)
			r.Post("/{id}/reviews", reviewH.Create)
		})

		r.Route("/reviews", func(r chi.Router) {
			r.Put("/{id}", reviewH.Update)
			r.Delete("/{id}", reviewH.Delete)
		})
	})

	// --- Server with graceful shutdown ---
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("api server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("server stopped")
}
