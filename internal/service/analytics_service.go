package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
)

type AnalyticsServiceInterface interface {
	RecordEvent(ctx context.Context, articleID string, input models.AnalyticsEventInput, ip, userAgent string) error
	GetArticleAnalytics(ctx context.Context, articleID string, startDate, endDate time.Time) (*models.ArticleAnalyticsReport, error)
	GetOverview(ctx context.Context) (*models.AnalyticsOverview, error)
	Aggregate(ctx context.Context, date time.Time) error
	PruneRawEvents(ctx context.Context, retentionDays int) error
}

type AnalyticsService struct {
	analyticsRepo repository.AnalyticsRepository
	articleRepo   repository.ArticleRepository
	salt          string
}

func NewAnalyticsService(
	analyticsRepo repository.AnalyticsRepository,
	articleRepo repository.ArticleRepository,
	salt string,
) *AnalyticsService {
	if salt == "" {
		salt = "go-crud-api-analytics-salt-v1"
	}
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		articleRepo:   articleRepo,
		salt:          salt,
	}
}

// GeneratePseudonymVisitorID hashes IP + UserAgent with monthly salt to preserve privacy (GDPR compliance).
func (s *AnalyticsService) GeneratePseudonymVisitorID(ip, userAgent string, t time.Time) string {
	monthlySalt := fmt.Sprintf("%s-%s", s.salt, t.Format("2006-01"))
	data := fmt.Sprintf("%s|%s|%s", ip, userAgent, monthlySalt)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func (s *AnalyticsService) RecordEvent(
	ctx context.Context,
	articleID string,
	input models.AnalyticsEventInput,
	ip, userAgent string,
) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// Verify article exists, is published, and not deleted (PERF-01 & SEC-05)
	exists, err := s.articleRepo.CheckArticlePublishedExists(ctx, articleID)
	if err != nil {
		return err
	}
	if !exists {
		return repository.ErrArticleNotFound
	}

	visitorID := s.GeneratePseudonymVisitorID(ip, userAgent, time.Now().UTC())
	return s.analyticsRepo.RecordEvent(ctx, articleID, input, visitorID)
}

func (s *AnalyticsService) GetArticleAnalytics(
	ctx context.Context,
	articleID string,
	startDate, endDate time.Time,
) (*models.ArticleAnalyticsReport, error) {
	if _, err := s.articleRepo.GetByID(ctx, articleID); err != nil {
		return nil, err
	}

	daily, err := s.analyticsRepo.GetArticleDailyAnalytics(ctx, articleID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	exactUniques, err := s.analyticsRepo.GetUniqueVisitorsByRange(ctx, articleID, startDate, endDate)
	if err != nil {
		exactUniques = 0
	}

	var totalViews, totalShares, totalReadingTime int64
	var totalScrollDepth float64

	for _, d := range daily {
		totalViews += d.TotalViews
		totalShares += d.TotalShares
		totalReadingTime += d.TotalReadingTime
		totalScrollDepth += d.AverageScrollDepth
	}

	avgScroll := 0.00
	if len(daily) > 0 {
		avgScroll = totalScrollDepth / float64(len(daily))
	}

	report := &models.ArticleAnalyticsReport{
		ArticleID:           articleID,
		StartDate:           startDate.Format("2006-01-02"),
		EndDate:             endDate.Format("2006-01-02"),
		TotalViews:          totalViews,
		ExactUniqueVisitors: exactUniques,
		TotalShares:         totalShares,
		TotalReadingTime:    totalReadingTime,
		AverageScrollDepth:  avgScroll,
		DailyBreakdown:      daily,
	}

	return report, nil
}

func (s *AnalyticsService) GetOverview(ctx context.Context) (*models.AnalyticsOverview, error) {
	return s.analyticsRepo.GetOverview(ctx)
}

func (s *AnalyticsService) Aggregate(ctx context.Context, date time.Time) error {
	return s.analyticsRepo.AggregateDaily(ctx, date)
}

func (s *AnalyticsService) PruneRawEvents(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	return s.analyticsRepo.PruneRawEvents(ctx, cutoff)
}
