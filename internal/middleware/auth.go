package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims merepresentasikan struktur payload (klaim data) yang disimpan di dalam token JWT.
//
// Konsep Belajar Golang & JWT:
// - Komponen Token JWT:
//  1. Header: Algoritma tanda tangan (misal: HS256).
//  2. Payload (Claims): Data yang kita bawa (misal: UserID, waktu expired).
//  3. Signature: Tanda tangan digital menggunakan Secret Key untuk mencegah pemalsuan data.
//
// - `jwt.RegisteredClaims`: Standar klaim bawaan RFC 7519 (seperti `exp` untuk expiry, `iat` untuk issued-at).
type JWTClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthMiddleware bertindak sebagai gerbang pengamanan (Gatekeeper) pada rute-rute privat.
//
// Konsep Belajar Golang & Gin:
//  1. Higher-Order Function / Middleware Pattern:
//     Fungsi ini menerima parameter konfigurasi (`jwtSecret`) dan mengembalikan `gin.HandlerFunc`.
//  2. Kontrol Alur Middleware:
//     - `c.AbortWithStatusJSON(...)`: Menghentikan eksekusi rute seketika jika token tidak sah.
//     Handler berikutnya TIDAK AKAN dieksekusi.
//     - `c.Set("user_id", claims.UserID)`: Menyimpan data pengguna ke dalam context request Gin.
//     Ini memungkinkan controller/handler selanjutnya membaca siapa user yang sedang request (mirip `Auth::user()` di Laravel).
//     - `c.Next()`: Mengizinkan request melanjutkan ke middleware atau controller berikutnya.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Ambil nilai header "Authorization"
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "header Authorization wajib disertakan"})
			return
		}

		// 2. Periksa format token: Harus diawali kata "Bearer " diikuti token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "format token salah, gunakan: Bearer <token>"})
			return
		}

		tokenString := parts[1]
		claims := &JWTClaims{}

		// 3. Parse dan validasi keaslian tanda tangan token menggunakan JWT Secret
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Cegah serangan "Algorithm Confusion": Pastikan algoritma yang digunakan adalah HMAC (HS256)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode tanda tangan tidak sesuai: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		// 4. Jika ada error saat parsing atau masa berlaku token sudah habis
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token tidak valid atau telah kadaluarsa"})
			return
		}

		// 5. Sukses: Sisipkan ID pengguna ke context agar bisa diakses di handler
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
