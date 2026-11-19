package models

import (
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

// Allowed analytics event types
const (
	EventTypeView        = "view"
	EventTypeScroll      = "scroll"
	EventTypeReadingTime = "reading_time"
	EventTypeShare       = "share"
)

// AnalyticsEventInput represents raw event payload submitted from browser.
type AnalyticsEventInput struct {
	EventID            string `json:"event_id"`
	SessionID          string `json:"session_id"`
	EventType          string `json:"event_type"`
	ReadingTimeSeconds int    `json:"reading_time_seconds,omitempty"`
	ScrollDepth        int    `json:"scroll_depth,omitempty"`
}

// Validate validates AnalyticsEventInput.
func (i *AnalyticsEventInput) Validate() error {
	v := utils.ValidationErrors{}

	i.EventID = strings.TrimSpace(i.EventID)
	if i.EventID == "" {
		v.Add("event_id", utils.ErrCodeRequired, "event_id is required for idempotency")
	}

	i.EventType = strings.TrimSpace(strings.ToLower(i.EventType))
	if i.EventType == "" {
		v.Add("event_type", utils.ErrCodeRequired, "event_type is required")
	} else if i.EventType != EventTypeView && i.EventType != EventTypeScroll && i.EventType != EventTypeReadingTime && i.EventType != EventTypeShare {
		v.Add("event_type", utils.ErrCodeInvalidFormat, "event_type must be one of: view, scroll, reading_time, share")
	}

	if i.ScrollDepth < 0 || i.ScrollDepth > 100 {
		v.Add("scroll_depth", utils.ErrCodeInvalidFormat, "scroll_depth must be between 0 and 100")
	}

	if i.ReadingTimeSeconds < 0 || i.ReadingTimeSeconds > 86400 {
		v.Add("reading_time_seconds", utils.ErrCodeInvalidFormat, "reading_time_seconds must be between 0 and 86400")
	}

	if v.HasErrors() {
		return v
	}
	return nil
}

// ArticleDailyAnalytics represents aggregated metrics for a single date.
type ArticleDailyAnalytics struct {
	ID                 int64     `json:"id"`
	ArticleID          string    `json:"article_id"`
	AnalyticsDate      string    `json:"analytics_date"`
	TotalViews         int64     `json:"total_views"`
	UniqueVisitors     int64     `json:"unique_visitors"`
	TotalShares        int64     `json:"total_shares"`
	TotalReadingTime   int64     `json:"total_reading_time"`
	AverageScrollDepth float64   `json:"average_scroll_depth"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ArticleAnalyticsReport represents comprehensive analytics for an article over a date range.
type ArticleAnalyticsReport struct {
	ArticleID          string                  `json:"article_id"`
	StartDate          string                  `json:"start_date"`
	EndDate            string                  `json:"end_date"`
	TotalViews         int64                   `json:"total_views"`
	ExactUniqueVisitors int64                  `json:"exact_unique_visitors"`
	TotalShares        int64                   `json:"total_shares"`
	TotalReadingTime   int64                   `json:"total_reading_time"`
	AverageScrollDepth float64                 `json:"average_scroll_depth"`
	DailyBreakdown     []ArticleDailyAnalytics `json:"daily_breakdown"`
}

// AnalyticsOverview represents overall platform-wide metrics.
type AnalyticsOverview struct {
	GlobalViews          int64 `json:"global_views"`
	GlobalUniqueVisitors int64 `json:"global_unique_visitors"`
	GlobalShares         int64 `json:"global_shares"`
	GlobalReadingTime    int64 `json:"global_reading_time"`
}
