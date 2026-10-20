package models

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Category merepresentasikan model entitas untuk kategori produk di database.
//
// Konsep Belajar Golang & GORM:
// - Struct Tag (`gorm:"..."`): Memberitahu GORM bagaimana kolom ini harus dibuat di tabel MySQL.
//   - `primaryKey`: Menjadikan field ID sebagai Primary Key.
//   - `autoIncrement`: Menjadikan nomor ID bertambah secara otomatis (1, 2, 3...).
//   - `type:varchar(255);not null;unique`: Menentukan tipe data SQL dan constraint unik (nama kategori tidak boleh kembar).
//   - Struct Tag (`json:"..."`): Mengatur bagaimana nama field ini ditulis ketika diubah menjadi JSON (Serialization).
//   - `gorm.DeletedAt`: Mengaktifkan fitur Soft Delete bawaan GORM.
//     Ketika record dihapus, record tidak benar-benar hilang dari database, melainkan
//     kolom `deleted_at` diisi timestamp saat penghapusan. Query selanjutnya otomatis menyaringnya.
type Category struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"type:varchar(255);not null;unique" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Validate memeriksa apakah data kategori yang dimasukkan sudah memenuhi syarat bisnis.
//
// Konsep Belajar Golang:
// - Di Go, validasi model umum diletakkan sebagai method pada struct terkait.
// - Fungsi ini mengembalikan `nil` jika data valid, atau mengembalikan `error` jika tidak valid.
// - `strings.TrimSpace` memastikan nama tidak hanya berisi karakter spasi kosong ("   ").
func (c Category) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("nama kategori wajib diisi")
	}
	return nil
}
