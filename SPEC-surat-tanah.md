# Spesifikasi Lengkap — Aplikasi Surat Tanah (Kecamatan Dumai Timur)

> Dokumen mandiri. Kerjakan dari **folder kosong**. Tidak ada kode/konteks sebelumnya.
> Kerjakan **bertahap per fase** (Bagian 13), konfirmasi tiap fase sebelum lanjut.
> Teks ketiga surat pada Bagian 10 diambil dari **blangko resmi yang sedang dipakai** — pertahankan kata demi kata, termasuk typo/inkonsistensinya. Jangan "diperbaiki" sendiri.

---

## 1. Ringkasan

Aplikasi desktop **standalone** untuk membuat & mencetak berkas pengurusan tanah garapan di **Kantor Camat Dumai Timur**. Petugas mengisi data **sekali**, aplikasi menyimpan ke database lokal dan menghasilkan **3 surat siap cetak**:

1. **Surat Pernyataan Tidak Bersengketa**
2. **Berita Acara Pengukuran Tanah**
3. **Sceets Kaart (Peta Situasi Tanah)**

**Tujuan utama aplikasi: mencegah duplikasi surat atas bidang tanah yang sama**, karena surat ganda berujung pada sengketa tanah. Deteksi duplikat dilakukan berdasarkan lokasi + batas-batas tanah.

**Catatan cakupan:** aplikasi dipasang di tingkat **kecamatan** dan melayani **banyak kelurahan** se-Kecamatan Dumai Timur. Karena itu Kelurahan adalah **data input per berkas**, bukan setting global.

**Siapa yang memakai:** aplikasi dijalankan **hanya oleh admin Kecamatan** pada satu komputer kantor. Bukan aplikasi multi-user, bukan aplikasi jaringan, dan tidak diakses oleh kelurahan. Semua konsekuensinya diatur di Bagian 2 dan Bagian 9.

---

## 2. Batasan Non-Negosiasi

- **Zero-install.** Deliverable = satu file `.exe`, tinggal double-click. Tanpa installer/runtime/DLL.
- **Mesin spek rendah.** Idle RAM < 50 MB.
- **Database lokal file tunggal** (SQLite) di samping exe. Backup = copy 1 file. Tanpa internet.
- **LOKAL SEPENUHNYA.** Server HTTP **wajib bind ke `127.0.0.1` saja**, JANGAN `0.0.0.0` atau `:8080`. Aplikasi tidak boleh bisa diakses dari komputer lain di jaringan. Tidak ada sinkronisasi, cloud, API eksternal, CDN, font online, atau panggilan jaringan apa pun — semua aset di-embed dalam binary.
- **Single-user.** Dijalankan oleh **admin Kecamatan** di satu komputer. Tidak ada sistem role/hak akses bertingkat.
- **`CGO_ENABLED=0` wajib bisa** (cross-compile ke Windows tanpa C compiler).
- **Surat POLOS** — tanpa kop/letterhead/logo. Hanya judul + isi + blok tanda tangan.

---

## 3. Stack (fixed — jangan diganti)

| Layer | Pilihan |
|---|---|
| Backend | **Go** (single static binary) |
| Router | **`github.com/go-chi/chi/v5`** |
| Database | **`modernc.org/sqlite`** (pure-Go). JANGAN `mattn/go-sqlite3` |
| Query | **`sqlc`** |
| Auth | **`golang.org/x/crypto/bcrypt`** + session cookie |
| UI input | **Svelte** → `dist/`, di-embed via `//go:embed` |
| Halaman cetak | **Go `html/template`** (server-rendered), TERPISAH dari Svelte |
| Kertas | **A4 (210 × 297 mm)**, CSS `@media print`, font sans-serif (Arial — [VERIFIKASI]) |

---

## 4. Arsitektur

