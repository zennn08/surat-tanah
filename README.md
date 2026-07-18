# SITANAH — Aplikasi Surat Tanah Garapan

Aplikasi desktop lokal untuk **Kantor Camat Dumai Timur**: petugas mengisi data
sebidang tanah garapan **satu kali**, aplikasi menghasilkan **3 surat siap cetak**
sesuai blangko resmi:

1. **Surat Pernyataan Tidak Bersengketa**
2. **Berita Acara Pengukuran Tanah**
3. **Sceets Kaart (Peta Situasi Tanah)**

Fitur inti: **deteksi surat ganda** — setiap lokasi + batas-batas tanah yang
diisi otomatis dibandingkan dengan seluruh berkas lama (merah/kuning/abu-abu),
karena surat ganda atas bidang yang sama berujung sengketa.

![Form berkas dengan denah batas dan deteksi duplikat](docs/img/10-langkah2-lokasi-batas.png)

## Unduh & jalankan

1. Unduh `surat-tanah.exe` dari [halaman Releases](https://github.com/zennn08/surat-tanah/releases).
2. Letakkan di satu folder, **klik dua kali** — selesai. Tanpa instalasi,
   tanpa internet; browser terbuka otomatis ke `http://127.0.0.1:8080`.
3. Login awal `admin` / `admin123` (wajib diganti saat login pertama).

Seluruh data tersimpan di **satu file** `surat-tanah.db` di samping exe —
backup cukup menyalin file itu.

📖 **[Panduan penggunaan lengkap dengan tangkapan layar →](docs/panduan-penggunaan.md)**

## Fitur

- **Deteksi duplikat 3 tingkat** — nama batas dinormalisasi (`H. Ahmad Yani` ≈
  `HAJI AHMAD YANI`, `Jl.` ≈ `Jalan`, `RT 007` ≈ `7`); duplikat persis hanya
  bisa disimpan dengan alasan tertulis yang teraudit (tanda ⚠ di daftar).
- **Pecah bidang** — tandai berkas induk saat warga menjual sebagian tanah;
  aplikasi menghitung sisa luas dan memperingatkan bila pecahan melebihi induk.
- **Arsip surat lama** — surat yang terbit sebelum aplikasi ada bisa diinput
  sebagai arsip (bertanda 🗄, nomor mengikuti tahun surat aslinya) agar ikut
  terjaring deteksi duplikat.
- **Form wizard 5 langkah** dengan denah batas mata angin dan langkah
  "Periksa & Simpan" — dirancang untuk petugas non-teknis.
- **Cetak A4 sesuai blangko resmi** — teks dipertahankan kata demi kata
  (termasuk ejaan asli "Sceets Kaart"), tiap surat pas satu halaman, kotak
  peta kosong untuk digambar tangan.
- **Master data** kelurahan, lurah (satu aktif per kelurahan), ketua RT, dan
  juru ukur; pencarian berkas termasuk **lewat nama pemilik tanah sebelah**.
- **Lokal sepenuhnya** — bind `127.0.0.1` saja, tanpa jaringan/cloud/CDN;
  semua aset (termasuk font) tertanam di dalam satu binary.

## Teknologi

| Lapisan | Pilihan |
|---|---|
| Backend | Go (satu binary statis, `CGO_ENABLED=0`) |
| Router | [chi](https://github.com/go-chi/chi) |
| Database | SQLite via [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) (pure Go) + [sqlc](https://sqlc.dev) |
| UI input | Svelte + Vite, di-embed via `go:embed` |
| Halaman cetak | Go `html/template`, A4 `@media print` |

## Pengembangan

Prasyarat: Go 1.25+, Node.js + Yarn, [sqlc](https://sqlc.dev) (hanya bila mengubah query).

```sh
# build frontend (sekali di awal, atau setiap mengubah frontend/src)
cd frontend && yarn install && yarn build && cd ..

# jalankan lokal (log di terminal)
make dev && ./surat-tanah.exe

# regenerasi kode query setelah mengubah schema.sql / queries.sql
make generate

# uji & pemeriksaan
make test && make vet
```

Data contoh untuk latihan: `./surat-tanah.exe --seed-contoh` (mengisi nama
fiktif lurah/ketua RT/juru ukur ke database kosong), atau tombol
**✨ Isi data contoh** pada form buat berkas.

## Rilis

Push tag `v*` → GitHub Actions membangun exe (vet + test + build) dan
menerbitkan Release otomatis; versi tertanam dari nama tag dan tampil di
footer aplikasi.

```sh
git tag v1.0.1 && git push origin v1.0.1
```

Build rilis lokal: `make release VERSION=v1.0.1`.

---

*SITANAH · © Kukerta UNRI Kec. Dumai Timur 2026*
