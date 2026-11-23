# Atlas Financial Core

Atlas Financial Core adalah backend service untuk sistem finansial inti yang dibangun menggunakan Go (`net/http`, `log/slog`) dan PostgreSQL (`pgx/v5`).

---

## 🚀 Cara Menjalankan

### Prasyarat
- **Go**: Versi **1.26.5** atau yang lebih baru (sesuai `backend/go.mod`).
- **PostgreSQL**: Database server lokal atau cloud (misal Neon DB / Supabase).
- **golang-migrate**: CLI untuk menjalankan database migration.
  ```powershell
  # Instalasi migrate CLI dengan driver PostgreSQL:
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```

---

### 1. Konfigurasi Environment Variable (`.env`)
Service membaca konfigurasi dari file `.env` di dalam direktori `backend/` atau dari system environment variables.

Buat atau edit file `backend/.env`:
```env
PORT=8080
DATABASE_URL=postgresql://user:password@localhost:5432/atlas_core?sslmode=disable
```

| Variable | Wajib | Default | Penjelasan |
|---|---|---|---|
| `PORT` | Tidak | `8080` | Port HTTP server. |
| `DATABASE_URL` | **Ya** | - | PostgreSQL connection string (`postgresql://<user>:<password>@<host>:<port>/<dbname>?sslmode=<mode>`). |

> ⚠️ **PENTING:** Aplikasi mewajibkan koneksi database aktif saat startup. Jika `DATABASE_URL` kosong atau database tidak dapat dijangkau, aplikasi akan log error dan langsung berhenti (`os.Exit(1)`).

---

### 2. Menjalankan Database Migration
File skrip migrasi SQL tersimpan di direktori `migrations/` pada root proyek:
- `000001_create_accounts`: Membuat tabel `accounts` dan constraint saldo non-negatif.
- `000002_update_accounts`: Mengatur default UUID, tipe timestamp with time zone, dan panjang currency.
- `000003_update_accounts`: Menetapkan constraint `NOT NULL` pada `owner_name` dan `currency`.

Jalankan perintah berikut dari **root direktori** proyek:

#### Menjalankan Migration ke Versi Terbaru (`up`)

**Windows (PowerShell):**
```powershell
# Jika DATABASE_URL sudah diset di environment:
migrate -path migrations -database $env:DATABASE_URL up

# Atau menggunakan connection string langsung:
migrate -path migrations -database "postgresql://user:password@localhost:5432/atlas_core?sslmode=disable" up
```

**Linux / macOS (Bash):**
```bash
migrate -path migrations -database "$DATABASE_URL" up
```

#### Perintah Migration Lainnya

- **Rollback 1 migrasi (`down 1`):**
  ```powershell
  migrate -path migrations -database $env:DATABASE_URL down 1
  ```

- **Mengecek versi migrasi saat ini (`version`):**
  ```powershell
  migrate -path migrations -database $env:DATABASE_URL version
  ```

---

### 3. Menjalankan Aplikasi
Buka terminal dan navigasikan ke direktori `backend`:

```powershell
cd backend
go run ./cmd/api
```

> ⚠️ **PENTING:** Gunakan `go run ./cmd/api` (menunjuk folder package), **bukan** `go run cmd/api/main.go`. Menjalankan hanya satu file `main.go` tidak akan meng-compile file lain yang berada dalam package yang sama.

### 4. Build Binary (Opsional)
Jika ingin mem-build binary production:

**Windows (PowerShell):**
```powershell
go build -o bin/api.exe ./cmd/api
.\bin\api.exe
```

**Linux / macOS:**
```bash
go build -o bin/api ./cmd/api
./bin/api
```

---

## 📡 Endpoint API & Contoh Request

Base URL: `http://localhost:8080`

### 1. Health Check

Memverifikasi apakah server berjalan.

- **URL:** `/health`
- **Method:** `GET`
- **Contoh Request:**
  ```bash
  curl -i http://localhost:8080/health
  ```
- **Response:** `200 OK`
  ```text
  OK
  ```

---

### 2. Buat Akun Baru (`Create Account`)

Membuat akun rekening baru dengan saldo awal.

- **URL:** `/accounts`
- **Method:** `POST`
- **Header:** `Content-Type: application/json`
- **Validation Rules:**
  - `owner_name`: Wajib diisi (string tidak boleh kosong).
  - `currency`: Wajib diisi, format kode ISO, saat ini hanya mendukung `"IDR"`.
  - `initial_deposit`: Minimal deposit awal adalah Rp100.000 (`100000`).
- **Contoh Request:**
  ```bash
  curl -i -X POST http://localhost:8080/accounts \
    -H "Content-Type: application/json" \
    -d '{
      "owner_name": "Budi Santoso",
      "currency": "IDR",
      "initial_deposit": 500000
    }'
  ```
- **Response Sukses (`201 Created`):**
  ```json
  {
    "id": "e4b2d19f-4375-4c07-82ba-b258dc76d299",
    "account_number": "8801839201",
    "owner_name": "Budi Santoso",
    "currency": "IDR",
    "balance": 500000,
    "created_at": "2026-11-23T08:30:00Z",
    "updated_at": null
  }
  ```
- **Response Error Validasi (`400 Bad Request`):**
  ```json
  {
    "error": {
      "status": 400,
      "code": "VALIDATION_ERROR",
      "message": "Minimum Deposit Must Be Rp100.000"
    }
  }
  ```

---

### 3. Ambil Detail Akun (`Get Account by Number`)

Mengambil data akun berdasarkan nomor akun (`account_number`).

- **URL:** `/accounts/{account_number}`
- **Method:** `GET`
- **Contoh Request:**
  ```bash
  curl -i http://localhost:8080/accounts/8801839201
  ```
- **Response Sukses (`200 OK`):**
  ```json
  {
    "id": "e4b2d19f-4375-4c07-82ba-b258dc76d299",
    "account_number": "8801839201",
    "owner_name": "Budi Santoso",
    "currency": "IDR",
    "balance": 500000,
    "created_at": "2026-11-23T08:30:00Z",
    "updated_at": null
  }
  ```
- **Response Error (`404 Not Found`):**
  ```json
  {
    "error": {
      "status": 404,
      "code": "ACCOUNT_NOT_FOUND",
      "message": "failed to search account: account not found"
    }
  }
  ```

---

### 4. Daftar Seluruh Akun (`List Accounts`)

Mengambil seluruh akun yang terdaftar, diurutkan berdasarkan waktu pembuatan.

- **URL:** `/accounts`
- **Method:** `GET`
- **Contoh Request:**
  ```bash
  curl -i http://localhost:8080/accounts
  ```
- **Response Sukses (`200 OK`):**
  ```json
  [
    {
      "id": "e4b2d19f-4375-4c07-82ba-b258dc76d299",
      "account_number": "8801839201",
      "owner_name": "Budi Santoso",
      "currency": "IDR",
      "balance": 500000,
      "created_at": "2026-11-23T08:30:00Z",
      "updated_at": null
    },
    {
      "id": "c1a93e50-2051-40ff-9076-2679958bc912",
      "account_number": "8801452910",
      "owner_name": "Siti Rahma",
      "currency": "IDR",
      "balance": 1500000,
      "created_at": "2026-11-23T09:00:00Z",
      "updated_at": null
    }
  ]
  ```

---

## 📚 Dokumentasi Arsitektur
Arsitektur sistem ini menggunakan pendekatan **Modular Monolith**. Detail keputusan arsitektur dapat dilihat pada dokumen ADR:
- [ADR 001: Modular Monolith Architecture](docs/adr/001-modular-monolith.md)
