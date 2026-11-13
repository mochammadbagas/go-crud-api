package service

import (
	"context"
	"errors"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

var (
	ErrCategoryNotFound   = errors.New("kategori tidak ditemukan")
	ErrCategorySlugExists = errors.New("slug kategori sudah digunakan")
)

// CategoryServiceInterface defines business actions for category management and retrieval.
type CategoryServiceInterface interface {
	List(ctx context.Context) ([]models.Category, error)
	GetByID(ctx context.Context, id uint64) (*models.Category, error)
	GetBySlug(ctx context.Context, slug string) (*models.Category, error)
	Create(ctx context.Context, input models.CategoryInput) (*models.Category, error)
	Update(ctx context.Context, id uint64, input models.CategoryInput) (*models.Category, error)
	Delete(ctx context.Context, id uint64) error
}

type CategoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) *CategoryService {
	return &CategoryService{categoryRepo: categoryRepo}
}

func (s *CategoryService) List(ctx context.Context) ([]models.Category, error) {
	return s.categoryRepo.List(ctx)
}

func (s *CategoryService) GetByID(ctx context.Context, id uint64) (*models.Category, error) {
	cat, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return cat, nil
}

func (s *CategoryService) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	cat, err := s.categoryRepo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return cat, nil
}

func (s *CategoryService) Create(ctx context.Context, input models.CategoryInput) (*models.Category, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(input.Name)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	// Check slug uniqueness
	existing, err := s.categoryRepo.GetBySlug(ctx, slug)
	if err == nil && existing != nil {
		return nil, ErrCategorySlugExists
	}

	cat := &models.Category{
		Name: strings.TrimSpace(input.Name),
		Slug: slug,
	}

	if err := s.categoryRepo.Create(ctx, cat); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrCategorySlugExists
		}
		return nil, err
	}

	return cat, nil
}

func (s *CategoryService) Update(ctx context.Context, id uint64, input models.CategoryInput) (*models.Category, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(input.Name)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	// If slug changed, verify uniqueness
	if slug != existing.Slug {
		collision, err := s.categoryRepo.GetBySlug(ctx, slug)
		if err == nil && collision != nil && collision.ID != id {
			return nil, ErrCategorySlugExists
		}
	}

	existing.Name = strings.TrimSpace(input.Name)
	existing.Slug = slug

	if err := s.categoryRepo.Update(ctx, existing); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrCategorySlugExists
		}
		return nil, err
	}

	return existing, nil
}

func (s *CategoryService) Delete(ctx context.Context, id uint64) error {
	_, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCategoryNotFound
		}
		return err
	}
	return s.categoryRepo.Delete(ctx, id)
}

