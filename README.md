# Go CRUD API - Product (Modern Production Standard)

REST API untuk mengelola data `Product` yang dikembangkan menggunakan **Go 1.25+** dengan arsitektur standar produksi modern menggunakan **Gin Gonic Framework** dan **GORM (MySQL)**.

Aplikasi ini dilengkapi dengan sistem **Autentikasi JWT (Token-Based)** untuk melindungi operasi modifikasi data produk.

## Fitur Utama
- **Gin Gonic Framework**: Menggunakan Gin untuk penanganan routing cepat, parser JSON, dan manajemen request/response context.
- **Autentikasi JWT (Stateless)**: Menggunakan token JWT via **golang-jwt/jwt/v5** untuk memproteksi endpoint write produk (`POST`, `PUT`, `DELETE`).
- **Enkripsi Bcrypt**: Menyimpan password pengguna secara aman menggunakan hash **Bcrypt** (`golang.org/x/crypto/bcrypt`).
- **Relasi Database (Belongs-To)**: Setiap Produk terikat pada satu Kategori (`category_id`) dengan GORM foreign key constraint.
- **GORM Preloading**: Secara otomatis memuat detail objek kategori pada response JSON produk (`GET`, `POST`, `PUT`) menggunakan `.Preload("Category")`.
- **Database & ORM**: Menggunakan **GORM** dengan database MySQL, lengkap dengan fitur *Connection Pooling*, *Auto Migration*, dan *Database Seeder* otomatis.
- **Soft Deletes**: Data produk yang dihapus tidak langsung hilang dari database melainkan ditandai via kolom `deleted_at`.
- **Structured Logging & Recovery**: Menggunakan middleware bawaan Gin yang andal untuk mencatat log dan menangkap *panic runtime error*.
- **CORS**: Mengizinkan akses API lintas domain/port (seperti frontend React/Vite).
- **Graceful Shutdown**: Server dapat dimatikan secara bersih ketika menerima sinyal OS (`SIGINT`/`Ctrl+C` atau `SIGTERM`).
- **Environment Configuration**: Pengaturan port, database, dan JWT Secret Key dikelola via file `.env`.
- **CLI Flag Database Reset**: Mendukung penghapusan seluruh tabel lama (`products`, `categories`, `users`) dan migrasi serta seeding ulang dari awal menggunakan flag `-fresh`.
- **Hot Reloading (Air)**: Mendukung *live-reload* kompilasi otomatis selama tahap development menggunakan **Air**.

---

## Struktur Folder

```text
go-crud-api/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point ringkas & High-level Orchestration
├── docs/
│   ├── technical_specification.md  # Spesifikasi teknis, ERD, & blueprint arsitektur
│   ├── learning_guide.md           # Panduan belajar mendalam arsitektur & konsep Golang
│   └── postman_collection.json     # Koleksi request Postman v2.1 siap import (Auto-Token)
├── internal/
│   ├── config/
│   │   └── config.go               # Loader konfigurasi dari .env / sistem operasi
│   ├── database/
│   │   ├── database.go             # Koneksi MySQL, connection pool, & auto-migration
│   │   └── seeder.go               # Database seeder awal kategori & produk
│   ├── handlers/
│   │   ├── auth_handler.go         # Controller registrasi, login & profil user
│   │   ├── auth_handler_test.go    # Unit test alur autentikasi JWT
│   │   ├── product_handler.go      # Controller penanganan HTTP request/response produk
│   │   └── product_handler_test.go # Unit test HTTP handler dengan InMemoryStore
│   ├── middleware/
│   │   ├── auth.go                 # Middleware proteksi token JWT
│   │   └── middleware.go           # Middleware CORS
│   ├── models/
│   │   ├── category.go             # Struct entity Category & validasinya
│   │   ├── product.go              # Struct entity Product & validasinya (relasi Category)
│   │   ├── product_test.go         # Table-driven test validasi entity Product
│   │   └── user.go                 # Struct entity User & GORM Bcrypt hooks
│   ├── routes/
│   │   └── routes.go               # Setup routing Gin & grouping rute publik/privat
│   └── store/
│       ├── store.go                # Interface/kontrak penyimpanan & InMemoryStore
│       └── mysql_store.go          # Implementasi query database menggunakan GORM (Preloaded)
├── .air.toml                       # Konfigurasi hot-reloading Air
├── .env                            # File konfigurasi lokal (tidak masuk git)
├── .env.example                    # Template file konfigurasi env
├── api.http                        # Koleksi request HTTP untuk uji coba via REST Client
├── go.mod                          # Manifest modul Go & daftar dependensi
└── go.sum                          # Checksum keamanan untuk dependensi Go
```

