package models

import "testing"

// TestProductValidation menguji fungsi validasi pada struct Product.
//
// Konsep Belajar Golang: Table-Driven Tests
// - Table-Driven Testing adalah pola pengujian paling populer dan direkomendasikan di Go.
// - Kumpulan kasus uji didefinisikan dalam bentuk slice of struct (`[]struct`).
// - Setiap kasus uji memiliki nama, input data, dan nilai ekspektasi (`wantErr`).
// - Memudahkan penambahan variasi skenario baru tanpa menduplikasi kode logika pengujian.
func TestProductValidation(t *testing.T) {
	tests := []struct {
		name    string
		product Product
		wantErr bool
	}{
		{
			name: "Produk valid",
			product: Product{
				Name:       "Monitor 24 Inch",
				Price:      1850000,
				Stock:      10,
				CategoryID: 1,
			},
			wantErr: false,
		},
		{
			name: "Nama produk kosong harus error",
			product: Product{
				Name:       "",
				Price:      1850000,
				Stock:      10,
				CategoryID: 1,
			},
			wantErr: true,
		},
		{
			name: "Nama produk hanya spasi harus error",
			product: Product{
				Name:       "   ",
				Price:      1850000,
				Stock:      10,
				CategoryID: 1,
			},
			wantErr: true,
		},
		{
			name: "Harga negatif harus error",
			product: Product{
				Name:       "Monitor 24 Inch",
				Price:      -1000,
				Stock:      10,
				CategoryID: 1,
			},
			wantErr: true,
		},
		{
			name: "Stok negatif harus error",
			product: Product{
				Name:       "Monitor 24 Inch",
				Price:      1850000,
				Stock:      -5,
				CategoryID: 1,
			},
			wantErr: true,
		},
		{
			name: "CategoryID tidak diisi harus error",
			product: Product{
				Name:       "Monitor 24 Inch",
				Price:      1850000,
				Stock:      10,
				CategoryID: 0,
			},
			wantErr: true,
		},
	}

	// Looping setiap skenario menggunakan sub-test `t.Run`
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.product.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Product.Validate() error = %v, ekspektasi wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