```
surat-tanah.exe (satu binary)
├── //go:embed frontend/dist  → UI input (Svelte SPA)
├── html/template             → 3 halaman cetak (A4, polos)
├── chi + JSON API            → /api/* (auth, master data, berkas)
├── modernc.org/sqlite        → surat-tanah.db (auto-create di samping exe)
└── auto-open browser         → http://127.0.0.1:8080  (bind localhost saja)
```

- `GET /` + aset → Svelte dari `embed.FS`
- `/api/...` → JSON API
- `GET /berkas/{id}/cetak` → 3 surat via `html/template`, tab baru, siap Ctrl+P
- Halaman cetak **wajib** `html/template` polos (bukan komponen Svelte).

---

## 5. Keputusan Final

- **Satu berkas tanah = satu kali input → 3 surat.**
- **Deteksi duplikat = PERINGATAN, bukan blokir.** Petugas boleh lanjut kalau yakin (lihat 7.1).
- **Kunci duplikat = Kelurahan + Jalan/Gang + RT + 4 nama batas** (Utara, Selatan, Barat, Timur).
- **Kotak peta di Sceets Kaart = kotak kosong**, digambar tangan setelah dicetak. Tidak ada upload/tools gambar.
- **Saksi sempadan = 4 orang** (di ketiga surat).
- **Penanda tangan: Lurah + Ketua RT** (bukan Camat), plus **2 Juru Ukur** (Kelurahan & Kecamatan) di Berita Acara dan Sceets Kaart.
- **Surat polos**, tanpa kop.
- **Dijalankan admin Kecamatan, lokal saja.** Bind `127.0.0.1`, satu user, tanpa role.

---

## 6. Data Model

