package utils_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type samplePayload struct {
	Name string `json:"name"`
}

func TestDecodeJSON_Success(t *testing.T) {
	body := `{"name":"alice"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rec := httptest.NewRecorder()

	var data samplePayload
	err := utils.DecodeJSON(rec, req, &data)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if data.Name != "alice" {
		t.Fatalf("expected name to be 'alice', got: %s", data.Name)
	}
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte{}))
	rec := httptest.NewRecorder()

	var data samplePayload
	err := utils.DecodeJSON(rec, req, &data)
	if err == nil {
		t.Fatal("expected error on empty body, got nil")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("expected 'cannot be empty' error message, got: %v", err)
	}
}

func TestDecodeJSON_PayloadTooLarge(t *testing.T) {
	// Custom limit of 10 bytes
	body := `{"name":"this payload is longer than 10 bytes"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rec := httptest.NewRecorder()

	var data samplePayload
	err := utils.DecodeJSON(rec, req, &data, 10)
	if err == nil {
		t.Fatal("expected ErrPayloadTooLarge, got nil")
	}
	if err != utils.ErrPayloadTooLarge {
		t.Fatalf("expected utils.ErrPayloadTooLarge, got: %v", err)
	}

	utils.HandleDecodeError(rec, err)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "PAYLOAD_TOO_LARGE") {
		t.Fatalf("expected PAYLOAD_TOO_LARGE in body, got: %s", rec.Body.String())
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	body := `{"name":`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rec := httptest.NewRecorder()

	var data samplePayload
	err := utils.DecodeJSON(rec, req, &data)
	if err == nil {
		t.Fatal("expected invalid format error, got nil")
	}

	utils.HandleDecodeError(rec, err)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "INVALID_REQUEST_BODY") {
		t.Fatalf("expected INVALID_REQUEST_BODY in body, got: %s", rec.Body.String())
	}
}
