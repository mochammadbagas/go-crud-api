package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/repository"
)

// StartScheduledPublisher runs a periodic background worker to publish scheduled articles.
func StartScheduledPublisher(ctx context.Context, articleRepo repository.ArticleRepository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial check on server startup
	if err := articleRepo.PublishScheduled(ctx); err != nil {
		slog.Error("Failed to publish scheduled articles on startup", slog.Any("error", err))
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Scheduled publisher worker stopped.")
			return
		case <-ticker.C:
			if err := articleRepo.PublishScheduled(ctx); err != nil {
				slog.Error("Failed to publish scheduled articles", slog.Any("error", err))
			}
		}
	}
}
