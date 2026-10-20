package models

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Product merepresentasikan entitas barang/produk di sistem e-commerce/inventaris.
//
// Konsep Belajar Golang & GORM:
// 1. Relasi Belongs-To:
//   - Field `CategoryID` menyimpan ID foreign key ke tabel `categories`.
//   - Field `Category *Category` adalah pointer struct yang menampung relasi objek kategori.
//   - Mengapa menggunakan POINTER (*Category)?
//     Karena relasi bisa bernilai kosong/null, dan pointer mempermudah kita mendeteksi
//     apakah data kategori sudah di-load (tidak nil) atau belum (nil), sekaligus menghindari circular copy.
//
// 2. GORM Constraint Tag:
//   - `constraint:OnUpdate:CASCADE,OnDelete:SET NULL;` memastikan jika kategori dihapus,
//     maka produk yang bersangkutan tidak ikut terhapus fatal, melainkan category_id-nya diset NULL.
//
// 3. `json:",omitempty"`:
//   - Memberitahu serializer JSON untuk menyembunyikan field tersebut jika bernilai nol/kosong/nil.
type Product struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	Price      float64        `gorm:"type:decimal(10,2);not null" json:"price"`
	Stock      int            `gorm:"type:int;not null" json:"stock"`
	CategoryID uint           `json:"category_id"`
	Category   *Category      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"category,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Validate memeriksa kelayakan data produk sebelum disimpan atau diperbarui ke database.
//
// Konsep Belajar Golang:
//   - Validasi berlapis: Validasi di level Model bertindak sebagai pertahanan pertama
//     sebelum data menyentuh database query layer.
//   - Menghindari magic numbers atau kondisi tak logis (seperti harga atau stok bertanda minus).
func (p Product) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("nama produk wajib diisi")
	}
	if p.Price < 0 {
		return errors.New("harga produk tidak boleh kurang dari 0")
	}
	if p.Stock < 0 {
		return errors.New("stok produk tidak boleh kurang dari 0")
	}
	if p.CategoryID == 0 {
		return errors.New("kategori produk wajib diisi")
	}
	return nil
}
