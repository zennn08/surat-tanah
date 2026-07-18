-- Skema penuh SPEC-surat-tanah.md Bagian 6 (v1).
-- Tambahan di luar spec: users.must_change_password (SPEC §9: paksa ganti
-- password saat login pertama).

CREATE TABLE users (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    username             TEXT NOT NULL UNIQUE,
    password_hash        TEXT NOT NULL,
    nama                 TEXT NOT NULL,
    role                 TEXT NOT NULL DEFAULT 'petugas',
    must_change_password INTEGER NOT NULL DEFAULT 0,
    created_at           TEXT NOT NULL DEFAULT (datetime('now'))
);

-- MASTER: Kelurahan se-Kecamatan Dumai Timur
CREATE TABLE kelurahan (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    nama       TEXT NOT NULL UNIQUE,
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
    nomor_rt     TEXT NOT NULL,
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
    id                INTEGER PRIMARY KEY CHECK (id = 1),
    kecamatan         TEXT,
    kota              TEXT,
    kantor_pertanahan TEXT
);

-- BERKAS TANAH (induk)
CREATE TABLE berkas_tanah (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    tahun         INTEGER NOT NULL,
    urutan        INTEGER NOT NULL,       -- nomor arsip internal per tahun
    nomor_berkas  TEXT NOT NULL UNIQUE,   -- mis. "0042/2026" (ARSIP INTERNAL, tidak dicetak)
    tanggal_surat TEXT NOT NULL,          -- tanggal di blok TTD ketiga surat

    -- PIHAK PERTAMA (penggarap / yang menyatakan / yang menguasai tanah)
    p1_nama       TEXT NOT NULL,
    p1_umur       INTEGER,
    p1_nik        TEXT NOT NULL,
    p1_pekerjaan  TEXT,
    p1_alamat     TEXT,

    -- PIHAK KEDUA (yang mengganti rugi) — hanya muncul di Berita Acara
    p2_nama       TEXT,
    p2_alamat     TEXT,

    -- LOKASI TANAH
    kelurahan_id  INTEGER NOT NULL REFERENCES kelurahan(id),
    jalan_gang    TEXT NOT NULL,
    rt            TEXT NOT NULL,
    luas_m2       REAL NOT NULL,          -- luas total
    luas_ganti_rugi_m2 REAL,              -- luas yang diganti rugi (Berita Acara)

    -- DATA PENGUKURAN (Berita Acara)
    tgl_pengukuran TEXT,
    juru_ukur_kel_id INTEGER REFERENCES juru_ukur(id),
    juru_ukur_kec_id INTEGER REFERENCES juru_ukur(id),

    -- PEJABAT
    lurah_id      INTEGER REFERENCES lurah(id),
    ketua_rt_id   INTEGER REFERENCES ketua_rt(id),

    -- PECAH BIDANG: berkas induk bila tanah ini pecahan/sebagian dari bidang lama
    induk_id      INTEGER REFERENCES berkas_tanah(id),

    -- ARSIP: 1 bila berkas ini salinan surat lama (dibuat sebelum aplikasi ada)
    arsip_lama    INTEGER NOT NULL DEFAULT 0,

    -- DUPLIKAT
    dup_key       TEXT NOT NULL,
    dup_override_alasan TEXT,
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
    nama_pemilik  TEXT NOT NULL,
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
