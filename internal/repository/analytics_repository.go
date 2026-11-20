package repository

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
)

type AnalyticsRepository interface {
	RecordEvent(ctx context.Context, articleID string, input models.AnalyticsEventInput, visitorID string) error
	GetArticleDailyAnalytics(ctx context.Context, articleID string, startDate, endDate time.Time) ([]models.ArticleDailyAnalytics, error)
	GetUniqueVisitorsByRange(ctx context.Context, articleID string, startDate, endDate time.Time) (int64, error)
	AggregateDaily(ctx context.Context, date time.Time) error
	PruneRawEvents(ctx context.Context, before time.Time) error
	GetOverview(ctx context.Context) (*models.AnalyticsOverview, error)
}

type sqlAnalyticsRepository struct {
	db *sql.DB
	q  *sqlc.Queries
}

func NewAnalyticsRepository(sqlDB *sql.DB) AnalyticsRepository {
	return &sqlAnalyticsRepository{
		db: sqlDB,
		q:  sqlc.New(sqlDB),
	}
}

func (r *sqlAnalyticsRepository) RecordEvent(ctx context.Context, articleID string, input models.AnalyticsEventInput, visitorID string) error {
	now := time.Now().UTC()
	return r.q.RecordAnalyticsEvent(ctx, sqlc.RecordAnalyticsEventParams{
		ArticleID:          articleID,
		EventID:            input.EventID,
		VisitorID:          visitorID,
		SessionID:          input.SessionID,
		EventType:          input.EventType,
		ReadingTimeSeconds: int32(input.ReadingTimeSeconds),
		ScrollDepth:        int32(input.ScrollDepth),
		CreatedAt:          now,
	})
}

func (r *sqlAnalyticsRepository) GetArticleDailyAnalytics(
	ctx context.Context,
	articleID string,
	startDate, endDate time.Time,
) ([]models.ArticleDailyAnalytics, error) {
	rows, err := r.q.GetArticleDailyAnalytics(ctx, sqlc.GetArticleDailyAnalyticsParams{
		ArticleID:       articleID,
		AnalyticsDate:   startDate,
		AnalyticsDate_2: endDate,
	})
	if err != nil {
		return nil, err
	}

	result := make([]models.ArticleDailyAnalytics, len(rows))
	for i, row := range rows {
		depth, _ := strconv.ParseFloat(row.AverageScrollDepth, 64)
		result[i] = models.ArticleDailyAnalytics{
			ID:                 row.ID,
			ArticleID:          row.ArticleID,
			AnalyticsDate:      row.AnalyticsDate.Format("2006-01-02"),
			TotalViews:         row.TotalViews,
			UniqueVisitors:     row.UniqueVisitors,
			TotalShares:        row.TotalShares,
			TotalReadingTime:   row.TotalReadingTime,
			AverageScrollDepth: depth,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		}
	}
	return result, nil
}

func (r *sqlAnalyticsRepository) GetUniqueVisitorsByRange(
	ctx context.Context,
	articleID string,
	startDate, endDate time.Time,
) (int64, error) {
	return r.q.GetUniqueVisitorsByDateRange(ctx, sqlc.GetUniqueVisitorsByDateRangeParams{
		ArticleID:   articleID,
		CreatedAt:   startDate,
		CreatedAt_2: endDate,
	})
}

func (r *sqlAnalyticsRepository) AggregateDaily(ctx context.Context, date time.Time) error {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	startOfNextDay := startOfDay.AddDate(0, 0, 1)

	return r.q.AggregateDailyAnalyticsForDate(ctx, sqlc.AggregateDailyAnalyticsForDateParams{
		AnalyticsDate: startOfDay,
		CreatedAt:     startOfDay,
		CreatedAt_2:   startOfNextDay,
	})
}

func (r *sqlAnalyticsRepository) PruneRawEvents(ctx context.Context, before time.Time) error {
	// Chunked batch delete to prevent long-held InnoDB range locks and replication lag (PERF-06)
	batchSize := 5000
	query := "DELETE FROM article_analytics_events WHERE created_at < ? LIMIT ?"
	for {
		res, err := r.db.ExecContext(ctx, query, before, batchSize)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 || affected < int64(batchSize) {
			break
		}
		// Brief pause to yield lock contention to incoming public analytics events
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return nil
}

func toInt64(val interface{}) int64 {
	switch v := val.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case uint64:
		return int64(v)
	case []byte:
		n, _ := strconv.ParseInt(string(v), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}

func (r *sqlAnalyticsRepository) GetOverview(ctx context.Context) (*models.AnalyticsOverview, error) {
	row, err := r.q.GetTotalAnalyticsOverview(ctx)
	if err != nil {
		return nil, err
	}

	overview := &models.AnalyticsOverview{
		GlobalViews:          toInt64(row.GlobalViews),
		GlobalUniqueVisitors: toInt64(row.GlobalUniqueVisitors),
		GlobalShares:         toInt64(row.GlobalShares),
		GlobalReadingTime:    toInt64(row.GlobalReadingTime),
	}

	return overview, nil
}