```sql
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nama          TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'petugas',
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

-- MASTER: Kelurahan se-Kecamatan Dumai Timur
CREATE TABLE kelurahan (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    nama       TEXT NOT NULL UNIQUE,     -- "Jaya Mukti", "Teluk Binjai", dst
    aktif      INTEGER NOT NULL DEFAULT 1
);

-- MASTER: Lurah (1 aktif per kelurahan)
CREATE TABLE lurah (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    kelurahan_id INTEGER NOT NULL REFERENCES kelurahan(id),
    nama         TEXT NOT NULL,
    nip          TEXT NOT NULL,
    aktif        INTEGER NOT NULL DEFAULT 1
);

-- MASTER: Ketua RT (banyak per kelurahan)
CREATE TABLE ketua_rt (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    kelurahan_id INTEGER NOT NULL REFERENCES kelurahan(id),
    nomor_rt     TEXT NOT NULL,          -- "007"
    nama         TEXT NOT NULL,
    aktif        INTEGER NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX idx_rt ON ketua_rt(kelurahan_id, nomor_rt, aktif);

-- MASTER: Juru Ukur (tingkat kelurahan & kecamatan)
CREATE TABLE juru_ukur (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    tingkat      TEXT NOT NULL,          -- 'kelurahan' | 'kecamatan'
    kelurahan_id INTEGER REFERENCES kelurahan(id),  -- NULL bila tingkat kecamatan
    nama         TEXT NOT NULL,
    nip          TEXT NOT NULL,
    aktif        INTEGER NOT NULL DEFAULT 1
);

-- PENGATURAN (1 baris) — identitas kantor
CREATE TABLE pengaturan (
    id             INTEGER PRIMARY KEY CHECK (id = 1),
    kecamatan      TEXT,   -- "Dumai Timur"
    kota           TEXT,   -- "Dumai"
    kantor_pertanahan TEXT -- "Kantor Pertanahan Kota Dumai"
);

-- BERKAS TANAH (induk)
CREATE TABLE berkas_tanah (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    tahun         INTEGER NOT NULL,
    urutan        INTEGER NOT NULL,       -- nomor arsip internal per tahun
    nomor_berkas  TEXT NOT NULL UNIQUE,   -- mis. "0042/2026" (ARSIP INTERNAL, lihat 7.3)
    tanggal_surat TEXT NOT NULL,          -- tanggal di blok TTD ketiga surat

    -- PIHAK PERTAMA (penggarap / yang menyatakan / yang menguasai tanah)
    p1_nama       TEXT NOT NULL,
    p1_umur       INTEGER,
    p1_nik        TEXT NOT NULL,
    p1_pekerjaan  TEXT,
    p1_alamat     TEXT,

    -- PIHAK KEDUA (yang mengganti rugi) — hanya muncul di Berita Acara
    p2_nama       TEXT,
    p2_alamat     TEXT,                   -- [VERIFIKASI] perlu atau tidak

    -- LOKASI TANAH
    kelurahan_id  INTEGER NOT NULL REFERENCES kelurahan(id),
    jalan_gang    TEXT NOT NULL,
    rt            TEXT NOT NULL,
    luas_m2       REAL NOT NULL,          -- luas total
    luas_ganti_rugi_m2 REAL,              -- luas yang diganti rugi (Berita Acara)

    -- DATA PENGUKURAN (Berita Acara)
    tgl_pengukuran TEXT,                  -- disimpan sbg hari/tgl/bulan/tahun saat render
    juru_ukur_kel_id INTEGER REFERENCES juru_ukur(id),
    juru_ukur_kec_id INTEGER REFERENCES juru_ukur(id),

    -- PEJABAT
    lurah_id      INTEGER REFERENCES lurah(id),
    ketua_rt_id   INTEGER REFERENCES ketua_rt(id),

    -- DUPLIKAT
    dup_key       TEXT NOT NULL,          -- hasil normalisasi, lihat 7.1
    dup_override_alasan TEXT,             -- diisi bila petugas lanjut meski ada peringatan
    dup_override_by     INTEGER REFERENCES users(id),
    dup_override_at     TEXT,

    status        TEXT NOT NULL DEFAULT 'terbit',
    created_by    INTEGER REFERENCES users(id),
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX idx_berkas_urutan ON berkas_tanah(tahun, urutan);
CREATE INDEX idx_berkas_dupkey ON berkas_tanah(dup_key);
CREATE INDEX idx_berkas_lokasi ON berkas_tanah(kelurahan_id, jalan_gang, rt);

-- BATAS TANAH (tepat 4 baris per berkas)
CREATE TABLE batas_tanah (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id     INTEGER NOT NULL REFERENCES berkas_tanah(id),
    arah          TEXT NOT NULL,          -- 'utara' | 'selatan' | 'barat' | 'timur'
    nama_pemilik  TEXT NOT NULL,          -- "berbatas dengan tanah <nama>"
    nama_normal   TEXT NOT NULL,          -- hasil normalisasi utk deteksi duplikat
    panjang_meter REAL
);
CREATE UNIQUE INDEX idx_batas ON batas_tanah(berkas_id, arah);
CREATE INDEX idx_batas_normal ON batas_tanah(nama_normal);

-- SAKSI SEMPADAN (tepat 4 per berkas)
CREATE TABLE saksi_sempadan (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id INTEGER NOT NULL REFERENCES berkas_tanah(id),
    urutan    INTEGER NOT NULL,           -- 1..4
    nama      TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_saksi ON saksi_sempadan(berkas_id, urutan);
```

---

## 7. Aturan Bisnis

### 7.1 Deteksi Duplikat (FITUR INTI)

**Tujuan:** mencegah dua surat terbit atas bidang tanah yang sama → mencegah sengketa.

**Normalisasi teks** (wajib, karena nama ditulis bervariasi). Buat helper `normalisasiNama(s string) string`:
1. Ubah ke huruf kecil semua.
2. Buang gelar/sapaan di awal: `h.`, `hj.`, `haji`, `hajjah`, `bpk`, `bapak`, `ibu`, `tn.`, `ny.`, `alm.`, `almarhum`. **[VERIFIKASI] daftar lengkapnya.**
3. Buang tanda baca (titik, koma, apostrof).
4. Rapatkan spasi ganda jadi satu, trim.

Contoh: `H. Ahmad Yani` · `HAJI AHMAD YANI` · `h.ahmad  yani` → semuanya jadi `ahmad yani`.

