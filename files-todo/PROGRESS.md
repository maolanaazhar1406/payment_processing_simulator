# PROGRESS.md

Catatan status harian untuk **Payment Processing Simulator**. Tambahkan entri baru di bagian paling atas setiap sesi. Jangan menghapus entri lama.

**Format entri:**
- **Tanggal dan sesi**
- **Tugas dikerjakan** (ID dari `TASK.md`)
- **Selesai**
- **Tertunda / belum selesai**
- **Blocker dan pertanyaan terbuka**
- **Keputusan baru** (jika ada, rujuk ID di `CONTEXT.md`)
- **Langkah berikutnya**

---

## 2026-10-29 · Sesi 4 (Dokumentasi Flow Aplikasi)

**Tugas dikerjakan:** T-032

**Selesai:**
- Membuat `APPLICATION-FLOW.md` untuk menjelaskan startup, arsitektur, middleware, idempotensi, authorize, capture, void, refund, GET transaksi, error, serta batas penyimpanan in-memory.
- Menambahkan diagram Mermaid untuk hubungan komponen, sequence request, dan state transaksi.
- Memperjelas perbedaan flow aplikasi dengan panduan flow Postman dan menautkan keduanya dari README.

**Tertunda / belum selesai:**
- Tidak ada.

**Blocker dan pertanyaan terbuka:**
- Tidak ada.

**Keputusan baru:** Tidak ada.

**Langkah berikutnya:**
1. Baca `APPLICATION-FLOW.md` untuk memahami proses aplikasi dari request masuk sampai respons.

---

## 2026-10-29 · Sesi 3 (Dokumentasi Flow Postman)

**Tugas dikerjakan:** T-031

**Selesai:**
- Menulis `POSTMAN-FLOW.md` berisi persiapan, urutan request, flow status transaksi, kartu uji, dan troubleshooting `IDEMPOTENCY_KEY_REUSED`.
- Menjelaskan cara membuat authorize baru dengan idempotency key unik ketika PAN pada body diubah, termasuk request retry yang harus tetap memakai body/key sama.
- Menautkan dokumen flow dari README.

**Tertunda / belum selesai:**
- Tidak ada.

**Blocker dan pertanyaan terbuka:**
- Tidak ada.

**Keputusan baru:** Tidak ada.

**Langkah berikutnya:**
1. Ikuti langkah di `POSTMAN-FLOW.md` untuk mencoba kartu uji lain.

---

## 2026-10-29 · Sesi 2 (Koleksi Postman)

**Tugas dikerjakan:** T-030

**Selesai:**
- Menambahkan koleksi Postman yang dapat diimpor untuk health check, authorize, idempotency replay/mismatch, capture/refund parsial, void, dan lookup transaksi.
- Koleksi menyimpan ID transaksi secara otomatis dan memeriksa respons melalui Post-response tests.
- Menambahkan instruksi import dan cara menjalankan di README.
- Memvalidasi JSON koleksi dan menjalankan seluruh 10 request berurutan terhadap server lokal; semua respons serta status transaksi sesuai.

**Tertunda / belum selesai:**
- Tidak ada.

**Blocker dan pertanyaan terbuka:**
- Newman tidak terpasang, sehingga Postman test scripts belum dijalankan di Newman; request koleksinya sendiri berhasil diverifikasi lewat HTTP.

**Keputusan baru:** Tidak ada.

**Langkah berikutnya:**
1. Import `payment-simulator.postman_collection.json` dan jalankan koleksi setelah server hidup.

---

## 2026-10-29 · Sesi 1 (Implementasi Simulator)

**Tugas dikerjakan:** T-001 hingga T-029

**Selesai:**
- Membuat modul Go 1.22+, struktur paket, Makefile, domain money/card/state machine, transaction store, idempotency, simulator issuer, dan service authorize/capture/void/refund.
- Menyediakan HTTP API, autentikasi merchant, middleware idempotensi dan logging aman, loader konfigurasi, entrypoint dengan graceful shutdown, README, dan koleksi `.http`.
- Menambah unit dan integration tests termasuk replay idempotensi, alur transaksi, ownership, serta verifikasi PAN/CVV tidak muncul di log.
- Cakupan `domain`: 90.5%; `service`: 86.6%.
- `make build`, `make test`, `make lint`, dan `go test ./... -race` berhasil.

**Tertunda / belum selesai:**
- Tidak ada.

**Blocker dan pertanyaan terbuka:**
- `golangci-lint` tidak terpasang di lingkungan; konfigurasi v2 ditambahkan, sedangkan `make lint` menjalankan `go vet`.

**Keputusan baru:** D-008 (IDR/USD dan perlakuan kode 91), D-009 (UUID), D-010 (interface pada sisi consumer).

**Langkah berikutnya:**
1. Jalankan server dengan `make run` dan gunakan contoh request di README atau `payment-simulator.http`.

---

## 2026-10-29 · Sesi 0 (Setup Dokumentasi)

**Tugas dikerjakan:** Penyusunan dokumen fondasi (`AGENTS.md`, `CONTEXT.md`, `TASK.md`, `PROGRESS.md`).

**Selesai:**
- Menyusun `AGENTS.md` berisi aturan proyek, struktur direktori, aturan Go, dan definisi selesai.
- Menyusun `CONTEXT.md` berisi arsitektur, model domain, state machine transaksi, kontrak API, aturan idempotensi, dan log keputusan D-001 hingga D-007.
- Menyusun `TASK.md` dengan 29 tugas dalam 6 fase.

**Tertunda / belum selesai:**
- Belum ada kode. Modul Go belum diinisialisasi (T-001).

**Blocker dan pertanyaan terbuka:**
- Apakah simulator perlu mendukung mata uang selain IDR dan USD pada fase awal?
- Apakah kode respons `91` (timeout) harus mengubah status transaksi menjadi `DECLINED` atau tetap pada status awal sehingga bisa dicoba ulang?
- Versi Go target: diasumsikan 1.22+. Konfirmasi jika berbeda.

**Keputusan baru:** Tidak ada keputusan baru di luar D-001 hingga D-007.

**Langkah berikutnya:**
1. Kerjakan T-001 (inisialisasi modul dan struktur direktori).
2. Lanjut T-002 dan T-003.
3. Putuskan pertanyaan terbuka tentang kode `91` sebelum mengerjakan T-010.

---

## Template Entri Baru

```
## YYYY-MM-DD · Sesi N

**Tugas dikerjakan:** T-XXX

**Selesai:**
-

**Tertunda / belum selesai:**
-

**Blocker dan pertanyaan terbuka:**
-

**Keputusan baru:** (ID di CONTEXT.md, atau "tidak ada")

**Langkah berikutnya:**
1.
```
