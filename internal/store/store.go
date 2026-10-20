package store

import (
	"errors"
	"sync"

	"crudapi/internal/models"
)

// Sentinel Errors untuk layer penyimpanan (Store).
//
// Konsep Belajar Golang:
//   - Sentinel Error adalah variabel error statis (diawali dengan "Err...") yang didefinisikan
//     di level package agar caller (seperti HTTP Handler) bisa mengecek tipe error spesifik
//     menggunakan `errors.Is(err, store.ErrNotFound)`.
var (
	ErrNotFound   = errors.New("record not found")
	ErrEmailTaken = errors.New("email already taken")
)

// Store mendefinisikan interface (kontrak) untuk seluruh operasi database.
//
// Konsep Belajar Golang & Clean Architecture:
//  1. Interface Segregation & Repository Pattern:
//     Handler tidak peduli apakah data disimpan di MySQL, PostgreSQL, MongoDB, atau Memory.
//     Handler hanya bergantung pada interface `Store` ini.
//  2. Idiom Go: "Accept interfaces, return structs".
//     Dengan cara ini, pengujian unit test (unit testing) dapat dengan mudah menggunakan
//     `InMemoryStore` tanpa perlu menyalakan database MySQL sungguhan!
type Store interface {
	// Operasi CRUD Produk
	GetAll() []models.Product
	GetByID(id uint) (models.Product, error)
	Create(p models.Product) (models.Product, error)
	Update(id uint, p models.Product) (models.Product, error)
	Delete(id uint) error

	// Operasi Pengguna (User/Auth)
	CreateUser(u models.User) (models.User, error)
	GetUserByEmail(email string) (models.User, error)
	GetUserByID(id uint) (models.User, error)
}

// InMemoryStore adalah implementasi interface Store yang menyimpan data di RAM (memori lokal).
// Sangat ideal digunakan untuk Unit Testing atau prototyping cepat.
//
// Konsep Belajar Golang:
//  1. Concurrency Safety dengan `sync.RWMutex`:
//     Server HTTP di Go melayani setiap request dalam Goroutine terpisah. Jika dua request
//     mengakses atau menulis ke `map` secara bersamaan tanpa proteksi mutex, program akan mengalami panic `concurrent map read and write`.
//  2. RWMutex (Read-Write Mutex):
//     - `RLock()` / `RUnlock()`: Mengizinkan banyak goroutine membaca (read) sekaligus secara paralel.
//     - `Lock()` / `Unlock()`: Mengunci eksklusif saat ada satu goroutine yang sedang menulis/mengubah (write) data.
type InMemoryStore struct {
	mu         sync.RWMutex
	data       map[uint]models.Product
	nextID     uint
	users      map[string]models.User
	usersByID  map[uint]models.User
	nextUserID uint
}

// NewInMemoryStore adalah fungsi konstruktor (Factory Function) untuk membuat instance InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data:       make(map[uint]models.Product),
		nextID:     1,
		users:      make(map[string]models.User),
		usersByID:  make(map[uint]models.User),
		nextUserID: 1,
	}
}

// ============================================================================
// METODE STORE UNTUK PRODUK (InMemoryStore)
// ============================================================================

// GetAll mengembalikan salinan seluruh produk yang tersimpan di memori.
//
// Konsep Belajar Golang:
//   - `make([]models.Product, 0, len(s.data))`: Alokasi slice dengan kapasitas (capacity)
//     yang sudah ditentukan di awal. Ini mencegah re-alokasi memori berulang kali saat looping `append`.
func (s *InMemoryStore) GetAll() []models.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Product, 0, len(s.data))
	for _, p := range s.data {
		result = append(result, p)
	}
	return result
}

// GetByID mengambil satu produk berdasarkan ID uniknya.
func (s *InMemoryStore) GetByID(id uint) (models.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.data[id]
	if !ok {
		return models.Product{}, ErrNotFound
	}
	return p, nil
}

// Create menambahkan produk baru ke dalam map dan memberikan ID otomatis berurutan.
func (s *InMemoryStore) Create(p models.Product) (models.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p.ID = s.nextID
	s.nextID++
	s.data[p.ID] = p
	return p, nil
}

// Update memperbarui data produk yang ada berdasarkan ID-nya.
func (s *InMemoryStore) Update(id uint, p models.Product) (models.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[id]; !ok {
		return models.Product{}, ErrNotFound
	}
	p.ID = id
	s.data[id] = p
	return p, nil
}

// Delete menghapus produk dari map berdasarkan ID.
func (s *InMemoryStore) Delete(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[id]; !ok {
		return ErrNotFound
	}
	delete(s.data, id)
	return nil
}

// ============================================================================
// METODE STORE UNTUK PENGGUNA (InMemoryStore)
// ============================================================================

// CreateUser menyimpan data user baru ke dalam memori.
// Memeriksa apakah email sudah terdaftar sebelumnya untuk mencegah duplikasi.
func (s *InMemoryStore) CreateUser(u models.User) (models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[u.Email]; exists {
		return models.User{}, ErrEmailTaken
	}

	// Simulasi GORM BeforeSave hook untuk mengenkripsi password dengan Bcrypt
	_ = u.BeforeSave(nil)

	u.ID = s.nextUserID
	s.nextUserID++
	s.users[u.Email] = u
	s.usersByID[u.ID] = u
	return u, nil
}

// GetUserByEmail mencari data user berdasarkan alamat email.
func (s *InMemoryStore) GetUserByEmail(email string) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, exists := s.users[email]
	if !exists {
		return models.User{}, ErrNotFound
	}
	return u, nil
}

// GetUserByID mencari data user berdasarkan ID numerik.
func (s *InMemoryStore) GetUserByID(id uint) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, exists := s.usersByID[id]
	if !exists {
		return models.User{}, ErrNotFound
	}
	return u, nil
}
