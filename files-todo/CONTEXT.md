# CONTEXT.md

Dokumen arsitektur dan keputusan desain untuk **Payment Processing Simulator**. Dokumen ini berubah jarang. Perubahan keputusan dicatat di bagian **Log Keputusan** di bagian bawah, jangan mengedit keputusan lama secara diam-diam.

---

## 1. Ringkasan Arsitektur

```
 Client (merchant simulasi)
        │  HTTP/JSON
        ▼
 ┌──────────────────────┐
 │   API Layer (api)    │  router, validasi request, middleware
 └──────────┬───────────┘
            ▼
 ┌──────────────────────┐
 │  Service (service)   │  use case: authorize, capture, void, refund
 └──┬───────────────┬───┘
    ▼               ▼
 ┌────────┐   ┌──────────────────┐
 │ Store  │   │ Simulator        │  issuer/acquirer simulasi
 │        │   │ (approve/decline │
 └────────┘   │  /timeout)       │
    ▲         └──────────────────┘
    │
 ┌──────────────────────┐
 │ Idempotency          │  penyimpanan hasil request per key
 └──────────────────────┘
```

Alur: request masuk → middleware (request ID, logging dengan masking) → cek `Idempotency-Key` → validasi → service → simulator (keputusan issuer) → update state di store → simpan respons idempotensi → kembalikan respons.

---

## 2. Stack Teknologi

| Area | Pilihan | Alasan |
|---|---|---|
| Bahasa | Go 1.22+ | Target proyek, konkurensi bawaan |
| HTTP | `net/http` standar (Go 1.22 `ServeMux` dengan pola metode) | Minim dependensi |
| Penyimpanan | In-memory dengan `sync.RWMutex` pada fase awal; interface `store` agar bisa diganti | Cepat untuk simulasi, tidak butuh setup |
| Serialisasi | `encoding/json` | Standar |
| Logging | `log/slog` | Standar, structured logging |
| ID | UUIDv4 via `github.com/google/uuid` | Dependensi kecil dan stabil |
| Testing | `testing`, `httptest`, `-race` | Standar |

Dependensi tambahan harus dicatat di Log Keputusan.

---

## 3. Model Domain

### 3.1 Entitas Utama

**Transaction**
- `ID` (string, UUID)
- `Type` (`AUTHORIZATION`)
- `Status` (lihat state machine)
- `Amount` (`Money`)
- `Currency` (ISO 4217, contoh `IDR`, `USD`)
- `CapturedAmount`, `RefundedAmount` (`Money`)
- `Card` (`CardInfo`): hanya menyimpan `Brand`, `Last4`, `ExpMonth`, `ExpYear`. **Tidak menyimpan PAN penuh atau CVV.**
- `MerchantID` (string)
- `ResponseCode` (string, mengikuti format kode respons simulasi)
- `CreatedAt`, `UpdatedAt` (`time.Time`, UTC)
- `Version` (int, untuk optimistic locking)

**Money**
- `Amount` (`int64`, dalam unit terkecil)
- `Currency` (string)
- Operasi: `Add`, `Sub`, `Compare`. Operasi antar mata uang berbeda harus ditolak.
- Fase awal hanya menerima `IDR` dan `USD`.

**IdempotencyRecord**
- `Key` (string)
- `MerchantID` (string)
- `RequestHash` (SHA-256 dari body request)
- `ResponseStatus` (int)
- `ResponseBody` ([]byte)
- `CreatedAt` (`time.Time`)
- `ExpiresAt` (`time.Time`, default 24 jam)

### 3.2 Kode Respons Simulasi

| Kode | Arti | Terminal? |
|---|---|---|
| `00` | Approved | Tidak |
| `05` | Declined oleh issuer | Ya (untuk request tersebut) |
| `51` | Dana tidak cukup (simulasi limit) | Ya |
| `54` | Kartu kedaluwarsa | Ya |
| `91` | Issuer tidak tersedia (timeout simulasi) | Ya; retry memakai transaksi dan key baru |
| `14` | Nomor kartu tidak valid | Ya |

Kode ini adalah model internal yang disederhanakan, bukan standar ISO 8583 lengkap.

---

## 4. State Machine Transaksi

