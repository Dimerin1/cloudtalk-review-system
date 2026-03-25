package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/dimerin1/cloudtalk-review-system/internal/model"
	"github.com/dimerin1/cloudtalk-review-system/internal/repository"
)

type ProductService struct {
	repo *repository.ProductRepository
}

func NewProductService(repo *repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Create(ctx context.Context, req model.CreateProductRequest) (*model.Product, error) {
	return s.repo.Create(ctx, req)
}

func (s *ProductService) GetByID(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) List(ctx context.Context) ([]model.Product, error) {
	return s.repo.List(ctx)
}

func (s *ProductService) Update(ctx context.Context, id uuid.UUID, req model.UpdateProductRequest) (*model.Product, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
