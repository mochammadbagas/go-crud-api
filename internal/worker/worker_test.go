package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/worker"
)

type mockArticleRepoForWorker struct {
	publishCount int32
}

func (m *mockArticleRepoForWorker) Create(ctx context.Context, a *models.Article, tagIDs, mediaIDs, relIDs []string) error {
	return nil
}
func (m *mockArticleRepoForWorker) Update(ctx context.Context, a *models.Article, tagIDs, mediaIDs, relIDs []string, uID string) error {
	return nil
}
func (m *mockArticleRepoForWorker) Delete(ctx context.Context, id string) error               { return nil }
func (m *mockArticleRepoForWorker) GetByID(ctx context.Context, id string) (*models.Article, error) {
	return nil, nil
}
func (m *mockArticleRepoForWorker) GetBySlug(ctx context.Context, slug string) (*models.Article, error) {
	return nil, nil
}
func (m *mockArticleRepoForWorker) GetPublishedBySlug(ctx context.Context, slug string) (*models.Article, error) {
	return nil, nil
}
func (m *mockArticleRepoForWorker) ListPublic(ctx context.Context, f models.ArticleFilter) ([]models.Article, int64, error) {
	return nil, 0, nil
}
func (m *mockArticleRepoForWorker) ListAdmin(ctx context.Context, f models.ArticleFilter) ([]models.Article, int64, error) {
	return nil, 0, nil
}
func (m *mockArticleRepoForWorker) PublishScheduled(ctx context.Context) error {
	atomic.AddInt32(&m.publishCount, 1)
	return nil
}
func (m *mockArticleRepoForWorker) Count(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockArticleRepoForWorker) ListRevisions(ctx context.Context, id string) ([]models.ArticleRevision, error) {
	return nil, nil
}
func (m *mockArticleRepoForWorker) CheckArticlePublishedExists(ctx context.Context, id string) (bool, error) {
	return true, nil
}

type mockAnalyticsSvcForWorker struct {
	aggregateCount int32
	pruneCount     int32
}

func (m *mockAnalyticsSvcForWorker) RecordEvent(ctx context.Context, articleID string, input models.AnalyticsEventInput, ip, userAgent string) error {
	return nil
}
func (m *mockAnalyticsSvcForWorker) GetArticleAnalytics(ctx context.Context, articleID string, start, end time.Time) (*models.ArticleAnalyticsReport, error) {
	return nil, nil
}
func (m *mockAnalyticsSvcForWorker) GetOverview(ctx context.Context) (*models.AnalyticsOverview, error) {
	return nil, nil
}
func (m *mockAnalyticsSvcForWorker) Aggregate(ctx context.Context, date time.Time) error {
	atomic.AddInt32(&m.aggregateCount, 1)
	return nil
}
func (m *mockAnalyticsSvcForWorker) PruneRawEvents(ctx context.Context, retentionDays int) error {
	atomic.AddInt32(&m.pruneCount, 1)
	return nil
}
func (m *mockAnalyticsSvcForWorker) GeneratePseudonymVisitorID(ip, userAgent string, date time.Time) string {
	return "pseudonym"
}

func TestStartScheduledPublisher_RunsAndStops(t *testing.T) {
	repo := &mockArticleRepoForWorker{}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		worker.StartScheduledPublisher(ctx, repo, 10*time.Millisecond)
		close(done)
	}()

	// Allow ticker to fire at least once
	time.Sleep(35 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Worker exited cleanly
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not exit cleanly upon context cancellation")
	}

	count := atomic.LoadInt32(&repo.publishCount)
	if count < 1 {
		t.Errorf("expected PublishScheduled to be called at least once, got %d", count)
	}
}

func TestStartAnalyticsAggregator_RunsAndStops(t *testing.T) {
	svc := &mockAnalyticsSvcForWorker{}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		worker.StartAnalyticsAggregator(ctx, svc, 10*time.Millisecond)
		close(done)
	}()

	time.Sleep(35 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Worker exited cleanly
	case <-time.After(500 * time.Millisecond):
		t.Fatal("analytics worker did not exit cleanly upon context cancellation")
	}

	aggCount := atomic.LoadInt32(&svc.aggregateCount)
	pruneCount := atomic.LoadInt32(&svc.pruneCount)
	if aggCount < 2 { // Runs for today and yesterday initially
		t.Errorf("expected Aggregate to be called >= 2 times, got %d", aggCount)
	}
	if pruneCount < 1 {
		t.Errorf("expected PruneRawEvents to be called >= 1 times, got %d", pruneCount)
	}
}