**Kunci duplikat (`dup_key`):**
```
dup_key = kelurahan_id | normalisasi(jalan_gang) | normalisasi(rt) |
          normal(batas_utara) | normal(batas_selatan) |
          normal(batas_barat) | normal(batas_timur)
```
(Nama batas diurutkan sesuai arah tetap: utara, selatan, barat, timur.)

**Tiga tingkat kecocokan yang ditampilkan ke petugas:**

| Tingkat | Kondisi | Tampilan |
|---|---|---|
| **Sama persis** | `dup_key` identik | Peringatan **MERAH** |
| **Mirip kuat** | Lokasi sama (kelurahan+jalan+RT) **dan** ≥ 2 dari 4 nama batas cocok | Peringatan **KUNING** |
| **Perlu dicek** | Lokasi sama saja (kelurahan+jalan+RT) | Info **ABU-ABU** |

**Perilaku (sesuai keputusan: peringatan, bukan blokir):**
- Cek dijalankan **saat petugas selesai mengisi batas-batas**, sebelum submit (endpoint `POST /api/berkas/cek-duplikat`) — supaya ketahuan lebih awal, bukan setelah semua diketik.
- Bila ada temuan, tampilkan **daftar berkas yang mirip**: nomor berkas, tanggal, nama Pihak Pertama, kelurahan/jalan/RT, luas, dan **keempat nama batasnya** — agar petugas bisa membandingkan sendiri.
- Sediakan tombol untuk **membuka/mencetak ulang berkas lama** (kemungkinan besar itu yang dicari, bukan bikin baru).
- Petugas tetap bisa lanjut. Bila lanjut saat status **MERAH**, **wajib isi alasan** singkat → simpan ke `dup_override_alasan`, `dup_override_by`, `dup_override_at`. Untuk KUNING/ABU-ABU cukup konfirmasi.
- Berkas yang dibuat lewat override **diberi tanda** di daftar berkas (mis. ikon peringatan), agar mudah diaudit.

**Pencarian tambahan (bantu petugas):** sediakan pencarian berkas by nama pemilik batas — supaya bisa cek "tanah yang berbatasan dengan si A sudah pernah diurus belum".

### 7.2 Editability
- Berkas boleh diedit selama diperlukan (tidak ada lock keras seperti aplikasi lain), **[VERIFIKASI]** — tetapi:
- Setiap perubahan pada **lokasi atau batas-batas** harus **memicu ulang pengecekan duplikat** dan memperbarui `dup_key`.
- **[VERIFIKASI]** Apakah setelah dicetak berkas perlu dikunci?

### 7.3 Nomor berkas
- Ketiga blangko asli **TIDAK memiliki nomor surat/register**. Karena itu `nomor_berkas` bersifat **arsip internal aplikasi** (untuk pencarian & rujukan), **tidak dicetak** di surat.
- **[VERIFIKASI]** Apakah kantor ingin nomor register dicetak di surat? Bila ya, tentukan formatnya.

### 7.4 Pejabat & Juru Ukur
- **Lurah** diambil otomatis dari kelurahan yang dipilih (yang `aktif`).
- **Ketua RT** diambil dari (kelurahan + nomor RT) yang dipilih.
- **Juru Ukur Kelurahan & Kecamatan** dipilih dari master data saat input.
- Bila data belum ada di master, tampilkan pesan yang mengarahkan petugas menambah di menu Master Data.

### 7.5 Turunan otomatis
- **Tanggal pengukuran** di Berita Acara ditulis dalam format: `Pada hari ini {Hari} Tanggal {DD} Bulan {NamaBulan} Tahun {YYYY}`. Buat helper konversi tanggal → nama hari & bulan **dalam Bahasa Indonesia** (Senin…Minggu, Januari…Desember).
- **Luas** ditampilkan dengan pemisah ribuan sesuai kebiasaan Indonesia. **[VERIFIKASI]** perlu terbilang atau tidak.

---

## 8. Flow Aplikasi

