package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"crudapi/internal/store"

	"github.com/gin-gonic/gin"
)

// TestProductHandler_Create menguji endpoint handler penambahan produk (POST /products).
//
// Konsep Belajar Golang: Unit Testing HTTP Handler dengan Mock Store & httptest
//  1. `gin.SetMode(gin.TestMode)`:
//     Menonaktifkan logging console Gin yang berisik saat testing berjalan.
//  2. Dependency Injection dengan `store.NewInMemoryStore()`:
//     Kita menggunakan store memori lokal, sehingga tes ini berjalan secepat kilat (sub-millisecond)
//     tanpa memerlukan koneksi internet, Docker, atau MySQL asli!
//  3. `httptest.NewRecorder()`:
//     Objek perekam respons virtual yang menampung status code dan response body HTTP.
func TestProductHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Inisialisasi in-memory repository store untuk keperluan pengujian
	s := store.NewInMemoryStore()
	h := NewProductHandler(s)

	tests := []struct {
		name           string
		body           map[string]any
		expectedStatus int
	}{
		{
			name: "Pembuatan produk valid berhasil",
			body: map[string]any{
				"name":        "Laptop Ultrabook",
				"price":       16500000.0,
				"stock":       8,
				"category_id": 1,
			},
			expectedStatus: http.StatusCreated, // Ekspektasi HTTP 201
		},
		{
			name: "Nama produk kosong ditolak",
			body: map[string]any{
				"name":        "",
				"price":       16500000.0,
				"stock":       8,
				"category_id": 1,
			},
			expectedStatus: http.StatusBadRequest, // Ekspektasi HTTP 400
		},
		{
			name: "Harga produk negatif ditolak",
			body: map[string]any{
				"name":        "Laptop Ultrabook",
				"price":       -5000.0,
				"stock":       8,
				"category_id": 1,
			},
			expectedStatus: http.StatusBadRequest, // Ekspektasi HTTP 400
		},
		{
			name: "Category ID tidak diisi ditolak",
			body: map[string]any{
				"name":  "Laptop Ultrabook",
				"price": 16500000.0,
				"stock": 8,
			},
			expectedStatus: http.StatusBadRequest, // Ekspektasi HTTP 400
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Buat ResponseRecorder dan Gin context palsu untuk request ini
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			// Serialisasi data uji ke format JSON buffer
			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			c.Request = req

			// Panggil fungsi handler secara langsung
			h.Create(c)

			// Validasi apakah status HTTP yang dihasilkan sesuai dengan yang diharapkan
			if rec.Code != tt.expectedStatus {
				t.Errorf("Ekspektasi status %v, didapatkan %v. Body Respons: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