---

## Persyaratan Sistem
- **Go 1.25** atau versi terbaru.
- **MySQL Server** (bisa menggunakan XAMPP, Laragon, Docker, dsb.).

---

## Langkah Menjalankan Aplikasi

1. **Persiapan Database**
   Buat database baru bernama `go_crud_api` di server MySQL lokal Anda.

2. **Pengaturan Environment**
   Salin file `.env.example` menjadi `.env` lalu sesuaikan kredensial database Anda:
   ```bash
   cp .env.example .env
   ```

3. **Menjalankan Server**
   Ada dua cara untuk menjalankan server:

   * **Cara Biasa**:
     ```bash
     go run cmd/server/main.go
     ```
     *Saat pertama kali dijalankan, GORM akan otomatis membuat tabel-tabel dan melakukan seeding (semai) data awal.*

   * **Cara Hot Reloading (Rekomendasi saat Development)**:
     Gunakan **Air** agar server otomatis restart setiap kali Anda menyimpan perubahan kode.
     1. Instal Air secara global (sekali saja):
        ```bash
        go install github.com/air-verse/air@latest
        ```
     2. Jalankan server dengan perintah:
        ```bash
        air
        ```

   **Reset Database (seperti `migrate:fresh` di Laravel)**:
   Jika Anda mengubah skema database dan ingin menjatuhkan (drop) semua tabel lama serta mengulang migrasi & seeding dari awal, jalankan server dengan flag `-fresh`:
   - Cara Biasa: `go run cmd/server/main.go -fresh`
   - Cara Air: `air -- -fresh` (melewatkan flag ke binary Go melalui Air)

---

## Endpoint API

| Method | Path             | Deskripsi                                   | Autentikasi |
|--------|------------------|---------------------------------------------|-------------|
| **POST**   | `/register`        | Registrasi pengguna baru                    | Publik      |
| **POST**   | `/login`           | Login pengguna dan mendapatkan token JWT    | Publik      |
| **GET**    | `/me`              | Mendapatkan profil user yang sedang login   | **JWT Bearer** |
| **GET**    | `/products`        | Mengambil semua data produk                 | Publik      |
| **GET**    | `/products/:id`    | Mengambil satu produk berdasarkan ID        | Publik      |
| **POST**   | `/products`        | Menambahkan produk baru                     | **JWT Bearer** |
| **PUT**    | `/products/:id`    | Mengubah data produk                        | **JWT Bearer** |
| **DELETE** | `/products/:id`    | Menghapus produk (Soft Delete)              | **JWT Bearer** |

---

## Panduan Pengujian di Postman