1. **Login**
2. **Daftar Berkas** — list, cari (nama pemohon, NIK, kelurahan, jalan, nama batas), penanda berkas hasil override
3. **Form Buat Berkas** (Svelte, satu form):
   - **Pihak Pertama**: Nama, Umur, NIK, Pekerjaan, Alamat
   - **Lokasi Tanah**: Kelurahan (dropdown master), Jalan/Gang, RT (dropdown Ketua RT), Luas (M²)
   - **Batas-batas** (4 arah tetap): nama pemilik tanah + panjang (Meter)
     → begitu keempat batas terisi, **jalankan cek duplikat** dan tampilkan hasilnya
   - **Data Pengukuran**: tanggal pengukuran, Juru Ukur Kelurahan, Juru Ukur Kecamatan
   - **Pihak Kedua** (ganti rugi): Nama, luas yang diganti rugi (M²)
   - **Saksi Sempadan** (4 orang): Nama
   - Tanggal surat
   - Submit → (bila ada peringatan merah, minta alasan) → simpan
4. **Detail Berkas** — data lengkap + status duplikat + tombol **Cetak**
5. **Cetak** — `GET /berkas/{id}/cetak` → 3 surat, tiap surat 1 halaman A4, `page-break-after: always`
6. **Menu Master Data** — CRUD Kelurahan, Lurah, Ketua RT, Juru Ukur
7. **Menu Pengaturan** — identitas kecamatan/kota/kantor pertanahan

---

## 9. Auth & Pengguna (single-user, lokal)

Aplikasi dipakai **hanya oleh admin Kecamatan** di satu komputer kantor, tanpa jaringan. Karena itu autentikasi dibuat **sesederhana mungkin** — cukup untuk mencegah orang lain yang lewat/memakai komputer itu membuka data, bukan untuk menahan serangan jaringan.

**Aturan:**
- **Satu akun saja** (admin Kecamatan). **Tidak ada sistem role**, tidak ada manajemen user, tidak ada pendaftaran user baru.
- Login: username + password. Password disimpan sebagai **hash bcrypt** (jangan plaintext).
- Sesi: cookie `HttpOnly`, `SameSite=Lax`. **Tidak perlu `Secure`/HTTPS** karena murni localhost.
- Sediakan menu **Ganti Password** untuk admin.
- Middleware chi: semua route kecuali `/login` wajib sesi valid.
- Kolom `created_by` di tabel berkas tetap dipertahankan untuk jejak audit (terutama override duplikat), meski penggunanya cuma satu.

**Seeder** (jalan saat DB kosong / flag `--seed`):
- Buat akun admin default (username `admin`, password default) dan **paksa ganti password saat login pertama**.
- Isi baris `pengaturan` (id=1) default: kecamatan "Dumai Timur", kota "Dumai", kantor pertanahan "Kantor Pertanahan Kota Dumai".
- **Seed daftar kelurahan se-Kecamatan Dumai Timur** — **[VERIFIKASI] daftar lengkapnya dari user.**

**Yang TIDAK perlu dibuat** (jangan over-engineer): multi-user, role/permission, reset password via email, rate limiting, CSRF token lintas origin, audit log kompleks, HTTPS/TLS.

---

## 10. TEMPLATE SURAT (teks resmi — JANGAN diubah kata-katanya)

Aturan umum:
- **Tanpa kop/logo.** Judul di tengah, huruf kapital, bold.
- Isi **justify**. Titik-titik pada blangko diganti nilai input, dengan **garis bawah** agar mirip blangko asli.
- Pertahankan typo/inkonsistensi asli (lihat Bagian 11).
- Beri jarak vertikal cukup pada blok tanda tangan (tanda tangan basah + meterai).

---

### 10.1 SURAT PERNYATAAN TIDAK BERSENGKETA

**Judul:** `SURAT PERNYATAAN TIDAK BERSENGKETA`

**Pembuka:**
> Saya yang bertanda tangan dibawah ini

