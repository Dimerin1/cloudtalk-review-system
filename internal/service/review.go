package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"github.com/dimerin1/cloudtalk-review-system/internal/cache"
	"github.com/dimerin1/cloudtalk-review-system/internal/events"
	"github.com/dimerin1/cloudtalk-review-system/internal/model"
	"github.com/dimerin1/cloudtalk-review-system/internal/repository"
)

type ReviewService struct {
	repo     *repository.ReviewRepository
	cache    *cache.Cache
	producer *events.Producer
	logger   *slog.Logger

	// ratingMu holds per-product mutexes for concurrent average recalculations.
	// Combined with the DB-level SELECT FOR UPDATE, this provides two layers
	// of concurrency safety.
	ratingMu sync.Map
}

func NewReviewService(
	repo *repository.ReviewRepository,
	cache *cache.Cache,
	producer *events.Producer,
	logger *slog.Logger,
) *ReviewService {
	return &ReviewService{
		repo:     repo,
		cache:    cache,
		producer: producer,
		logger:   logger,
	}
}

func (s *ReviewService) Create(ctx context.Context, productID uuid.UUID, req model.CreateReviewRequest) (*model.Review, error) {
	review, err := s.repo.Create(ctx, productID, req)
	if err != nil {
		return nil, err
	}

	if err := s.recalculateAndCache(ctx, productID); err != nil {
		s.logger.Error("recalculate average failed", "error", err)
	}

	if err := s.producer.PublishReviewEvent(ctx, "review.created", review.ID, productID, review.Rating); err != nil {
		s.logger.Error("publish event failed", "error", err)
	}

	return review, nil
}

func (s *ReviewService) GetByProductID(ctx context.Context, productID uuid.UUID) ([]model.Review, error) {
	// Try cache first.
	if reviews, err := s.cache.GetProductReviews(ctx, productID); err == nil {
		return reviews, nil
	}

	reviews, err := s.repo.GetByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	// Populate cache for subsequent reads.
	if err := s.cache.SetProductReviews(ctx, productID, reviews); err != nil {
		s.logger.Error("cache set reviews failed", "error", err)
	}

	return reviews, nil
}

func (s *ReviewService) Update(ctx context.Context, id uuid.UUID, req model.UpdateReviewRequest) (*model.Review, error) {
	review, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	if err := s.recalculateAndCache(ctx, review.ProductID); err != nil {
		s.logger.Error("recalculate average failed", "error", err)
	}

	if err := s.producer.PublishReviewEvent(ctx, "review.updated", review.ID, review.ProductID, review.Rating); err != nil {
		s.logger.Error("publish event failed", "error", err)
	}

	return review, nil
}

func (s *ReviewService) Delete(ctx context.Context, id uuid.UUID) error {
	review, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	if err := s.recalculateAndCache(ctx, review.ProductID); err != nil {
		s.logger.Error("recalculate average failed", "error", err)
	}

	if err := s.producer.PublishReviewEvent(ctx, "review.deleted", review.ID, review.ProductID, review.Rating); err != nil {
		s.logger.Error("publish event failed", "error", err)
	}

	return nil
}

// getProductMutex returns a mutex scoped to a single product ID.
func (s *ReviewService) getProductMutex(productID uuid.UUID) *sync.Mutex {
	mu, _ := s.ratingMu.LoadOrStore(productID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// recalculateAndCache recalculates the average and refreshes the cache.
// A per-product mutex serializes Go-level access; the repository additionally
// uses SELECT FOR UPDATE for database-level safety.
func (s *ReviewService) recalculateAndCache(ctx context.Context, productID uuid.UUID) error {
	mu := s.getProductMutex(productID)
	mu.Lock()
	defer mu.Unlock()

	newAvg, err := s.repo.RecalculateAverage(ctx, productID)
	if err != nil {
		return fmt.Errorf("recalculate: %w", err)
	}

	// Invalidate stale reviews cache.
	if err := s.cache.InvalidateProduct(ctx, productID); err != nil {
		s.logger.Error("cache invalidate failed", "error", err)
	}

	// Write fresh average back to cache.
	if err := s.cache.SetProductRating(ctx, productID, newAvg); err != nil {
		s.logger.Error("cache set rating failed", "error", err)
	}

	return nil
}
