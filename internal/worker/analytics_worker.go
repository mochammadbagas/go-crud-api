package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/service"
)

// StartAnalyticsAggregator periodically rolls up raw events into daily aggregates.
func StartAnalyticsAggregator(ctx context.Context, analyticsSvc service.AnalyticsServiceInterface, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial aggregation run
	runAggregation(ctx, analyticsSvc)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Analytics aggregator worker stopped.")
			return
		case <-ticker.C:
			runAggregation(ctx, analyticsSvc)
		}
	}
}

func runAggregation(ctx context.Context, analyticsSvc service.AnalyticsServiceInterface) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	if err := analyticsSvc.Aggregate(ctx, today); err != nil {
		slog.Error("Failed to aggregate daily analytics for today", slog.Any("error", err))
	}
	if err := analyticsSvc.Aggregate(ctx, yesterday); err != nil {
		slog.Error("Failed to aggregate daily analytics for yesterday", slog.Any("error", err))
	}
}