**Blok identitas (label rata kiri, titik dua sejajar):**
```
Nama       : {{P1.Nama}}
U m u r    : {{P1.Umur}} Tahun
NIK        : {{P1.NIK}}
Pekerjaan  : {{P1.Pekerjaan}}
A l a m a t: {{P1.Alamat}}
```
*(Perhatikan penulisan asli dengan spasi: "U m u r", "A l a m a t".)*

**Paragraf pernyataan:**
> dengan pikiran dan akal yang sehat serta tidak dipengaruhi oleh siapapun juga telah menyatakan bahwa benar saya mengusahakan/menggarap sebidang tanah yang terletak di jalan/gang {{JalanGang}} RT {{RT}} Kelurahan {{Kelurahan}} Kecamatan {{Kecamatan}} Kota {{Kota}} seluas {{Luas}} M².

**Batas-batas:**
> dengan batas-batas sebagai berikut :

| | | | |
|---|---|---|---|
| Sebelah Utara | Berbatas dengan tanah | {{BatasUtara.Nama}} | {{BatasUtara.Meter}} Meter |
| Sebelah Selatan | Berbatas dengan tanah | {{BatasSelatan.Nama}} | {{BatasSelatan.Meter}} Meter |
| Sebelah Barat | Berbatas dengan tanah | {{BatasBarat.Nama}} | {{BatasBarat.Meter}} Meter |
| Sebelah Timur | Berbatas dengan tanah | {{BatasTimur.Nama}} | {{BatasTimur.Meter}} Meter |

*(Tanpa garis tabel — tampilkan seperti blangko: label kiri, nama di garis titik-titik, satuan "Meter" di kanan.)*

**Paragraf tidak bersengketa:**
> Selama saya mengusahakan tanah tersebut tidak pernah terjadi persengketaan dengan batas tanah orang lain atau persengketaan lainnya dan sama sekali tidak pernah bersangkutan dengan pihak manapun seperti Kredit Bank, digadaikan dan lain sebagainya. Apabila terjadi tuntutan dari pihak manapun juga, maka saya tidak melibatkan pihak pemerintah dan saksi-saksi yang bertanda tangan dalam surat keterangan ganti kerugian ini.

**Penutup:**
> Demikian Surat Pernyataan ini saya buat dengan sebenarnya untuk dapat dipergunakan seperlunya.

**Blok tanda tangan (rata kanan):**
```
                            {{Kota}}, {{TanggalSurat}}
                            Saya Yang Memberi Pernyataan,

                            ┌──────────────┐
                            │  Materai     │
                            │  10.000,-    │
                            └──────────────┘

                            ({{P1.Nama}})
```

**Saksi sempadan (kiri):**
```
Saksi – saksi Sempadan :

1. ________________   ( {{Saksi1.Nama}} )
2. ________________   ( {{Saksi2.Nama}} )
3. ________________   ( {{Saksi3.Nama}} )
4. ________________   ( {{Saksi4.Nama}} )
```

**Blok pejabat:**
```
                    MENGETAHUI :

LURAH {{Kelurahan}}                    KETUA RT {{RT}}


({{Lurah.Nama}})                       ({{KetuaRT.Nama}})
NIP. {{Lurah.NIP}}
```
**[VERIFIKASI]** Pada blangko, NIP Lurah tercetak di bawah garis; pastikan posisinya.

---

### 10.2 BERITA ACARA PENGUKURAN TANAH

**Judul:** `BERITA ACARA PENGUKURAN TANAH`

**Paragraf pembuka:**
> Pada hari ini {{Hari}} Tanggal {{Tanggal}} Bulan {{Bulan}} Tahun {{Tahun}}, kami petugas Juru Ukur telah melaksanakan pengukuran tanah atas nama {{P1.Nama}} dan dihadiri langsung oleh PIHAK PERTAMA yang terletak di Jln / Gang {{JalanGang}} RT {{RT}} Kelurahan {{Kelurahan}} seluas {{Luas}} M² dengan batas-batas sebagai berikut :

**Batas-batas** — format sama dengan Surat 1 (4 arah, nama + Meter).

