# Panduan Integrasi dan Penggunaan Resend Mailer

Dokumen ini menjelaskan langkah demi langkah integrasi dan penggunaan layanan email **Resend** pada proyek **Go CRUD API Starter Kit**.

---

## 1. Arsitektur Mailer Proyek

Proyek ini menggunakan arsitektur pluggable mailer yang didefinisikan dalam interface `mailer.Mailer` (`internal/mailer/mailer.go`):
- **`log` (Development)**: Mencetak seluruh payload email (HTML, tautan verifikasi, token) langsung ke terminal konsol tanpa mengirim email sungguhan.
- **`resend` (Production / Staging)**: Mengirimkan email transaksional sungguhan menggunakan REST API dari [Resend](https://resend.com) via HTTP POST (`https://api.resend.com/emails`).

Jika `MAILER_DRIVER=resend` dipilih tetapi `RESEND_API_KEY` belum diisi, sistem secara otomatis melakukan *fallback* ke `LogMailer` dengan pesan peringatan di log.

---

## 2. Langkah 1: Pendaftaran Akun Resend

1. Buka situs resmi [https://resend.com](https://resend.com).
2. Klik tombol **Get Started** atau **Sign Up**.
3. Daftarkan akun menggunakan email Anda atau login langsung via akun GitHub.
4. Lakukan verifikasi email pendaftaran sesuai petunjuk yang dikirimkan ke kotak masuk Anda.

---

## 3. Langkah 2: Membuat API Key

1. Setelah masuk ke Dashboard Resend, pilih menu **API Keys** pada navigasi sebelah kiri (atau buka [https://resend.com/api-keys](https://resend.com/api-keys)).
2. Klik tombol **Create API Key**.
3. Isi informasi API Key:
   - **Name**: Beri nama deskriptif, misalnya `go-crud-api-dev` atau `go-crud-api-prod`.
   - **Permission**: Pilih **Full access** (atau **Sending access** jika hanya dibatasi untuk pengiriman email).
   - **Domain**: Pilih `All domains` (atau pilih domain khusus jika sudah ada).
4. Klik **Add**.
5. **PENTING:** Salin nilai API Key yang muncul (berawalan `re_...`). Simpan di tempat yang aman karena key ini hanya ditampilkan sekali.

---

## 4. Langkah 3: Menentukan Sender Domain (Testing vs Production)

Resend menerapkan kebijakan keamanan pengiriman email sebagai berikut:

### Opsi A: Mode Testing / Sandbox (Gratis & Cepat)
Jika Anda belum memiliki domain sendiri atau baru ingin mencoba di lingkungan lokal:
- Gunakan alamat pengirim bawaan Resend: `onboarding@resend.dev`.
- **Batasan Sandbox Resend:** Email hanya dapat dikirim ke **alamat email yang sama dengan email yang terdaftar pada akun Resend Anda**.
- Jika Anda mencoba mengirim ke email lain saat memakai `onboarding@resend.dev`, Resend akan mengembalikan HTTP 403:  
  `"You can only send testing emails to your own email address (...)"`.

### Opsi B: Mode Production (Domain Sendiri / Custom Domain)
Untuk mengirim email ke pengguna publik/siapa saja tanpa batasan:
1. Di Dashboard Resend, klik menu **Domains** -> **Add Domain**.
2. Masukkan nama domain atau subdomain Anda, misalnya: `mail.domainanda.com` atau `domainanda.com`.
3. Pilih region (misal: `us-east-1` atau region terdekat).
4. Klik **Add Domain**.
5. Resend akan menampilkan daftar DNS Records:
   - **DKIM** (TXT / CNAME)
   - **SPF / Return-Path** (MX / TXT)
6. Buka DNS Management penyedia domain Anda (Cloudflare, Niagahoster, Namecheap, GoDaddy, Route53, dll.) dan tambahkan record yang diberikan.
7. Kembali ke Resend dan klik **Verify DNS Records**. Tunggu beberapa menit hingga status berubah menjadi **Verified**.
8. Setelah terverifikasi, Anda dapat mengirim dari alamat mana pun dengan domain tersebut, contoh: `no-reply@domainanda.com` atau `support@mail.domainanda.com`.

---

## 5. Langkah 4: Konfigurasi File `.env`

Buka file `.env` di direktori root proyek Anda, lalu sesuaikan variabel berikut:

### Contoh Konfigurasi untuk Testing (Akun Baru / Sandbox):
```dotenv
# Driver mailer
MAILER_DRIVER=resend

# API Key dari Dashboard Resend
RESEND_API_KEY=re_123456789_abcdefghijklmnopqrstuvwxyz

# Gunakan onboarding@resend.dev untuk testing
MAIL_FROM=Go Starter Kit <onboarding@resend.dev>

# URL Frontend (digunakan untuk menyusun tautan verifikasi & reset password)
FRONTEND_URL=http://localhost:3000
```

### Contoh Konfigurasi untuk Production (Domain Terverifikasi):
```dotenv
MAILER_DRIVER=resend
RESEND_API_KEY=re_prod_xxxxxxxxxxxxxxxxxxxxxxxx
MAIL_FROM=My Application <noreply@domainanda.com>
FRONTEND_URL=https://app.domainanda.com
```

---

## 6. Langkah 5: Fitur-fitur Aplikasi yang Menggunakan Mailer

Backend `go-crud-api` secara otomatis mengirimkan email transaksional dengan template HTML responsif pada alur berikut:

| No | Fitur | Endpoint API | Deskripsi & Isi Email |
| :--- | :--- | :--- | :--- |
| 1 | **Verifikasi Email Registrasi** | `POST /api/v1/auth/register` | Mengirim tautan aktivasi akun (`${FRONTEND_URL}/verify-email?token=...`) dengan masa berlaku 24 jam. |
| 2 | **Lupa Password (Reset Password)** | `POST /api/v1/auth/forgot-password` | Mengirim tautan reset kata sandi (`${FRONTEND_URL}/reset-password?token=...`) dengan masa berlaku 15 menit. |
| 3 | **Konfirmasi Penggantian Email** | `POST /api/v1/users/me/email/request` | Mengirim email konfirmasi ke alamat email **baru** (`${FRONTEND_URL}/confirm-email-change?token=...`) dengan masa berlaku 1 jam. |
| 4 | **Peringatan Keamanan Penggantian Email** | `POST /api/v1/users/me/email/request` | Mengirim email peringatan keamanan ke alamat email **lama** memberitahukan adanya permintaan perubahan email. |

---

## 7. Langkah 6: Menjalankan dan Menguji Pengiriman Email

### 1. Jalankan Server API
Jalankan aplikasi backend:
```bash
go run ./cmd/server
```

Pastikan pada log terminal muncul pesan inisialisasi:
```text
level=INFO msg="Menginisialisasi ResendMailer (Mode Production)" from="Go Starter Kit <onboarding@resend.dev>"
```

### 2. Uji Coba Pengiriman (Contoh: Forgot Password)

Kirimkan request HTTP ke endpoint `/api/v1/auth/forgot-password`.  
*(Catatan: Jika memakai `onboarding@resend.dev`, gunakan email yang sama dengan email akun Resend Anda)*:

```bash
curl -X POST http://localhost:8080/api/v1/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{
    "email": "email_anda_di_resend@example.com"
  }'
```

Response yang diharapkan:
```json
{
  "code": 200,
  "status": "OK",
  "message": "If the email is registered, password reset instructions have been sent."
}
```

### 3. Pantau Log Pengiriman di Resend Dashboard
1. Buka [https://resend.com/emails](https://resend.com/emails).
2. Anda akan melihat log email yang baru saja dikirim beserta statusnya (**Delivered**, **Bounced**, **Opened**, atau **Clicked**).
3. Anda juga dapat melihat pratinjau pesan HTML yang telah diterima.

---

## 8. Troubleshooting Masalah Umum

### 1. `resend API returned status 403: You can only send testing emails to your own email address`
- **Penyebab:** Anda menggunakan `MAIL_FROM=onboarding@resend.dev` tetapi mengirim email ke alamat tujuan yang berbeda dengan akun Resend Anda.
- **Solusi:**
  - Untuk pengetesan: Kirim hanya ke email akun Resend Anda.
  - Untuk umum: Daftarkan dan verifikasi domain sendiri di menu **Domains** pada Dashboard Resend.

### 2. `Peringatan: MAILER_DRIVER disetel 'resend' tetapi RESEND_API_KEY kosong. Fallback otomatis ke LogMailer.`
- **Penyebab:** Nilai `RESEND_API_KEY` belum diset di file `.env` atau bernilai string kosong.
- **Solusi:** Periksa kembali file `.env` dan pastikan server membaca `.env` yang benar.

### 3. `resend API returned status 401: API key is invalid`
- **Penyebab:** API Key salah atau telah dihapus/di-revoke dari dashboard Resend.
- **Solusi:** Buat API Key baru di dashboard Resend dan perbarui nilai `RESEND_API_KEY` di `.env`.

### 4. `resend API returned status 422: The domain is not verified`
- **Penyebab:** Menggunakan `MAIL_FROM` dengan domain kustom yang belum lolos verifikasi DNS.
- **Solusi:** Buka menu **Domains** di Resend dan pastikan semua DNS record (DKIM & SPF) sudah berstatus hijau (**Verified**).
