package models

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User merepresentasikan data akun pengguna di dalam sistem untuk keperluan autentikasi.
//
// Konsep Belajar Golang:
// 1. Tag `json:"-"`:
//   - Sangat penting dalam aspek keamanan! Tanda `-` memberitahu parser JSON untuk
//     TIDAK PERNAH menyertakan field `Password` pada respon JSON API apa pun (mencegah kebocoran data).
//
// 2. Tag `gorm:"uniqueIndex"`:
//   - Membuat indeks unik pada level database untuk kolom `email`, sehingga database MySQL
//     menolak otomatis jika ada dua user dengan email yang sama.
type User struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Email     string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	Password  string         `gorm:"type:varchar(255);not null" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Validate memeriksa format data masukan akun pengguna.
//
// Konsep Belajar Golang:
// - Memastikan email memiliki format dasar (misal tidak kosong dan memiliki karakter '@').
// - Memastikan panjang password memenuhi standar keamanan minimal (>= 6 karakter).
func (u User) Validate() error {
	trimmedEmail := strings.TrimSpace(u.Email)
	if trimmedEmail == "" {
		return errors.New("email wajib diisi")
	}
	if !strings.Contains(trimmedEmail, "@") {
		return errors.New("format email tidak valid")
	}
	if len(u.Password) < 6 {
		return errors.New("password minimal harus 6 karakter")
	}
	return nil
}

// BeforeSave adalah Lifecycle Hook bawaan GORM.
// Fungsi ini otomatis dipanggil oleh GORM secara tepat sebelum proses INSERT atau UPDATE dieksekusi.
//
// Konsep Belajar Golang & Keamanan:
//   - Mengapa menggunakan Pointer Receiver (*User)?
//     Karena kita ingin mengubah isi field `u.Password` menjadi versi hash-nya secara in-place.
//   - Pemeriksaan `len(u.Password) < 60`:
//     Algoritma Bcrypt selalu menghasilkan string hash dengan panjang tepat 60 karakter.
//     Pengecekan ini mencegah double-hashing (password yang sudah berupa hash tidak sengaja di-hash ulang).
func (u *User) BeforeSave(tx *gorm.DB) error {
	if u.Password != "" && len(u.Password) < 60 {
		hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashed)
	}
	return nil
}

// CheckPassword membandingkan plain-text password yang dikirim user saat login
// dengan password hash yang tersimpan aman di database.
//
// Konsep Belajar Golang:
//   - Bcrypt menggunakan salt acak di dalamnya, sehingga dua string yang sama menghasilkan
//     hash yang berbeda. Kita tidak bisa membandingkan dengan `==`, melainkan harus memakai
//     fungsi `bcrypt.CompareHashAndPassword`.
//   - Mengembalikan `nil` jika cocok, atau mengembalikan error jika tidak cocok.
func (u *User) CheckPassword(plainPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plainPassword))
}