**Paragraf ganti rugi:**
> Tanah garapan sebagaimana dimaksud akan diganti rugi oleh PIHAK KEDUA seluas {{LuasGantiRugi}} m²

**Penutup:**
> Demikian Berita Acara Pengukuran Tanah ini dibuat dengan sebenarnya dan dapat dipergunakan seperlunya.

**Tanggal (rata kanan):**
```
                            {{Kota}}, {{TanggalSurat}}
```

**Blok tanda tangan pihak (2 kolom):**
```
PIHAK PERTAMA,                         PIHAK KEDUA,


...........................            ...........................
({{P1.Nama}})                          ({{P2.Nama}})
```

**Kolom kiri — Saksi Sempadan:**
```
Saksi-Saksi Sempadan :

1. ________________ : ____________
2. ________________ : ____________
3. ________________ : ____________
4. ________________ : ____________
```
*(Blangko asli memakai pola `nama : tanda tangan`. Isi kolom nama dengan {{Saksi1..4.Nama}}.)*

**Kolom kanan — Juru Ukur:**
```
1. Petugas Juru Ukur Kelurahan

   ...........................
   {{JuruUkurKel.Nama}}
   NIP. {{JuruUkurKel.NIP}}

2. Petugas Juru Ukur Kecamatan

   ...........................
   {{JuruUkurKec.Nama}}
   NIP : {{JuruUkurKec.NIP}}
```

**Blok pejabat:**
```
                    Mengetahui :

LURAH {{Kelurahan}}                    KETUA RT {{RT}}


({{Lurah.Nama}})                       ({{KetuaRT.Nama}})
NIP. {{Lurah.NIP}}
```

**[VERIFIKASI]** Pada scan terdapat tulisan tangan "PIHAK I" dan "PIHAK II" di dekat judul & blok Lurah — kemungkinan catatan petugas soal siapa yang menandatangani di mana. Konfirmasi maksudnya sebelum diimplementasikan.

---

### 10.3 SCEETS KAART (PETA SITUASI TANAH)

**Judul (2 baris, tengah):**
```
SCEETS KAART
(PETA SITUASI TANAH)
```

**Paragraf pembuka:**
> Sebidang tanah yang akan ditetapkan status haknya oleh {{KantorPertanahan}} yang terletak di :

**Blok identitas lokasi:**
```
Jalan/Gang     : {{JalanGang}}
Desa/Kelurahan : {{Kelurahan}}
Kecamatan      : {{Kecamatan}}
Kota           : {{Kota}}
Luas Tanah     : ± {{Luas}} M²
Dikuasai Oleh  : {{P1.Nama}}
```

**KOTAK PETA:**
- Kotak persegi besar bergaris, **KOSONG** (untuk digambar tangan setelah dicetak).
- Tinggi kira-kira **setengah halaman** (± 12–13 cm) agar cukup untuk sketsa.
- Di dalam kotak, pojok kiri atas: **simbol mata angin (penunjuk Utara)** — gambar sederhana pakai SVG/CSS, jangan file gambar eksternal.
- **[VERIFIKASI]** ukuran kotak yang pas.

**Blok tanda tangan (rata kanan):**
```
                            {{Kota}}, {{TanggalSurat}}
                            Yang Menguasai Tanah,


                            ( {{P1.Nama}} )
```

**Dua kolom bawah:**
```
Saksi – saksi Sempadan :          Juru Ukur :

1. ____________ ( {{Saksi1}} )    1. ___________ ( {{JuruUkurKel.Nama}} )
                                     NIP. {{JuruUkurKel.NIP}}
2. ____________ ( {{Saksi2}} )
                                  2. ___________ ( {{JuruUkurKec.Nama}} )
3. ____________ ( {{Saksi3}} )       NIP. {{JuruUkurKec.NIP}}

4. ____________ ( {{Saksi4}} )
```

