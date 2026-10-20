package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"crudapi/internal/models"
	"crudapi/internal/store"

	"github.com/gin-gonic/gin"
)

// ProductHandler menangani seluruh HTTP request yang berkaitan dengan entitas Produk (CRUD).
type ProductHandler struct {
	store store.Store
}

// NewProductHandler membuat instance ProductHandler baru dengan menginjeksi interface Store.
func NewProductHandler(s store.Store) *ProductHandler {
	return &ProductHandler{store: s}
}

// List mengambil seluruh data produk yang ada di sistem (GET /products).
//
// Konsep Belajar Golang & REST API:
// - Endpoint ini bersifat publik (dapat diakses siapa saja tanpa perlu login/token).
// - Mengembalikan HTTP 200 OK beserta array produk dalam format JSON.
func (h *ProductHandler) List(c *gin.Context) {
	products := h.store.GetAll()
	c.JSON(http.StatusOK, products)
}

// Get mengambil detail satu produk berdasarkan parameter ID di URL (GET /products/:id).
//
// Konsep Belajar Golang:
//   - `c.Param("id")`: Mengambil nilai variabel dari path URL.
//   - `strconv.Atoi`: Mengonversi string ke integer. Jika bukan angka valid atau <= 0,
//     kembalikan respons HTTP 400 Bad Request.
func (h *ProductHandler) Get(c *gin.Context) {
	idVal, err := strconv.Atoi(c.Param("id"))
	if err != nil || idVal <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid, harus angka positif"})
		return
	}

	product, err := h.store.GetByID(uint(idVal))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
			return
		}
		slog.Error("Database error saat mengambil produk", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan internal"})
		return
	}

	c.JSON(http.StatusOK, product)
}

// Create menambahkan data produk baru ke sistem (POST /products).
//
// Konsep Belajar Golang & Keamanan:
//   - Endpoint ini terproteksi oleh AuthMiddleware (wajib menyertakan Bearer Token).
//   - Validasi berlapis: `ShouldBindJSON` memeriksa format JSON, dan `input.Validate()`
//     memeriksa aturan bisnis (harga tidak minus, stok tidak minus, nama tidak kosong).
//   - Mengembalikan HTTP 201 Created jika produk berhasil disimpan.
func (h *ProductHandler) Create(c *gin.Context) {
	var input models.Product
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format body request tidak valid"})
		return
	}

	if err := input.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.store.Create(input)
	if err != nil {
		slog.Error("Database error saat menyimpan produk baru", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan internal"})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// Update mengubah data produk yang sudah ada secara keseluruhan (PUT /products/:id).
//
// Konsep Belajar Golang & REST API:
// - Dalam standar REST, method `PUT` digunakan untuk mengganti resource secara lengkap.
// - Menghasilkan HTTP 404 Not Found jika ID produk yang ingin diubah tidak ada di database.
func (h *ProductHandler) Update(c *gin.Context) {
	idVal, err := strconv.Atoi(c.Param("id"))
	if err != nil || idVal <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid, harus angka positif"})
		return
	}

	var input models.Product
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format body request tidak valid"})
		return
	}

	if err := input.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.store.Update(uint(idVal), input)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
			return
		}
		slog.Error("Database error saat memperbarui produk", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan internal"})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// Delete menghapus produk berdasarkan ID (DELETE /products/:id).
//
// Konsep Belajar Golang & REST API:
//   - Menggunakan Soft Delete (data ditandai deleted_at dan tidak terhapus permanen dari MySQL).
//   - Sesuai standar RFC RESTful API, operasi delete yang sukses mengembalikan status
//     HTTP 204 No Content (artinya sukses, dan tidak ada body response yang perlu dikirim).
func (h *ProductHandler) Delete(c *gin.Context) {
	idVal, err := strconv.Atoi(c.Param("id"))
	if err != nil || idVal <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid, harus angka positif"})
		return
	}

	if err := h.store.Delete(uint(idVal)); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
			return
		}
		slog.Error("Database error saat menghapus produk", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan internal"})
		return
	}

	c.Status(http.StatusNoContent)
}
