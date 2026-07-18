-- ===== USERS =====

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, nama, role, must_change_password)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = ?, must_change_password = 0 WHERE id = ?;

-- ===== PENGATURAN =====

-- name: EnsurePengaturanRow :exec
INSERT INTO pengaturan (id, kecamatan, kota, kantor_pertanahan)
VALUES (1, 'Dumai Timur', 'Dumai', 'Kantor Pertanahan Kota Dumai')
ON CONFLICT(id) DO NOTHING;

-- name: GetPengaturan :one
SELECT * FROM pengaturan WHERE id = 1;

-- name: UpdatePengaturan :exec
UPDATE pengaturan SET kecamatan = ?, kota = ?, kantor_pertanahan = ? WHERE id = 1;

-- ===== KELURAHAN =====

-- name: CountKelurahan :one
SELECT COUNT(*) FROM kelurahan;

-- name: ListKelurahan :many
SELECT * FROM kelurahan ORDER BY nama;

-- name: GetKelurahan :one
SELECT * FROM kelurahan WHERE id = ?;

-- name: CreateKelurahan :one
INSERT INTO kelurahan (nama, aktif) VALUES (?, 1) RETURNING *;

-- name: UpdateKelurahan :exec
UPDATE kelurahan SET nama = ?, aktif = ? WHERE id = ?;

-- name: DeleteKelurahan :exec
DELETE FROM kelurahan WHERE id = ?;

-- ===== LURAH =====

-- name: ListLurah :many
SELECT l.*, k.nama AS kelurahan_nama
FROM lurah l JOIN kelurahan k ON k.id = l.kelurahan_id
ORDER BY k.nama, l.aktif DESC, l.nama;

-- name: GetLurah :one
SELECT * FROM lurah WHERE id = ?;

-- name: CreateLurah :one
INSERT INTO lurah (kelurahan_id, nama, nip, aktif) VALUES (?, ?, ?, ?) RETURNING *;

-- name: UpdateLurah :exec
UPDATE lurah SET kelurahan_id = ?, nama = ?, nip = ?, aktif = ? WHERE id = ?;

-- name: DeleteLurah :exec
DELETE FROM lurah WHERE id = ?;

-- name: DeactivateLurahByKelurahan :exec
UPDATE lurah SET aktif = 0 WHERE kelurahan_id = ?;

-- ===== KETUA RT =====

-- name: ListKetuaRT :many
SELECT r.*, k.nama AS kelurahan_nama
FROM ketua_rt r JOIN kelurahan k ON k.id = r.kelurahan_id
ORDER BY k.nama, r.nomor_rt;

-- name: GetKetuaRT :one
SELECT * FROM ketua_rt WHERE id = ?;

-- name: CreateKetuaRT :one
INSERT INTO ketua_rt (kelurahan_id, nomor_rt, nama, aktif) VALUES (?, ?, ?, ?) RETURNING *;

-- name: UpdateKetuaRT :exec
UPDATE ketua_rt SET kelurahan_id = ?, nomor_rt = ?, nama = ?, aktif = ? WHERE id = ?;

-- name: DeleteKetuaRT :exec
DELETE FROM ketua_rt WHERE id = ?;

-- ===== JURU UKUR =====

-- name: ListJuruUkur :many
SELECT j.*, k.nama AS kelurahan_nama
FROM juru_ukur j LEFT JOIN kelurahan k ON k.id = j.kelurahan_id
ORDER BY j.tingkat, k.nama, j.nama;

-- name: GetJuruUkur :one
SELECT * FROM juru_ukur WHERE id = ?;

-- name: CreateJuruUkur :one
INSERT INTO juru_ukur (tingkat, kelurahan_id, nama, nip, aktif) VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateJuruUkur :exec
UPDATE juru_ukur SET tingkat = ?, kelurahan_id = ?, nama = ?, nip = ?, aktif = ? WHERE id = ?;

-- name: DeleteJuruUkur :exec
DELETE FROM juru_ukur WHERE id = ?;

-- ===== BERKAS TANAH =====

-- name: GetMaxUrutan :one
SELECT COALESCE(MAX(urutan), 0) FROM berkas_tanah WHERE tahun = ?;

-- name: GetActiveLurahByKelurahan :one
SELECT * FROM lurah WHERE kelurahan_id = ? AND aktif = 1 LIMIT 1;

