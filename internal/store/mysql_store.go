package store

import (
	"errors"

	"crudapi/internal/models"

	"gorm.io/gorm"
)

// MySQLStore adalah implementasi nyata dari interface Store menggunakan GORM ORM untuk database MySQL.
type MySQLStore struct {
	db *gorm.DB
}

// NewMySQLStore adalah fungsi konstruktor untuk menginisialisasi MySQLStore dengan pointer koneksi database GORM.
func NewMySQLStore(db *gorm.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

// ============================================================================
// METODE PRODUK (GORM / MySQL)
// ============================================================================

// GetAll mengambil semua data produk dari database MySQL yang belum di-soft-delete.
//
// Konsep Belajar Golang & GORM:
//   - `.Preload("Category")`: Secara otomatis melakukan Eager Loading (LEFT JOIN / query sekunder)
//     untuk mengisi objek relasi `Category` pada masing-masing produk, mencegah masalah query N+1.
func (s *MySQLStore) GetAll() []models.Product {
	var products []models.Product
	s.db.Preload("Category").Find(&products)
	return products
}

// GetByID mengambil satu data produk spesifik berdasarkan primary key ID.
//
// Konsep Belajar Golang & GORM:
//   - `errors.Is(err, gorm.ErrRecordNotFound)`: Cara idiomatik di Go modern untuk memeriksa
//     apakah record tidak ditemukan di database.
//   - Kita memetakan error internal GORM menjadi `store.ErrNotFound` agar layer handler
//     tetap decoupled (tidak bergantung langsung pada package GORM).
func (s *MySQLStore) GetByID(id uint) (models.Product, error) {
	var product models.Product
	err := s.db.Preload("Category").First(&product, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, err
	}
	return product, nil
}

// Create menyimpan data produk baru ke tabel MySQL.
//
// Konsep Belajar Golang & GORM:
//   - Setelah produk berhasil dibuat (`s.db.Create`), GORM secara otomatis mengisi field `p.ID`
//     dengan auto-increment ID yang baru dihasilkan database.
//   - Kita memanggil `.Preload("Category").First(...)` agar response JSON yang dikembalikan
//     ke pengguna langsung memuat detail nama kategori lengkap.
func (s *MySQLStore) Create(p models.Product) (models.Product, error) {
	if err := s.db.Create(&p).Error; err != nil {
		return models.Product{}, err
	}
	s.db.Preload("Category").First(&p, p.ID)
	return p, nil
}

// Update memperbarui atribut produk di database MySQL.
//
// Konsep Belajar Golang & GORM:
//   - `existing.Category = nil`: Sangat penting! Mengosongkan pointer struct asosiasi memberitahu
//     GORM bahwa kita hanya ingin memperbarui kolom scalar `category_id`, bukan menimpa data master tabel categories.
//   - `s.db.Save(&existing)`: Menjalankan perintah UPDATE SQL dengan semua perubahan kolom terbaru.
func (s *MySQLStore) Update(id uint, p models.Product) (models.Product, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return models.Product{}, err
	}

	// Menetapkan nilai atribut baru
	existing.Name = p.Name
	existing.Price = p.Price
	existing.Stock = p.Stock
	existing.CategoryID = p.CategoryID
	existing.Category = nil // Reset pointer asosiasi agar GORM fokus memperbarui foreign key

	if err := s.db.Save(&existing).Error; err != nil {
		return models.Product{}, err
	}

	// Muat ulang data terbaru beserta objek kategori terbarunya
	s.db.Preload("Category").First(&existing, existing.ID)
	return existing, nil
}

// Delete melakukan Soft Delete pada produk.
//
// Konsep Belajar Golang & GORM:
//   - Karena model `Product` memiliki field `DeletedAt gorm.DeletedAt`,
//     pemanggilan `s.db.Delete` TIDAK akan menghapus baris fisik dari tabel MySQL,
//     melainkan mengeksekusi: `UPDATE products SET deleted_at = NOW() WHERE id = ?`.
func (s *MySQLStore) Delete(id uint) error {
	existing, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.db.Delete(&existing).Error; err != nil {
		return err
	}
	return nil
}

// ============================================================================
// METODE PENGGUNA / AUTH (GORM / MySQL)
// ============================================================================

// CreateUser menyimpan data user baru ke database MySQL.
// Password akan di-hash secara otomatis melalui GORM BeforeSave hook pada model User.
func (s *MySQLStore) CreateUser(u models.User) (models.User, error) {
	if err := s.db.Create(&u).Error; err != nil {
		return models.User{}, err
	}
	return u, nil
}

// GetUserByEmail mencari entitas user berdasarkan kolom email.
func (s *MySQLStore) GetUserByEmail(email string) (models.User, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, err
	}
	return user, nil
}

// GetUserByID mencari entitas user berdasarkan Primary Key ID.
func (s *MySQLStore) GetUserByID(id uint) (models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, err
	}
	return user, nil
}
