package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS (Cross-Origin Resource Sharing) adalah middleware untuk mengizinkan aplikasi web frontend
// (seperti React, Vue, Next.js, Vite) mengakses REST API ini meskipun berbeda domain atau beda port.
//
// Konsep Belajar Web & Golang:
//  1. Kebijakan Same-Origin Policy (SOP):
//     Secara default, browser memblokir request HTTP via JavaScript (Fetch/Axios) jika domain/port frontend
//     berbeda dengan backend (contoh: frontend di http://localhost:5173 dan backend di http://localhost:8080).
//  2. Pre-flight Request (HTTP OPTIONS):
//     Sebelum browser mengirim request berpotensi berbahaya (seperti POST, PUT, DELETE dengan custom headers),
//     browser akan otomatis mengirim request "uji coba" bertipe OPTIONS.
//     Middleware ini mendeteksi method OPTIONS dan langsung mengembalikan HTTP 204 No Content dengan header izin CORS.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Mengizinkan semua origin domain mengakses API (Gunakan spesifik domain di production)
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		// Menangani Pre-flight Request
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