> [!TIP]
> **Cara Paling Cepat (1-Klik Import)**:
> Anda tidak perlu membuat request satu per satu secara manual. Cukup buka aplikasi **Postman** -> klik tombol **Import** -> pilih file [`docs/postman_collection.json`](file:///c:/Users/mmdbc/Development/Personal/Sandbox/Golang/go-crud-api/docs/postman_collection.json).
> Seluruh rute akan otomatis terimpor rapi, dan token JWT akan **otomatis tersimpan ke variabel `{{token}}` saat Anda menjalankan request Login**!

> [!IMPORTANT]
> Jika mengonfigurasi Postman secara manual untuk request **POST** dan **PUT**:
> 1. Pilih tab **Body** di bawah kolom URL.
> 2. Pilih format **raw**.
> 3. Ubah dropdown teks di paling kanan dari **Text** menjadi **JSON** (header `Content-Type: application/json` akan terpasang otomatis).

### 1. Registrasi Pengguna (`POST /register`)
Daftarkan akun pengguna baru:
* **URL:** `http://localhost:8080/register`
* **Body (JSON):**
  ```json
  {
    "email": "user@gmail.com",
    "password": "password123"
  }
  ```
* **Response Contoh (201 Created):**
  ```json
  {
    "id": 1,
    "email": "user@gmail.com",
    "created_at": "2026-06-28T12:00:00Z",
    "updated_at": "2026-06-28T12:00:00Z"
  }
  ```

### 2. Login Pengguna (`POST /login`)
Login untuk mendapatkan token JWT:
* **URL:** `http://localhost:8080/login`
* **Body (JSON):**
  ```json
  {
    "email": "user@gmail.com",
    "password": "password123"
  }
  ```
* **Response Contoh (200 OK):**
  ```json
  {
    "token": "eyJhbGciOiJIUzI1NiIsIn..."
  }
  ```
  *Salin seluruh string token yang dikembalikan untuk digunakan pada request terproteksi.*

### 3. Cek Profil Saya (`GET /me`)
Mendapatkan info akun berdasarkan token:
* **URL:** `http://localhost:8080/me`
* **Headers di Postman**:
  * Pilih tab **Headers**, tambah key baru: `Authorization` dengan value `Bearer <TOKEN_YANG_ANDA_SALIN>`
  * *(Atau di Postman pilih tab **Auth** -> Type: **Bearer Token** -> Tempel token di kolom Token).*
* **Response Contoh (200 OK):**
  ```json
  {
    "id": 1,
    "email": "user@gmail.com",
    "created_at": "2026-06-28T12:00:00Z",
    "updated_at": "2026-06-28T12:00:00Z"
  }
  ```

### 4. Tambah Produk Baru (`POST /products` - Terproteksi)
Menambahkan produk baru. Request ini wajib menggunakan **Bearer Token** seperti langkah nomor 3.
* **URL:** `http://localhost:8080/products`
* **Headers**: `Authorization: Bearer <TOKEN>`
* **Body (JSON):**
  ```json
  {
    "name": "Headphone ANC",
    "price": 1200000,
    "stock": 10,
    "category_id": 1
  }
  ```
* **Response Contoh (201 Created):**
  ```json
  {
    "id": 4,
    "name": "Headphone ANC",
    "price": 1200000,
    "stock": 10,
    "category_id": 1,
    "category": {
      "id": 1,
      "name": "Elektronik",
      "created_at": "2026-06-28T06:23:00Z",
      "updated_at": "2026-06-28T06:23:00Z"
    },
    "created_at": "2026-06-28T12:45:00Z",
    "updated_at": "2026-06-28T12:45:00Z"
  }
  ```

### 5. Ubah Produk (`PUT /products/:id` - Terproteksi)
Mengubah data produk yang sudah ada. Wajib menyertakan **Bearer Token**.
* **URL:** `http://localhost:8080/products/1`
* **Headers**: `Authorization: Bearer <TOKEN>`
* **Body (JSON):**
  ```json
  {
    "name": "Keyboard Mekanikal Pro",
    "price": 600000,
    "stock": 15,
    "category_id": 2
  }
  ```

### 6. Hapus Produk (`DELETE /products/:id` - Terproteksi)
Menghapus produk menggunakan soft delete. Wajib menyertakan **Bearer Token**.
* **URL:** `http://localhost:8080/products/1`
* **Headers**: `Authorization: Bearer <TOKEN>`
* **Response Contoh (204 No Content):**
  * *Body kosong (Sukses).*

---

## Menjalankan Unit Test

Proyek ini telah dilengkapi dengan automated unit testing yang memanfaatkan **Table-Driven Tests** dan **In-Memory Store Mocking** (dapat dijalankan secara instan tanpa perlu menyalakan database MySQL):

- **Model Test (`internal/models/product_test.go`)**: Menguji validasi logika bisnis atribut produk (nama, harga, stok, relasi kategori).
- **Product Handler Test (`internal/handlers/product_handler_test.go`)**: Menguji pembuatan produk via HTTP POST menggunakan `httptest.ResponseRecorder` dan `InMemoryStore`.
- **Auth Handler Test (`internal/handlers/auth_handler_test.go`)**: Menguji alur registrasi, validasi email duplikat (409 Conflict), login verifikasi token JWT, hingga endpoint profil `/me`.

Jalankan seluruh pengujian dengan perintah:
```bash
go test ./... -v
```
