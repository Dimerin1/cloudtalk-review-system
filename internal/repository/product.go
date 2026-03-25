package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dimerin1/cloudtalk-review-system/internal/model"
)

type ProductRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, req model.CreateProductRequest) (*model.Product, error) {
	product := &model.Product{ID: uuid.New()}

	err := r.db.QueryRow(ctx,
		`INSERT INTO products (id, name, description, price)
		 VALUES ($1, $2, $3, $4)
		 RETURNING name, description, price, created_at, updated_at`,
		product.ID, req.Name, req.Description, req.Price,
	).Scan(&product.Name, &product.Description, &product.Price, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	// Initialize the rating row so JOINs always resolve.
	_, err = r.db.Exec(ctx,
		`INSERT INTO product_ratings (product_id, average_rating, review_count) VALUES ($1, 0, 0)`,
		product.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("init product rating: %w", err)
	}

	product.AverageRating = 0
	return product, nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	product := &model.Product{ID: id}

	err := r.db.QueryRow(ctx,
		`SELECT p.name, p.description, p.price, p.created_at, p.updated_at,
		        COALESCE(pr.average_rating, 0)
		 FROM products p
		 LEFT JOIN product_ratings pr ON p.id = pr.product_id
		 WHERE p.id = $1`, id,
	).Scan(&product.Name, &product.Description, &product.Price,
		&product.CreatedAt, &product.UpdatedAt, &product.AverageRating)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get product: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) List(ctx context.Context) ([]model.Product, error) {
	rows, err := r.db.Query(ctx,
		`SELECT p.id, p.name, p.description, p.price, p.created_at, p.updated_at,
		        COALESCE(pr.average_rating, 0)
		 FROM products p
		 LEFT JOIN product_ratings pr ON p.id = pr.product_id
		 ORDER BY p.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]model.Product, 0)
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price,
			&p.CreatedAt, &p.UpdatedAt, &p.AverageRating); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}

func (r *ProductRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateProductRequest) (*model.Product, error) {
	product := &model.Product{ID: id}

	err := r.db.QueryRow(ctx,
		`UPDATE products
		 SET name = COALESCE($2, name),
		     description = COALESCE($3, description),
		     price = COALESCE($4, price),
		     updated_at = NOW()
		 WHERE id = $1
		 RETURNING name, description, price, created_at, updated_at`,
		id, req.Name, req.Description, req.Price,
	).Scan(&product.Name, &product.Description, &product.Price,
		&product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update product: %w", err)
	}

	// Fetch current average rating so the response includes it.
	_ = r.db.QueryRow(ctx,
		`SELECT COALESCE(average_rating, 0) FROM product_ratings WHERE product_id = $1`, id,
	).Scan(&product.AverageRating)

	return product, nil
}

func (r *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if result.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