-- name: CreateBerkas :one
INSERT INTO berkas_tanah (
    tahun, urutan, nomor_berkas, tanggal_surat,
    p1_nama, p1_umur, p1_nik, p1_pekerjaan, p1_alamat,
    p2_nama, p2_alamat,
    kelurahan_id, jalan_gang, rt, luas_m2, luas_ganti_rugi_m2,
    tgl_pengukuran, juru_ukur_kel_id, juru_ukur_kec_id,
    lurah_id, ketua_rt_id, induk_id, arsip_lama,
    dup_key, dup_override_alasan, dup_override_by, dup_override_at,
    created_by
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateBerkas :exec
UPDATE berkas_tanah SET
    tanggal_surat = ?,
    p1_nama = ?, p1_umur = ?, p1_nik = ?, p1_pekerjaan = ?, p1_alamat = ?,
    p2_nama = ?, p2_alamat = ?,
    kelurahan_id = ?, jalan_gang = ?, rt = ?, luas_m2 = ?, luas_ganti_rugi_m2 = ?,
    tgl_pengukuran = ?, juru_ukur_kel_id = ?, juru_ukur_kec_id = ?,
    lurah_id = ?, ketua_rt_id = ?, induk_id = ?, arsip_lama = ?,
    dup_key = ?, dup_override_alasan = ?, dup_override_by = ?, dup_override_at = ?,
    updated_at = datetime('now')
WHERE id = ?;

-- name: GetBerkasDetail :one
SELECT b.*, k.nama AS kelurahan_nama,
    l.nama AS lurah_nama, l.nip AS lurah_nip,
    rt.nama AS ketua_rt_nama, rt.nomor_rt AS ketua_rt_nomor,
    jk.nama AS juru_kel_nama, jk.nip AS juru_kel_nip,
    jc.nama AS juru_kec_nama, jc.nip AS juru_kec_nip,
    i.nomor_berkas AS induk_nomor, i.luas_m2 AS induk_luas
FROM berkas_tanah b
JOIN kelurahan k ON k.id = b.kelurahan_id
LEFT JOIN lurah l ON l.id = b.lurah_id
LEFT JOIN ketua_rt rt ON rt.id = b.ketua_rt_id
LEFT JOIN juru_ukur jk ON jk.id = b.juru_ukur_kel_id
LEFT JOIN juru_ukur jc ON jc.id = b.juru_ukur_kec_id
LEFT JOIN berkas_tanah i ON i.id = b.induk_id
WHERE b.id = ?;

-- name: ListPecahanByInduk :many
SELECT id, nomor_berkas, p1_nama, luas_m2 FROM berkas_tanah WHERE induk_id = ? ORDER BY id;

-- name: ListBerkas :many
SELECT b.id, b.nomor_berkas, b.tanggal_surat, b.p1_nama, b.p1_nik,
    b.jalan_gang, b.rt, b.luas_m2, b.dup_override_alasan, b.arsip_lama, b.created_at,
    k.nama AS kelurahan_nama
FROM berkas_tanah b
JOIN kelurahan k ON k.id = b.kelurahan_id
WHERE ?1 = '' OR
    b.p1_nama LIKE '%' || ?1 || '%' OR
    b.p1_nik LIKE '%' || ?1 || '%' OR
    b.nomor_berkas LIKE '%' || ?1 || '%' OR
    b.jalan_gang LIKE '%' || ?1 || '%' OR
    k.nama LIKE '%' || ?1 || '%' OR
    EXISTS (SELECT 1 FROM batas_tanah bt
            WHERE bt.berkas_id = b.id AND bt.nama_pemilik LIKE '%' || ?1 || '%')
ORDER BY b.created_at DESC, b.id DESC
LIMIT 500;

-- name: ListBerkasByDupPrefix :many
SELECT b.id, b.nomor_berkas, b.tanggal_surat, b.p1_nama, b.jalan_gang, b.rt,
    b.luas_m2, b.dup_key, k.nama AS kelurahan_nama
FROM berkas_tanah b
JOIN kelurahan k ON k.id = b.kelurahan_id
WHERE b.dup_key LIKE ?1 || '%' AND b.id != ?2
ORDER BY b.created_at DESC;

-- ===== BATAS & SAKSI =====

-- name: InsertBatas :exec
INSERT INTO batas_tanah (berkas_id, arah, nama_pemilik, nama_normal, panjang_meter)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteBatasByBerkas :exec
DELETE FROM batas_tanah WHERE berkas_id = ?;

-- name: ListBatasByBerkas :many
SELECT * FROM batas_tanah WHERE berkas_id = ?
ORDER BY CASE arah WHEN 'utara' THEN 1 WHEN 'selatan' THEN 2 WHEN 'barat' THEN 3 ELSE 4 END;

-- name: InsertSaksi :exec
INSERT INTO saksi_sempadan (berkas_id, urutan, nama) VALUES (?, ?, ?);

-- name: DeleteSaksiByBerkas :exec
DELETE FROM saksi_sempadan WHERE berkas_id = ?;

-- name: ListSaksiByBerkas :many
SELECT * FROM saksi_sempadan WHERE berkas_id = ? ORDER BY urutan;