```
              authorize (approved)
   [ ] ─────────────────────────────▶ AUTHORIZED
                                         │   │   │
                             capture     │   │   │ void
                  ┌──────────────────────┘   │   └──────────────┐
                  ▼                          │                  ▼
              CAPTURED ─── refund ──▶ PARTIALLY_REFUNDED     VOIDED
                  │                          │
                  │ refund penuh             │ refund sisa
                  ▼                          ▼
              REFUNDED ◀──────────────────────┘

   [ ] ─────────────────────────────▶ DECLINED   (terminal)
```

### 4.1 Aturan Transisi

| Dari | Operasi | Ke | Syarat |
|---|---|---|---|
| (awal) | authorize | AUTHORIZED | kode respons `00` |
| (awal) | authorize | DECLINED | kode respons selain `00` |
| AUTHORIZED | capture | CAPTURED | `captureAmount` ≤ `Amount` |
| AUTHORIZED | void | VOIDED | belum pernah di-capture |
| CAPTURED | refund | PARTIALLY_REFUNDED | `refund` < `CapturedAmount - RefundedAmount` |
| CAPTURED | refund | REFUNDED | refund = sisa yang belum direfund |
| PARTIALLY_REFUNDED | refund | PARTIALLY_REFUNDED / REFUNDED | refund ≤ sisa |

Operasi lain pada status yang tidak sesuai harus mengembalikan error `ErrInvalidTransition` dengan kode HTTP `409 Conflict`.

---

## 5. Kontrak API (Ringkas)

Semua endpoint berada di bawah `/v1`. Respons menggunakan JSON. Setiap respons error memiliki format:

```json
{ "error": { "code": "INVALID_TRANSITION", "message": "..." } }
```

| Method | Path | Deskripsi | Header wajib |
|---|---|---|---|
| POST | `/v1/transactions/authorize` | Otorisasi transaksi baru | `Idempotency-Key`, `X-Merchant-ID` |
| POST | `/v1/transactions/{id}/capture` | Capture sebagian atau penuh | `Idempotency-Key`, `X-Merchant-ID` |
| POST | `/v1/transactions/{id}/void` | Batalkan otorisasi | `Idempotency-Key`, `X-Merchant-ID` |
| POST | `/v1/transactions/{id}/refund` | Refund sebagian atau penuh | `Idempotency-Key`, `X-Merchant-ID` |
| GET | `/v1/transactions/{id}` | Ambil detail transaksi | `X-Merchant-ID` |
| GET | `/healthz` | Health check | - |

Contoh body authorize:

```json
{
  "amount": 150000,
  "currency": "IDR",
  "card": { "number": "4111111111111111", "exp_month": 12, "exp_year": 2030, "cvv": "123" }
}
```

Catatan: `number` dan `cvv` hanya dipakai di memori untuk validasi dan simulasi, lalu dibuang. Tidak boleh masuk ke log atau disimpan.

### 5.1 Aturan Idempotensi

1. Key wajib untuk operasi POST yang mengubah state.
2. Jika key sudah ada dengan `RequestHash` yang sama: kembalikan respons tersimpan.
3. Jika key sudah ada tetapi `RequestHash` berbeda: kembalikan `422 Unprocessable Entity` dengan kode `IDEMPOTENCY_KEY_REUSED`.
4. Jika request pertama masih diproses (belum ada respons): kembalikan `409` dengan kode `REQUEST_IN_PROGRESS`.
5. Key berlaku per merchant dan kedaluwarsa setelah 24 jam.

---

## 6. Simulator Issuer

Keputusan issuer ditentukan oleh **nomor kartu uji** agar hasil bisa diprediksi:

| Pola nomor kartu (4 digit terakhir) | Hasil |
|---|---|
| `0000` | Approved (`00`) |
| `0005` | Declined (`05`) |
| `0051` | Dana tidak cukup (`51`) |
| `0054` | Kartu kedaluwarsa (`54`) |
| `0091` | Timeout (`91`) |
| lainnya | Approved (`00`) |

Simulator dapat menambahkan latensi acak (konfigurasi via env, default 50 ms) agar perilaku menyerupai jaringan nyata.

Validasi kartu uji: nomor harus lolos algoritma Luhn. Nomor yang tidak valid menghasilkan kode `14`.

---

## 7. Keamanan dan Kepatuhan Simulasi

