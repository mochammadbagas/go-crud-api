package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"crudapi/internal/middleware"
	"crudapi/internal/models"
	"crudapi/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler menangani seluruh HTTP request yang berhubungan dengan autentikasi (Registrasi, Login, Profile).
//
// Konsep Belajar Golang & Dependency Injection:
//   - `store.Store`: Kita menyuntikkan interface `Store`, bukan struct konkrit database.
//     Ini membuat AuthHandler sangat modular dan mudah diuji secara terisolasi (Unit Testing).
//   - `jwtSecret`: Kunci rahasia untuk menandatangani token JWT disuntikkan dari layer konfigurasi.
type AuthHandler struct {
	store     store.Store
	jwtSecret string
}

// NewAuthHandler adalah fungsi konstruktor untuk membuat instance AuthHandler baru.
func NewAuthHandler(s store.Store, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: s, jwtSecret: jwtSecret}
}

// RegisterInput mendefinisikan struktur payload JSON yang diharapkan saat user mendaftar.
// Tag `binding:"required"` adalah validator otomatis bawaan Gin (via package go-playground/validator).
type RegisterInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginInput mendefinisikan struktur payload JSON saat user melakukan login.
type LoginInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register menangani pembuatan akun pengguna baru (POST /register).
//
// Alur Proses:
// 1. Parsing request body JSON ke struct RegisterInput menggunakan `c.ShouldBindJSON`.
// 2. Validasi format email dan panjang password melalui `models.User.Validate()`.
// 3. Simpan data user ke database; jika email kembar, kembalikan HTTP 409 Conflict.
// 4. Jika berhasil, kembalikan HTTP 201 Created beserta data user baru (password otomatis tersembunyi).
func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format body request tidak valid"})
		return
	}

	user := models.User{
		Email:    input.Email,
		Password: input.Password,
	}

	// Validasi aturan bisnis model
	if err := user.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.store.CreateUser(user)
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": "email sudah terdaftar"})
			return
		}
		slog.Error("Database error saat registrasi user", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan internal"})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// Login memverifikasi email & password, lalu mengembalikan token JWT jika valid (POST /login).
//
// Alur Proses:
// 1. Cari user di database berdasarkan email.
// 2. Bandingkan password plain dengan hash Bcrypt menggunakan `user.CheckPassword()`.
// 3. Jika cocok, bentuk JWT Claims berisi UserID dan masa berlaku (24 jam).
// 4. Tandatangani token dengan algoritma HS256 dan kunci rahasia (jwtSecret).
// 5. Kembalikan string token kepada client dengan HTTP 200 OK.
func (h *AuthHandler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format body request tidak valid"})
		return
	}

	user, err := h.store.GetUserByEmail(input.Email)
	if err != nil {
		// Demi keamanan, jangan beri tahu apakah email yang salah atau password yang salah
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "email atau password salah"})
			return
		}
		slog.Error("Database error saat login", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan internal"})
		return
	}

	// Bandingkan password
	if err := user.CheckPassword(input.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email atau password salah"})
		return
	}

	// Siapkan klaim token JWT dengan masa aktif 24 jam
	tokenClaims := middleware.JWTClaims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Buat token dan tanda-tangani
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		slog.Error("Gagal menanda-tangani token JWT", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// Me mengembalikan data profil pengguna yang saat ini sedang login (GET /me).
//
// Konsep Belajar Golang & Gin:
// - Endpoint ini berada di balik `AuthMiddleware`.
// - `c.Get("user_id")`: Membaca user ID yang sebelumnya telah disisipkan oleh middleware ke context.
// - Type Assertion `userIDVal.(uint)`: Di Go, interface{} harus di-cast ke tipe aslinya (uint) sebelum digunakan.
func (h *AuthHandler) Me(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID := userIDVal.(uint)
	user, err := h.store.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user tidak ditemukan"})
			return
		}
		slog.Error("Database error saat mencari user", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan internal"})
		return
	}

	c.JSON(http.StatusOK, user)
}