**Blok pejabat:**
```
                    Mengetahui :

LURAH {{Kelurahan}}                    KETUA RT {{RT}}


({{Lurah.Nama}})                       ({{KetuaRT.Nama}})
NIP. {{Lurah.NIP}}
```

---

## 11. Catatan Blangko Asli (SENGAJA dipertahankan)

Jangan "memperbaiki" tanpa persetujuan user:

1. **"SCEETS KAART"** — ejaan asli pada blangko (dari Belanda *schetskaart*). Pertahankan apa adanya.
2. **Surat Pernyataan** menyebut "...saksi-saksi yang bertanda tangan dalam **surat keterangan ganti kerugian** ini", padahal judulnya Surat Pernyataan. Pertahankan.
3. Penulisan berspasi pada label: **"U m u r"**, **"A l a m a t"**. Pertahankan.
4. Penulisan satuan tidak konsisten antar surat: **"M²"** (Surat Pernyataan & Sceets Kaart) vs **"m²"** (baris ganti rugi di Berita Acara). Pertahankan sesuai masing-masing blangko.
5. Sceets Kaart memakai **"± "** sebelum luas; surat lain tidak.

**[VERIFIKASI]** Konfirmasi ke user apakah semua ini dipertahankan.

---

## 12. Daftar [VERIFIKASI]

- **Daftar lengkap kelurahan** se-Kecamatan Dumai Timur (untuk seeder).
- Arti tulisan tangan "PIHAK I"/"PIHAK II" pada blangko Berita Acara.
- Apakah Pihak Kedua perlu data lain (NIK/alamat) selain nama.
- Apakah kantor ingin nomor register dicetak di surat (blangko asli tidak ada).
- Apakah berkas dikunci setelah dicetak, atau tetap bisa diedit.
- Daftar gelar/sapaan yang dibuang saat normalisasi nama.
- Font & margin cetak A4 yang tepat.
- Ukuran kotak peta Sceets Kaart.
- Apakah luas perlu ditulis terbilang.

---

## 13. Urutan Pengerjaan (per fase, konfirmasi tiap fase)

**Fase 0 — Scaffold**
1. `go mod init surat-tanah`; tambah chi + `modernc.org/sqlite`.
2. Buka/buat DB (belum ada tabel), `PRAGMA journal_mode=WAL`, `db.Ping()`.
3. Server chi **bind ke `127.0.0.1` saja** + `GET /` "Hello World" + auto-open browser ke `http://127.0.0.1:{PORT}`.
4. Pastikan `go build` **dan** cross-compile `.exe` (CGO off) sukses.

**Fase A — Fondasi**
5. Skema DB penuh + migrasi idempotent + sqlc.
6. Seeder (admin default + pengaturan + daftar kelurahan).
7. Auth single-user: login, logout, ganti password, session middleware, bcrypt. Pastikan server **bind `127.0.0.1`**.

**Fase B — Master Data**
8. CRUD Kelurahan, Lurah, Ketua RT, Juru Ukur.
9. Menu Pengaturan.

**Fase C — Inti + Deteksi Duplikat**
10. Model berkas + helper `normalisasiNama()` + pembentukan `dup_key` + helper tanggal Indonesia (hari & bulan). **Tulis unit test untuk normalisasi & dup_key.**
11. Endpoint `POST /api/berkas/cek-duplikat` dengan 3 tingkat kecocokan. Uji via API dulu.
12. Form Svelte buat berkas + panel hasil cek duplikat (merah/kuning/abu) + alur override beralasan.
13. Detail berkas + edit (re-cek duplikat bila lokasi/batas berubah).

**Fase D — Cetak**
14. 3 template `html/template` sesuai Bagian 10, A4, `page-break-after: always`, route `/berkas/{id}/cetak`.
15. Uji cetak/Save-as-PDF, bandingkan dengan blangko asli, rapikan margin & jarak TTD.
16. Polish: pencarian berkas (termasuk **cari by nama pemilik batas**), penanda berkas override, validasi.

**Selesaikan Fase 0 dan tunjukkan hasilnya sebelum lanjut.**
