package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("resource not found")

// Product represents a product in the system.
type Product struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	AverageRating float64   `json:"average_rating"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Review represents a product review.
type Review struct {
	ID         uuid.UUID `json:"id"`
	ProductID  uuid.UUID `json:"product_id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	ReviewText string    `json:"review_text"`
	Rating     int       `json:"rating"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateProductRequest is the payload for creating a product.
type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

// UpdateProductRequest is the payload for updating a product.
// Nil fields are left unchanged.
type UpdateProductRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
}

// CreateReviewRequest is the payload for creating a review.
type CreateReviewRequest struct {
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	ReviewText string `json:"review_text"`
	Rating     int    `json:"rating"`
}

// UpdateReviewRequest is the payload for updating a review.
// Nil fields are left unchanged.
type UpdateReviewRequest struct {
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	ReviewText *string `json:"review_text"`
	Rating     *int    `json:"rating"`
}

// ReviewEvent is published to the message broker when a review changes.
type ReviewEvent struct {
	Type      string    `json:"type"`
	ReviewID  uuid.UUID `json:"review_id"`
	ProductID uuid.UUID `json:"product_id"`
	Rating    int       `json:"rating,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
