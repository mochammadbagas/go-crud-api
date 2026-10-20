package routes

import (
	"crudapi/internal/handlers"
	"crudapi/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Setup menginisialisasi router Gin, mendaftarkan middleware global,
// dan memetakan seluruh kelompok endpoint API (Routing Layer).
//
// Konsep Belajar Golang & Clean Architecture:
//  1. Separation of Concerns:
//     Memisahkan routing ke file tersendiri membuat `main.go` tidak tercampur dengan
//     daftar puluhan rute endpoint HTTP, sehingga kode sangat mudah dirawat dan diperluas.
//  2. Route Grouping:
//     Mengelompokkan rute berdasarkan aksesibilitas (Public vs Protected JWT).
func Setup(
	productHandler *handlers.ProductHandler,
	authHandler *handlers.AuthHandler,
	jwtSecret string,
) *gin.Engine {
	// 1. Inisialisasi router engine baru
	r := gin.New()

	// 2. Pasang middleware global:
	// - gin.Logger(): Mencatat log latensi dan status HTTP request ke console terminal
	// - gin.Recovery(): Menangkap error panic runtime sehingga aplikasi tidak crash
	// - middleware.CORS(): Mengizinkan permintaan lintas domain dari browser frontend
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// 3. Rute Publik Autentikasi
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)

	// 4. Rute Profil User (Memerlukan Token JWT)
	r.GET("/me", middleware.AuthMiddleware(jwtSecret), authHandler.Me)

	// 5. Kelompok Rute Produk
	// - Read Operations (Publik): Siapa saja bisa melihat katalog produk
	r.GET("/products", productHandler.List)
	r.GET("/products/:id", productHandler.Get)

	// - Write Operations (Terproteksi JWT): Hanya user yang sudah login yang boleh membuat, mengubah, atau menghapus
	protected := r.Group("")
	protected.Use(middleware.AuthMiddleware(jwtSecret))
	{
		protected.POST("/products", productHandler.Create)
		protected.PUT("/products/:id", productHandler.Update)
		protected.DELETE("/products/:id", productHandler.Delete)
	}

	return r
}
