package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/dimerin1/cloudtalk-review-system/internal/model"
)

const (
	productRatingKey  = "product:%s:rating"
	productReviewsKey = "product:%s:reviews"
	cacheTTL          = 15 * time.Minute
)

// Cache wraps Redis operations for product reviews and ratings.
type Cache struct {
	client *redis.Client
}

func New(client *redis.Client) *Cache {
	return &Cache{client: client}
}

// GetProductRating returns the cached average rating for a product.
func (c *Cache) GetProductRating(ctx context.Context, productID uuid.UUID) (float64, error) {
	key := fmt.Sprintf(productRatingKey, productID)
	return c.client.Get(ctx, key).Float64()
}

// SetProductRating caches the average rating for a product.
func (c *Cache) SetProductRating(ctx context.Context, productID uuid.UUID, rating float64) error {
	key := fmt.Sprintf(productRatingKey, productID)
	return c.client.Set(ctx, key, rating, cacheTTL).Err()
}

// GetProductReviews returns cached reviews for a product.
func (c *Cache) GetProductReviews(ctx context.Context, productID uuid.UUID) ([]model.Review, error) {
	key := fmt.Sprintf(productReviewsKey, productID)
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var reviews []model.Review
	if err := json.Unmarshal(val, &reviews); err != nil {
		return nil, err
	}
	return reviews, nil
}

// SetProductReviews caches reviews for a product.
func (c *Cache) SetProductReviews(ctx context.Context, productID uuid.UUID, reviews []model.Review) error {
	key := fmt.Sprintf(productReviewsKey, productID)
	data, err := json.Marshal(reviews)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, cacheTTL).Err()
}

// InvalidateProduct removes all cached data for a product.
func (c *Cache) InvalidateProduct(ctx context.Context, productID uuid.UUID) error {
	keys := []string{
		fmt.Sprintf(productRatingKey, productID),
		fmt.Sprintf(productReviewsKey, productID),
	}
	return c.client.Del(ctx, keys...).Err()
}
