package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"crudapi/internal/models"
	"crudapi/internal/store"

	"github.com/gin-gonic/gin"
)

// TestAuthHandler menguji alur autentikasi (Register, Login, Me) menggunakan InMemoryStore dan httptest.
func TestAuthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSecret := "test_secret_key"
	s := store.NewInMemoryStore()
	h := NewAuthHandler(s, jwtSecret)

	// 1. Uji Registrasi Berhasil
	t.Run("Register Sukses", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		body := map[string]string{
			"email":    "user@example.com",
			"password": "password123",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		h.Register(c)

		if rec.Code != http.StatusCreated {
			t.Errorf("ekspektasi status 201, didapatkan %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	// 2. Uji Registrasi Email Kembar (Duplikat)
	t.Run("Register Email Kembar Harus 409", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		body := map[string]string{
			"email":    "user@example.com", // Email yang sama dengan di atas
			"password": "password123",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		h.Register(c)

		if rec.Code != http.StatusConflict {
			t.Errorf("ekspektasi status 409 Conflict, didapatkan %d", rec.Code)
		}
	})

	// 3. Uji Login Berhasil
	t.Run("Login Sukses Menghasilkan Token", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		body := map[string]string{
			"email":    "user@example.com",
			"password": "password123",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		h.Login(c)

		if rec.Code != http.StatusOK {
			t.Errorf("ekspektasi status 200 OK, didapatkan %d", rec.Code)
		}

		var res map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		if res["token"] == "" {
			t.Errorf("token tidak boleh kosong")
		}
	})

	// 4. Uji Login Password Salah
	t.Run("Login Password Salah Harus 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		body := map[string]string{
			"email":    "user@example.com",
			"password": "wrongpassword",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		c.Request = req

		h.Login(c)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("ekspektasi status 401 Unauthorized, didapatkan %d", rec.Code)
		}
	})

	// 5. Uji Endpoint Me (Profil) dengan Context User
	t.Run("Me Mengembalikan Data Pengguna", func(t *testing.T) {
		// Buat user dummy langsung di store
		u, _ := s.CreateUser(models.User{Email: "me@example.com", Password: "hashedpassword"})

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		// Simulasi middleware menyuntikkan user_id ke context
		c.Set("user_id", u.ID)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		c.Request = req

		h.Me(c)

		if rec.Code != http.StatusOK {
			t.Errorf("ekspektasi status 200 OK, didapatkan %d", rec.Code)
		}
	})
}
