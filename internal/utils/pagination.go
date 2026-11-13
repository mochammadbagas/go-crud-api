package utils

import (
	"math"
	"net/http"
	"strconv"
)

// Pagination represents pagination metadata.
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// PaginatedResponse wraps data with pagination metadata.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// ParsePagination extracts page and limit from query parameters with defaults and bounds.
func ParsePagination(r *http.Request, defaultLimit, maxLimit int) (page int, limit int) {
	q := r.URL.Query()

	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err = strconv.Atoi(q.Get("limit"))
	if err != nil || limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	return page, limit
}

// BuildPagination constructs the Pagination struct.
func BuildPagination(page, limit int, totalItems int64) Pagination {
	totalPages := 0
	if limit > 0 && totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(limit)))
	}

	return Pagination{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// CalculateOffset returns the SQL OFFSET for the given page and limit.
func CalculateOffset(page, limit int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * limit
}

