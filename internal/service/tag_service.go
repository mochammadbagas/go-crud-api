package service

import (
	"context"
	"errors"
	"strings"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type TagServiceInterface interface {
	List(ctx context.Context) ([]models.Tag, error)
	GetByID(ctx context.Context, id string) (*models.Tag, error)
	GetBySlug(ctx context.Context, slug string) (*models.Tag, error)
	Create(ctx context.Context, input models.TagInput) (*models.Tag, error)
	Update(ctx context.Context, id string, input models.TagInput) (*models.Tag, error)
	Delete(ctx context.Context, id string) error
}

type TagService struct {
	tagRepo repository.TagRepository
}

func NewTagService(tagRepo repository.TagRepository) *TagService {
	return &TagService{tagRepo: tagRepo}
}

func (s *TagService) List(ctx context.Context) ([]models.Tag, error) {
	return s.tagRepo.List(ctx)
}

func (s *TagService) GetByID(ctx context.Context, id string) (*models.Tag, error) {
	return s.tagRepo.GetByID(ctx, id)
}

func (s *TagService) GetBySlug(ctx context.Context, slug string) (*models.Tag, error) {
	return s.tagRepo.GetBySlug(ctx, slug)
}

func (s *TagService) Create(ctx context.Context, input models.TagInput) (*models.Tag, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(input.Name)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	existing, err := s.tagRepo.GetBySlug(ctx, slug)
	if err == nil && existing != nil {
		return nil, repository.ErrTagSlugExists
	}

	tag := &models.Tag{
		Name: strings.TrimSpace(input.Name),
		Slug: slug,
	}

	if err := s.tagRepo.Create(ctx, tag); err != nil {
		if errors.Is(err, repository.ErrTagSlugExists) || errors.Is(err, repository.ErrDuplicate) {
			return nil, repository.ErrTagSlugExists
		}
		return nil, err
	}
	return tag, nil
}

func (s *TagService) Update(ctx context.Context, id string, input models.TagInput) (*models.Tag, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.tagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = utils.GenerateSlug(input.Name)
	} else {
		slug = utils.GenerateSlug(slug)
	}

	if slug != existing.Slug {
		collision, err := s.tagRepo.GetBySlug(ctx, slug)
		if err == nil && collision != nil && collision.ID != id {
			return nil, repository.ErrTagSlugExists
		}
	}

	existing.Name = strings.TrimSpace(input.Name)
	existing.Slug = slug

	if err := s.tagRepo.Update(ctx, existing); err != nil {
		if errors.Is(err, repository.ErrTagSlugExists) || errors.Is(err, repository.ErrDuplicate) {
			return nil, repository.ErrTagSlugExists
		}
		return nil, err
	}
	return existing, nil
}

func (s *TagService) Delete(ctx context.Context, id string) error {
	_, err := s.tagRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.tagRepo.Delete(ctx, id)
}
