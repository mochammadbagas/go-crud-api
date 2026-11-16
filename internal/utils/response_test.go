package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultErrorCode(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{http.StatusBadRequest, "BAD_REQUEST"},
		{http.StatusUnauthorized, "UNAUTHORIZED"},
		{http.StatusForbidden, "FORBIDDEN"},
		{http.StatusNotFound, "NOT_FOUND"},
		{http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED"},
		{http.StatusConflict, "CONFLICT"},
		{http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED"},
		{http.StatusInternalServerError, "INTERNAL_SERVER_ERROR"},
		{http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
		{418, "CLIENT_ERROR"},
		{502, "SERVER_ERROR"},
	}

	for _, tt := range tests {
		got := DefaultErrorCode(tt.status)
		if got != tt.expected {
			t.Errorf("DefaultErrorCode(%d) = %q, expected %q", tt.status, got, tt.expected)
		}
	}
}

func TestErrorResponseSerialization(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, http.StatusNotFound, "resource not found")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", resp.Code)
	}
	if resp.Error != "resource not found" {
		t.Errorf("expected error 'resource not found', got %q", resp.Error)
	}
	if resp.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %q", resp.Message)
	}
}

func TestErrorCodeResponseSerialization(t *testing.T) {
	rec := httptest.NewRecorder()
	ErrorCode(rec, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "email is already registered")

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != "EMAIL_ALREADY_EXISTS" {
		t.Errorf("expected code EMAIL_ALREADY_EXISTS, got %q", resp.Code)
	}
	if resp.Error != "email is already registered" {
		t.Errorf("expected error 'email is already registered', got %q", resp.Error)
	}
	if resp.Message != "email is already registered" {
		t.Errorf("expected message 'email is already registered', got %q", resp.Message)
	}
}