- **Masking**: Fungsi `MaskPAN(pan string) string` mengembalikan `************1111`. Dipakai di setiap log dan respons.
- **Tidak ada persistensi data kartu sensitif**: Hanya `Last4`, brand, dan masa berlaku yang disimpan.
- **Logging aman**: Middleware logging tidak boleh mencetak body request untuk endpoint `/authorize`.
- **Autentikasi merchant (sederhana)**: Header `X-Merchant-ID` dicocokkan dengan daftar merchant di konfigurasi. Ini bukan autentikasi produksi.
- **TLS**: Tidak diwajibkan pada fase lokal, tetapi konfigurasi server harus siap menerima `tls.Config`.

---

## 8. Konfigurasi

Semua konfigurasi melalui environment variable:

| Variabel | Default | Keterangan |
|---|---|---|
| `PORT` | `8080` | Port HTTP |
| `SIM_LATENCY_MS` | `50` | Latensi simulasi issuer |
| `IDEMPOTENCY_TTL_HOURS` | `24` | Masa berlaku key idempotensi |
| `MERCHANTS` | `merchant_demo` | Daftar merchant, dipisah koma |
| `LOG_LEVEL` | `info` | Level `slog` |

---

## 9. Strategi Error

- Error domain didefinisikan sebagai variabel sentinel di paket `domain`, misalnya `ErrInvalidTransition`, `ErrNotFound`, `ErrCurrencyMismatch`.
- Layer `api` memetakan error domain ke kode HTTP dan kode error string melalui satu fungsi terpusat.
- Error tak terduga dicatat dengan `slog` dan dikembalikan sebagai `500 INTERNAL_ERROR` tanpa detail internal.

---

## 10. Konkurensi

- Store in-memory memakai `sync.RWMutex` per store.
- Setiap update transaksi menggunakan `Version` untuk optimistic locking. Konflik menghasilkan `409 CONFLICT`.
- Tidak ada goroutine background pada fase awal. Pembersihan idempotency record yang kedaluwarsa dilakukan secara lazy saat lookup.

---

## 11. Rencana Pengembangan Jangka Panjang (Belum Diputuskan)

- Ganti store ke PostgreSQL dengan migrasi terstruktur.
- Tambahkan webhook untuk notifikasi status transaksi.
- Tambahkan endpoint rekonsiliasi harian (batch file CSV).
- Tambahkan metrik Prometheus.

Item di atas hanya dicatat sebagai arah. Implementasi dimulai setelah keputusan resmi dicatat di Log Keputusan.

---

## 12. Log Keputusan

| Tanggal | ID | Keputusan | Alasan |
|---|---|---|---|
| 2026-10-29 | D-001 | Gunakan `net/http` standar tanpa framework | Minim dependensi, cukup untuk skala simulasi |
| 2026-10-29 | D-002 | `Money` disimpan sebagai `int64` unit terkecil | Menghindari kesalahan pembulatan `float64` |
| 2026-10-29 | D-003 | Store awal in-memory dengan interface | Cepat dikembangkan, mudah diganti nanti |
| 2026-10-29 | D-004 | Keputusan issuer ditentukan dari 4 digit terakhir PAN | Hasil uji dapat diprediksi dan terdokumentasi |
| 2026-10-29 | D-005 | Idempotensi wajib untuk semua operasi mengubah state | Mencegah efek ganda akibat retry |
| 2026-10-29 | D-006 | PAN dan CVV tidak pernah disimpan atau dilog | Prinsip keamanan data kartu, bahkan dalam simulasi |
| 2026-10-29 | D-007 | Optimistic locking via field `Version` | Konkurensi sederhana tanpa lock jangka panjang |
| 2026-10-29 | D-008 | Fase awal menerima IDR dan USD; respons issuer `91` menghasilkan transaksi `DECLINED`, dan retry harus memakai transaksi serta key baru | Menetapkan pertanyaan terbuka dari sesi setup tanpa mengubah state transaksi yang sudah tersimpan |
| 2026-10-29 | D-009 | Gunakan `github.com/google/uuid` untuk ID transaksi | Sudah dipilih pada stack awal; dependensi kecil dan stabil |
| 2026-10-29 | D-010 | Interface storage didefinisikan pada consumer (`service` dan `api`); paket `store` hanya menyediakan implementasi in-memory | Mengikuti aturan interface di sisi konsumen pada `AGENTS.md` dan menghindari kontrak duplikat yang tidak dipakai |
