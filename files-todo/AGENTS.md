# AGENTS.md

Panduan permanen untuk agen (AI maupun manusia) yang bekerja di repositori **Payment Processing Simulator**.

## 1. Tujuan Proyek

Membangun simulator pemrosesan pembayaran berbasis Go yang meniru alur kerja payment gateway nyata: menerima permintaan pembayaran, memvalidasi, meneruskan ke "acquirer/issuer" simulasi, menangani otorisasi, capture, refund, void, dan rekonsiliasi, serta mencatat seluruh transaksi secara konsisten.

Proyek ini bersifat **edukatif dan simulasi**. Tidak boleh terhubung ke jaringan pembayaran nyata, tidak boleh memproses data kartu asli, dan tidak boleh menyimpan data kartu sensitif (PAN, CVV) dalam bentuk apa pun.

## 2. Prinsip Utama

1. **Keamanan data kartu**: Gunakan nomor kartu uji saja (misalnya dari daftar test PAN standar). Jangan pernah log PAN atau CVV secara lengkap. Masking wajib: tampilkan hanya 4 digit terakhir.
2. **Idempotensi**: Setiap operasi yang mengubah state wajib mendukung header `Idempotency-Key`. Permintaan ulang dengan key yang sama harus mengembalikan hasil yang sama tanpa efek samping ganda.
3. **Konsistensi state**: Transisi status transaksi harus mengikuti state machine yang didefinisikan di `CONTEXT.md`. Transisi ilegal harus ditolak dengan error yang jelas.
4. **Sederhana dulu**: Hindari abstraksi berlebihan. Tambahkan kompleksitas hanya jika ada kebutuhan nyata yang tercatat di `TASK.md`.
5. **Testable by default**: Logika bisnis dipisahkan dari HTTP handler dan storage agar bisa diuji unit tanpa server.

## 3. Struktur Direktori (Acuan)

```
payment-simulator/
├── cmd/
│   └── server/           # entrypoint aplikasi
├── internal/
│   ├── api/              # HTTP handler, router, middleware
│   ├── domain/           # entitas dan aturan bisnis (Transaction, Money, dll.)
│   ├── service/          # use case: authorize, capture, refund, void
│   ├── store/            # interface dan implementasi penyimpanan
│   ├── simulator/        # simulasi issuer/acquirer (approve, decline, timeout)
│   └── idempotency/      # logika penyimpanan dan pencocokan idempotency key
├── pkg/                  # kode yang boleh diekspor (jika ada)
├── test/                 # integration test dan fixture
├── AGENTS.md
├── CONTEXT.md
├── PROGRESS.md
├── TASK.md
├── go.mod
└── README.md
```

Jangan menambahkan folder baru di root tanpa mencatat alasannya di `CONTEXT.md`.

## 4. Aturan Kode Go

- Ikuti `gofmt` dan `go vet` sebelum commit. Gunakan `golangci-lint` bila tersedia.
- Gunakan `context.Context` sebagai parameter pertama untuk fungsi yang melakukan I/O.
- Error wrapping wajib memakai `fmt.Errorf("...: %w", err)`. Gunakan `errors.Is` dan `errors.As` untuk pengecekan.
- Jangan gunakan `panic` untuk alur normal. `panic` hanya boleh di inisialisasi yang gagal fatal.
- Nominal uang **wajib** disimpan sebagai integer dalam unit terkecil (misalnya sen atau cent) dengan tipe `Money`, bukan `float64`.
- Nama paket pendek, huruf kecil, tanpa underscore.
- Interface didefinisikan di sisi konsumen, bukan di sisi implementasi.
- Hindari variabel global kecuali untuk konstanta.
- Setiap fungsi publik memiliki komentar dokumentasi yang dimulai dengan nama fungsinya.

## 5. Aturan Testing

- Setiap perubahan logika bisnis wajib disertai unit test.
- Gunakan table-driven test untuk skenario state machine dan validasi.
- Integration test berada di `test/` dan menjalankan server pada port acak.
- Target awal: cakupan test untuk `domain` dan `service` minimal 80%.
- Jalankan `go test ./... -race` sebelum menandai tugas selesai.

## 6. Aturan Kerja Agen

Setiap sesi kerja, agen wajib:

1. **Membaca dulu** `AGENTS.md`, `CONTEXT.md`, `TASK.md`, dan `PROGRESS.md` sebelum mengubah kode.
2. **Memilih satu tugas** dari `TASK.md` dan menandainya sebagai `IN PROGRESS`. Hindari mengerjakan banyak tugas sekaligus.
3. **Memperbarui `PROGRESS.md`** di akhir sesi dengan ringkasan apa yang selesai, apa yang tertunda, dan blocker yang ditemui.
4. **Memindahkan tugas** yang selesai ke status `DONE` di `TASK.md`, lengkap dengan tanggal.
5. **Tidak mengubah arsitektur** tanpa mencatat keputusan baru di `CONTEXT.md` (bagian "Log Keputusan").
6. **Tidak menghapus test** yang gagal. Perbaiki kodenya, atau jika requirement berubah, catat perubahan itu di `CONTEXT.md`.
7. **Meminta klarifikasi** bila requirement ambigu, bukan menebak untuk hal yang berdampak pada state transaksi atau nominal uang.

## 7. Larangan

- Menambahkan dependensi baru tanpa alasan tertulis di `CONTEXT.md`.
- Menyimpan secret, API key, atau kredensial di repositori. Gunakan environment variable.
- Melakukan commit langsung ke branch `main` untuk perubahan fitur.
- Menonaktifkan validasi, idempotency check, atau masking untuk mempercepat pengujian.
- Menggunakan `float64` untuk nominal uang.
- Mengirim request ke layanan eksternal nyata dari kode simulator.

## 8. Format Commit

Gunakan Conventional Commits:

```
feat(service): tambah operasi refund parsial
fix(domain): perbaiki validasi transisi void setelah capture
test(api): tambah test idempotency untuk authorize
docs(context): catat keputusan penggunaan integer untuk Money
```

## 9. Definisi Selesai (Definition of Done)

Sebuah tugas dianggap selesai jika:

- [ ] Kode memenuhi aturan di file ini dan `CONTEXT.md`.
- [ ] `gofmt`, `go vet`, dan `go test ./... -race` lolos.
- [ ] Unit test untuk logika baru sudah ditulis.
- [ ] Tidak ada log yang membocorkan data kartu.
- [ ] `TASK.md` dan `PROGRESS.md` sudah diperbarui.
