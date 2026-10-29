# TASK.md

To-do list aktif untuk **Payment Processing Simulator**. File ini diperbarui setiap sesi kerja.

**Aturan pemakaian** (lihat juga `AGENTS.md` bagian 6):
- Hanya satu tugas boleh berstatus `IN PROGRESS` pada satu waktu.
- Tugas yang selesai dipindahkan ke bagian **DONE** beserta tanggal.
- Tugas baru ditambahkan ke **BACKLOG** dengan prioritas (P0 = kritis, P1 = penting, P2 = nanti).
- Tugas yang menyentuh arsitektur wajib merujuk ID keputusan di `CONTEXT.md`.

---

## IN PROGRESS

_(kosong)_

---

## BACKLOG

_(kosong)_

---

## DONE

### Fase 1: Fondasi

- [x] **T-001** Inisialisasi modul Go dan struktur direktori — selesai 2026-10-29.
- [x] **T-002** Tambahkan `Makefile` dengan target `build`, `test`, `lint`, `run` — selesai 2026-10-29.
- [x] **T-003** Implementasi `domain.Money` dan table-driven tests — selesai 2026-10-29.
- [x] **T-004** Implementasi `MaskPAN`, validasi Luhn, dan tests — selesai 2026-10-29.
- [x] **T-005** Definisikan entitas transaksi/kartu dan error sentinel — selesai 2026-10-29.
- [x] **T-006** Implementasi state machine dan test tabel transisi — selesai 2026-10-29.

### Fase 2: Penyimpanan dan Simulator

- [x] **T-007** Port transaksi di sisi consumer, in-memory store, mutex, dan optimistic locking (D-010) — selesai 2026-10-29.
- [x] **T-008** Simulator issuer deterministik dengan latensi configurable — selesai 2026-10-29.
- [x] **T-009** Penyimpanan idempotensi, hash, TTL, dan lazy expiration — selesai 2026-10-29.

### Fase 3: Service

- [x] **T-010** Service `Authorize` dan validasi kartu/transaksi — selesai 2026-10-29.
- [x] **T-011** Service `Capture` penuh/sebagian dan batas jumlah — selesai 2026-10-29.
- [x] **T-012** Service `Void` dengan validasi state — selesai 2026-10-29.
- [x] **T-013** Service `Refund` penuh/sebagian dan akumulasi — selesai 2026-10-29.
- [x] **T-014** Service `GetTransaction` dengan pengecekan merchant — selesai 2026-10-29.

### Fase 4: HTTP API

- [x] **T-015** Router `net/http` Go 1.22 dan `/healthz` — selesai 2026-10-29.
- [x] **T-016** Endpoint authorize dan validasi header — selesai 2026-10-29.
- [x] **T-017** Endpoint capture, void, refund, dan pemetaan error — selesai 2026-10-29.
- [x] **T-018** Endpoint GET transaksi — selesai 2026-10-29.
- [x] **T-019** Middleware idempotensi dengan replay dan pemeriksaan hash — selesai 2026-10-29.
- [x] **T-020** Logging tanpa body sensitif; diuji agar PAN/CVV tidak tercetak — selesai 2026-10-29.
- [x] **T-021** Autentikasi merchant melalui `MERCHANTS` — selesai 2026-10-29.

### Fase 5: Konfigurasi, Entrypoint, dan Pengujian

- [x] **T-022** Loader konfigurasi environment dengan default dan validasi — selesai 2026-10-29.
- [x] **T-023** Entrypoint server dengan graceful shutdown — selesai 2026-10-29.
- [x] **T-024** Integration test authorize → capture → refund dan authorize → void — selesai 2026-10-29.
- [x] **T-025** Integration test replay idempotensi dan body berbeda — selesai 2026-10-29.
- [x] **T-026** Cakupan domain 90.5% dan service 86.6% — selesai 2026-10-29.

### Fase 6: Dokumentasi dan Rapi-rapi

- [x] **T-032** Dokumentasi flow aplikasi HTTP, service, simulator, dan store — selesai 2026-10-29.
- [x] **T-031** Dokumentasi flow Postman, state transaksi, dan troubleshooting idempotensi — selesai 2026-10-29.
- [x] **T-027** README cara menjalankan, contoh curl, dan nomor kartu uji — selesai 2026-10-29.
- [x] **T-028** Contoh request VS Code REST Client — selesai 2026-10-29.
- [x] **T-029** Evaluasi `golangci-lint` dan konfigurasi v2 — selesai 2026-10-29.
- [x] **T-030** Koleksi Postman dengan collection variables dan tes alur transaksi; tervalidasi pada server lokal — selesai 2026-10-29.

---

## Catatan Prioritas Saat Ini

Semua tugas T-001 hingga T-032 selesai. `golangci-lint` tidak tersedia di lingkungan ini; konfigurasi disertakan dan `make lint` memverifikasi dengan `go vet`.
