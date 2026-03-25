package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dimerin1/cloudtalk-review-system/internal/model"
)

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(ctx context.Context, productID uuid.UUID, req model.CreateReviewRequest) (*model.Review, error) {
	review := &model.Review{
		ID:        uuid.New(),
		ProductID: productID,
	}

	err := r.db.QueryRow(ctx,
		`INSERT INTO reviews (id, product_id, first_name, last_name, review_text, rating)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING first_name, last_name, review_text, rating, created_at, updated_at`,
		review.ID, productID, req.FirstName, req.LastName, req.ReviewText, req.Rating,
	).Scan(&review.FirstName, &review.LastName, &review.ReviewText,
		&review.Rating, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		// Foreign key violation means the product doesn't exist.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("create review: %w", err)
	}

	return review, nil
}

func (r *ReviewRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Review, error) {
	review := &model.Review{ID: id}

	err := r.db.QueryRow(ctx,
		`SELECT product_id, first_name, last_name, review_text, rating, created_at, updated_at
		 FROM reviews WHERE id = $1`, id,
	).Scan(&review.ProductID, &review.FirstName, &review.LastName,
		&review.ReviewText, &review.Rating, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get review: %w", err)
	}

	return review, nil
}

func (r *ReviewRepository) GetByProductID(ctx context.Context, productID uuid.UUID) ([]model.Review, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, product_id, first_name, last_name, review_text, rating, created_at, updated_at
		 FROM reviews WHERE product_id = $1
		 ORDER BY created_at DESC`, productID,
	)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	reviews := make([]model.Review, 0)
	for rows.Next() {
		var rv model.Review
		if err := rows.Scan(&rv.ID, &rv.ProductID, &rv.FirstName, &rv.LastName,
			&rv.ReviewText, &rv.Rating, &rv.CreatedAt, &rv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, rv)
	}

	return reviews, nil
}

func (r *ReviewRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateReviewRequest) (*model.Review, error) {
	review := &model.Review{ID: id}

	err := r.db.QueryRow(ctx,
		`UPDATE reviews
		 SET first_name = COALESCE($2, first_name),
		     last_name = COALESCE($3, last_name),
		     review_text = COALESCE($4, review_text),
		     rating = COALESCE($5, rating),
		     updated_at = NOW()
		 WHERE id = $1
		 RETURNING product_id, first_name, last_name, review_text, rating, created_at, updated_at`,
		id, req.FirstName, req.LastName, req.ReviewText, req.Rating,
	).Scan(&review.ProductID, &review.FirstName, &review.LastName,
		&review.ReviewText, &review.Rating, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update review: %w", err)
	}

	return review, nil
}

// Delete removes a review and returns it so the caller knows the product_id.
func (r *ReviewRepository) Delete(ctx context.Context, id uuid.UUID) (*model.Review, error) {
	review := &model.Review{ID: id}

	err := r.db.QueryRow(ctx,
		`DELETE FROM reviews WHERE id = $1
		 RETURNING product_id, first_name, last_name, review_text, rating, created_at, updated_at`,
		id,
	).Scan(&review.ProductID, &review.FirstName, &review.LastName,
		&review.ReviewText, &review.Rating, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("delete review: %w", err)
	}

	return review, nil
}

// RecalculateAverage recomputes the average rating for a product.
// Uses SELECT FOR UPDATE to guarantee concurrency safety at the database level.
func (r *ReviewRepository) RecalculateAverage(ctx context.Context, productID uuid.UUID) (float64, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock the rating row to prevent concurrent updates.
	_, err = tx.Exec(ctx,
		`SELECT 1 FROM product_ratings WHERE product_id = $1 FOR UPDATE`, productID)
	if err != nil {
		return 0, fmt.Errorf("lock rating row: %w", err)
	}

	var avgRating float64
	var count int
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM reviews WHERE product_id = $1`,
		productID,
	).Scan(&avgRating, &count)
	if err != nil {
		return 0, fmt.Errorf("calc average: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE product_ratings
		 SET average_rating = $2, review_count = $3, updated_at = NOW()
		 WHERE product_id = $1`,
		productID, avgRating, count,
	)
	if err != nil {
		return 0, fmt.Errorf("update rating: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return avgRating, nil
}
