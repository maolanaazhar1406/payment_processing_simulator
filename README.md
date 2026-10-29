# Payment Processing Simulator

Simulator edukatif berbasis Go untuk alur authorize, capture, void, dan refund. Aplikasi hanya memakai nomor kartu uji; tidak terhubung ke payment network dan tidak menyimpan PAN atau CVV.

Lihat [APPLICATION-FLOW.md](./APPLICATION-FLOW.md) untuk penjelasan alur request dan komponen aplikasi. Panduan pengujian Postman tersedia di [POSTMAN-FLOW.md](./POSTMAN-FLOW.md).

## Menjalankan

Gunakan Go 1.22 atau yang lebih baru:

```sh
make run
```

Server berjalan di `http://localhost:8080`. Konfigurasi:

| Environment variable | Default | Keterangan |
|---|---:|---|
| `PORT` | `8080` | Port HTTP |
| `SIM_LATENCY_MS` | `50` | Latensi issuer simulasi |
| `IDEMPOTENCY_TTL_HOURS` | `24` | Masa berlaku key idempotensi |
| `MERCHANTS` | `merchant_demo` | Merchant yang diizinkan, dipisahkan koma |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, atau `error` |

Semua operasi `POST` memerlukan `Idempotency-Key` dan `X-Merchant-ID`. Endpoint transaksi hanya tersedia untuk merchant yang tercantum pada `MERCHANTS`.

## Alur contoh

```sh
curl -i http://localhost:8080/v1/transactions/authorize \
  -H 'Content-Type: application/json' \
  -H 'X-Merchant-ID: merchant_demo' \
  -H 'Idempotency-Key: auth-demo-1' \
  -d '{"amount":150000,"currency":"IDR","card":{"number":"4111111111111111","exp_month":12,"exp_year":2030,"cvv":"123"}}'
```

Gunakan `id` dari respons authorize pada endpoint capture, refund, void, atau GET `/v1/transactions/{id}`. Capture dan refund menerima body `{"amount":100000,"currency":"IDR"}`; void tidak memerlukan body.

## Postman

Import file [`payment-simulator.postman_collection.json`](./payment-simulator.postman_collection.json) ke Postman. Jalankan server dengan `make run`, lalu pilih **Payment Processing Simulator** dan jalankan request sesuai urutan (atau gunakan Collection Runner). Koleksi otomatis menyimpan ID transaksi dari respons authorize, menguji replay dan penggunaan ulang idempotency key, lalu menjalankan capture/refund sebagian, void, dan GET transaksi.

Nilai `baseUrl` dan `merchantId` tersedia sebagai collection variables dan dapat diubah dari tab **Variables**. Koleksi menggunakan kartu uji saja.

Lihat [POSTMAN-FLOW.md](./POSTMAN-FLOW.md) untuk urutan lengkap, status yang diharapkan, panduan jika mengganti nomor kartu memunculkan `IDEMPOTENCY_KEY_REUSED`, dan cara mengganti semua idempotency key saat memulai siklus test baru.

Contoh kartu uji Luhn-valid. Tanggal kedaluwarsa harus berada di masa depan:

| Nomor uji | Hasil |
|---|---|
| `4111111111111111` | Approved (`00`) |
| `4111111111120005` | Declined (`05`) |
| `4111111111150051` | Dana tidak cukup (`51`) |
| `4111111111120054` | Kedaluwarsa (`54`) |
| `4111111111170091` | Issuer tidak tersedia (`91`); transaksi DECLINED, ulangi dengan transaksi/key baru |

PAN dan CVV hanya digunakan sementara saat validasi/simulasi. Respons transaksi hanya memuat brand, empat digit terakhir, dan masa berlaku.

## Pengembangan

```sh
make build
make test
make lint
go test ./... -race
```

`make lint` menjalankan `go vet`; konfigurasi `.golangci.yml` disediakan untuk `golangci-lint` v2 bila tersedia.
