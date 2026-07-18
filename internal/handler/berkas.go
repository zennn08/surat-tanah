package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"surat-tanah/internal/auth"
	"surat-tanah/internal/db"
	"surat-tanah/internal/surat"
)

var arahList = [4]string{"utara", "selatan", "barat", "timur"}

// ===== Input =====

type batasInput struct {
	Nama         string   `json:"nama"`
	PanjangMeter *float64 `json:"panjang_meter"`
}

type berkasInput struct {
	TanggalSurat string `json:"tanggal_surat"`

	P1Nama      string `json:"p1_nama"`
	P1Umur      *int64 `json:"p1_umur"`
	P1Nik       string `json:"p1_nik"`
	P1Pekerjaan string `json:"p1_pekerjaan"`
	P1Alamat    string `json:"p1_alamat"`

	P2Nama   string `json:"p2_nama"`
	P2Alamat string `json:"p2_alamat"`

	KelurahanID     int64    `json:"kelurahan_id"`
	JalanGang       string   `json:"jalan_gang"`
	KetuaRtID       int64    `json:"ketua_rt_id"`
	LuasM2          float64  `json:"luas_m2"`
	LuasGantiRugiM2 *float64 `json:"luas_ganti_rugi_m2"`

	TglPengukuran string `json:"tgl_pengukuran"`
	JuruUkurKelID *int64 `json:"juru_ukur_kel_id"`
	JuruUkurKecID *int64 `json:"juru_ukur_kec_id"`

	BatasUtara   batasInput `json:"batas_utara"`
	BatasSelatan batasInput `json:"batas_selatan"`
	BatasBarat   batasInput `json:"batas_barat"`
	BatasTimur   batasInput `json:"batas_timur"`

	Saksi []string `json:"saksi"`

	// Pecah bidang: id berkas induk bila tanah ini pecahan dari bidang lama.
	IndukID *int64 `json:"induk_id"`

	// Arsip: true bila berkas ini salinan surat lama (dibuat sebelum aplikasi).
	ArsipLama bool `json:"arsip_lama"`

	OverrideAlasan string `json:"override_alasan"`
}

func (in *berkasInput) batas() [4]batasInput {
	return [4]batasInput{in.BatasUtara, in.BatasSelatan, in.BatasBarat, in.BatasTimur}
}

func (in *berkasInput) validate() error {
	in.TanggalSurat = strings.TrimSpace(in.TanggalSurat)
	in.P1Nama = strings.TrimSpace(in.P1Nama)
	in.P1Nik = strings.TrimSpace(in.P1Nik)
	in.JalanGang = strings.TrimSpace(in.JalanGang)
	in.BatasUtara.Nama = strings.TrimSpace(in.BatasUtara.Nama)
	in.BatasSelatan.Nama = strings.TrimSpace(in.BatasSelatan.Nama)
	in.BatasBarat.Nama = strings.TrimSpace(in.BatasBarat.Nama)
	in.BatasTimur.Nama = strings.TrimSpace(in.BatasTimur.Nama)

	if _, err := time.Parse("2006-01-02", in.TanggalSurat); err != nil {
		return errors.New("tanggal surat wajib diisi (format YYYY-MM-DD)")
	}
	if in.TglPengukuran != "" {
		if _, err := time.Parse("2006-01-02", in.TglPengukuran); err != nil {
			return errors.New("tanggal pengukuran tidak valid")
		}
	}
	if in.P1Nama == "" || in.P1Nik == "" {
		return errors.New("nama dan NIK Pihak Pertama wajib diisi")
	}
	if in.KelurahanID == 0 {
		return errors.New("kelurahan wajib dipilih")
	}
	if in.JalanGang == "" {
		return errors.New("jalan/gang wajib diisi")
	}
	if in.KetuaRtID == 0 {
		return errors.New("RT wajib dipilih")
	}
	if in.LuasM2 <= 0 {
		return errors.New("luas tanah wajib diisi")
	}
	for i, b := range in.batas() {
		if b.Nama == "" {
			return fmt.Errorf("nama batas sebelah %s wajib diisi", arahList[i])
		}
	}
	if len(in.Saksi) != 4 {
		return errors.New("saksi sempadan harus 4 orang")
	}
	for i, s := range in.Saksi {
		in.Saksi[i] = strings.TrimSpace(s)
		if in.Saksi[i] == "" {
			return fmt.Errorf("nama saksi ke-%d wajib diisi", i+1)
		}
	}
	return nil
}

// ===== Deteksi duplikat (SPEC 7.1) =====

type dupMatch struct {
	Tingkat       string      `json:"tingkat"` // merah | kuning | abu
	ID            int64       `json:"id"`
	NomorBerkas   string      `json:"nomor_berkas"`
	TanggalSurat  string      `json:"tanggal_surat"`
	P1Nama        string      `json:"p1_nama"`
	KelurahanNama string      `json:"kelurahan_nama"`
	JalanGang     string      `json:"jalan_gang"`
	Rt            string      `json:"rt"`
	LuasM2        float64     `json:"luas_m2"`
	Batas         []batasView `json:"batas"`
	CocokBatas    int         `json:"cocok_batas"` // berapa dari 4 nama batas yang sama
}

type batasView struct {
	Arah         string   `json:"arah"`
	Nama         string   `json:"nama"`
	PanjangMeter *float64 `json:"panjang_meter"`
}

type dupResult struct {
	Tingkat string     `json:"tingkat"` // merah | kuning | abu | aman
	Matches []dupMatch `json:"matches"`
}

// tingkatRank untuk menentukan tingkat terparah.
var tingkatRank = map[string]int{"aman": 0, "abu": 1, "kuning": 2, "merah": 3}

// cekDuplikat membandingkan lokasi+batas terhadap berkas lama (excludeID diabaikan).
func (h *Handler) cekDuplikat(ctx context.Context, kelurahanID int64, jalan, rt string, batas [4]string, excludeID int64) (dupResult, error) {
	res := dupResult{Tingkat: "aman", Matches: []dupMatch{}}

	fullKey := surat.DupKey(kelurahanID, jalan, rt, batas[0], batas[1], batas[2], batas[3])
	prefix := surat.LokasiPrefix(kelurahanID, jalan, rt)

	rows, err := h.q.ListBerkasByDupPrefix(ctx, db.ListBerkasByDupPrefixParams{
		Column1: sql.NullString{String: prefix, Valid: true},
		ID:      excludeID,
	})
	if err != nil {
		return res, err
	}

	var batasNorm [4]string
	for i, b := range batas {
		batasNorm[i] = surat.NormalisasiNama(b)
	}

	for _, r := range rows {
		candBatas := surat.BatasDariDupKey(r.DupKey)
		cocok := 0
		for i := range batasNorm {
			if batasNorm[i] != "" && batasNorm[i] == candBatas[i] {
				cocok++
			}
		}
		tingkat := "abu"
		if r.DupKey == fullKey {
			tingkat = "merah"
		} else if cocok >= 2 {
			tingkat = "kuning"
		}

		batasRows, err := h.q.ListBatasByBerkas(ctx, r.ID)
		if err != nil {
			return res, err
		}
		bv := make([]batasView, 0, 4)
		for _, b := range batasRows {
			v := batasView{Arah: b.Arah, Nama: b.NamaPemilik}
			if b.PanjangMeter.Valid {
				v.PanjangMeter = &b.PanjangMeter.Float64
			}
			bv = append(bv, v)
		}

		res.Matches = append(res.Matches, dupMatch{
			Tingkat: tingkat, ID: r.ID, NomorBerkas: r.NomorBerkas,
			TanggalSurat: r.TanggalSurat, P1Nama: r.P1Nama,
			KelurahanNama: r.KelurahanNama, JalanGang: r.JalanGang, Rt: r.Rt,
			LuasM2: r.LuasM2, Batas: bv, CocokBatas: cocok,
		})
		if tingkatRank[tingkat] > tingkatRank[res.Tingkat] {
			res.Tingkat = tingkat
		}
	}
	return res, nil
}

// CekDuplikat: POST /api/berkas/cek-duplikat
// Body: {kelurahan_id, jalan_gang, rt, batas_utara..timur: {nama}, exclude_id}
func (h *Handler) CekDuplikat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		KelurahanID  int64      `json:"kelurahan_id"`
		JalanGang    string     `json:"jalan_gang"`
		Rt           string     `json:"rt"`
		BatasUtara   batasInput `json:"batas_utara"`
		BatasSelatan batasInput `json:"batas_selatan"`
		BatasBarat   batasInput `json:"batas_barat"`
		BatasTimur   batasInput `json:"batas_timur"`
		ExcludeID    int64      `json:"exclude_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if in.KelurahanID == 0 || strings.TrimSpace(in.JalanGang) == "" || strings.TrimSpace(in.Rt) == "" {
		writeErr(w, http.StatusBadRequest, "kelurahan, jalan/gang, dan RT wajib diisi")
		return
	}
	res, err := h.cekDuplikat(r.Context(), in.KelurahanID, in.JalanGang, in.Rt,
		[4]string{in.BatasUtara.Nama, in.BatasSelatan.Nama, in.BatasBarat.Nama, in.BatasTimur.Nama}, in.ExcludeID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal mengecek duplikat")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// cekInduk memastikan berkas induk (bila diisi) ada dan bukan dirinya sendiri.
func (h *Handler) cekInduk(ctx context.Context, in *berkasInput, selfID int64) error {
	if in.IndukID == nil {
		return nil
	}
	if *in.IndukID == selfID {
		return errors.New("berkas tidak bisa menjadi induk dirinya sendiri")
	}
	if _, err := h.q.GetBerkasDetail(ctx, *in.IndukID); errors.Is(err, sql.ErrNoRows) {
		return errors.New("berkas induk tidak ditemukan")
	} else if err != nil {
		return err
	}
	return nil
}

// resolveRefs memastikan ketua RT & lurah aktif tersedia; mengembalikan nomor RT + lurah id.
func (h *Handler) resolveRefs(ctx context.Context, in *berkasInput) (rtNomor string, lurahID int64, err error) {
	krt, err := h.q.GetKetuaRT(ctx, in.KetuaRtID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && krt.KelurahanID != in.KelurahanID) {
		return "", 0, errors.New("ketua RT tidak ditemukan di kelurahan itu — tambahkan di Master Data")
	}
	if err != nil {
		return "", 0, err
	}
	lurah, err := h.q.GetActiveLurahByKelurahan(ctx, in.KelurahanID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, errors.New("belum ada Lurah aktif untuk kelurahan itu — tambahkan di Master Data")
	}
	if err != nil {
		return "", 0, err
	}
	return krt.NomorRt, lurah.ID, nil
}

// simpanRelasi menulis ulang 4 batas + 4 saksi milik berkas (dalam tx).
func simpanRelasi(ctx context.Context, qtx *db.Queries, berkasID int64, in *berkasInput) error {
	if err := qtx.DeleteBatasByBerkas(ctx, berkasID); err != nil {
		return err
	}
	if err := qtx.DeleteSaksiByBerkas(ctx, berkasID); err != nil {
		return err
	}
	for i, b := range in.batas() {
		if err := qtx.InsertBatas(ctx, db.InsertBatasParams{
			BerkasID: berkasID, Arah: arahList[i], NamaPemilik: b.Nama,
			NamaNormal:   surat.NormalisasiNama(b.Nama),
			PanjangMeter: nullFloat(b.PanjangMeter),
		}); err != nil {
			return err
		}
	}
	for i, s := range in.Saksi {
		if err := qtx.InsertSaksi(ctx, db.InsertSaksiParams{
			BerkasID: berkasID, Urutan: int64(i + 1), Nama: s,
		}); err != nil {
			return err
		}
	}
	return nil
}

func nullFloat(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *p, Valid: true}
}

func nullIntZero(v int64) sql.NullInt64 {
	if v == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}

// CreateBerkas: POST /api/berkas
func (h *Handler) CreateBerkas(w http.ResponseWriter, r *http.Request) {
	var in berkasInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	rtNomor, lurahID, err := h.resolveRefs(r.Context(), &in)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.cekInduk(r.Context(), &in, 0); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	batas := in.batas()
	dup, err := h.cekDuplikat(r.Context(), in.KelurahanID, in.JalanGang, rtNomor,
		[4]string{batas[0].Nama, batas[1].Nama, batas[2].Nama, batas[3].Nama}, 0)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal mengecek duplikat")
		return
	}
	if dup.Tingkat == "merah" && strings.TrimSpace(in.OverrideAlasan) == "" {
		// SPEC 7.1: lanjut saat MERAH wajib beralasan.
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":    "terdeteksi duplikat persis — wajib isi alasan untuk melanjutkan",
			"duplikat": dup,
		})
		return
	}

	uid, _ := auth.UserIDFromContext(r.Context())
	dupKey := surat.DupKey(in.KelurahanID, in.JalanGang, rtNomor,
		batas[0].Nama, batas[1].Nama, batas[2].Nama, batas[3].Nama)

	tahun, _ := time.Parse("2006-01-02", in.TanggalSurat)
	maxUrutan, err := h.q.GetMaxUrutan(r.Context(), int64(tahun.Year()))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	urutan := maxUrutan.(int64) + 1
	nomor := fmt.Sprintf("%04d/%d", urutan, tahun.Year())

	tx, err := h.sqldb.BeginTx(r.Context(), nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	params := db.CreateBerkasParams{
		Tahun: int64(tahun.Year()), Urutan: urutan, NomorBerkas: nomor,
		TanggalSurat: in.TanggalSurat,
		P1Nama:       in.P1Nama, P1Umur: nullIntPtr(in.P1Umur), P1Nik: in.P1Nik,
		P1Pekerjaan: nullStr(in.P1Pekerjaan), P1Alamat: nullStr(in.P1Alamat),
		P2Nama: nullStr(in.P2Nama), P2Alamat: nullStr(in.P2Alamat),
		KelurahanID: in.KelurahanID, JalanGang: in.JalanGang, Rt: rtNomor,
		LuasM2: in.LuasM2, LuasGantiRugiM2: nullFloat(in.LuasGantiRugiM2),
		TglPengukuran: nullStr(in.TglPengukuran),
		JuruUkurKelID: nullIntPtr(in.JuruUkurKelID), JuruUkurKecID: nullIntPtr(in.JuruUkurKecID),
		LurahID: nullIntZero(lurahID), KetuaRtID: nullIntZero(in.KetuaRtID),
		IndukID: nullIntPtr(in.IndukID), ArsipLama: boolToInt(in.ArsipLama),
		DupKey: dupKey, CreatedBy: nullIntZero(uid),
	}
	if dup.Tingkat == "merah" {
		params.DupOverrideAlasan = nullStr(strings.TrimSpace(in.OverrideAlasan))
		params.DupOverrideBy = nullIntZero(uid)
		params.DupOverrideAt = nullStr(time.Now().Format(time.RFC3339))
	}

	created, err := qtx.CreateBerkas(r.Context(), params)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan berkas")
		return
	}
	if err := simpanRelasi(r.Context(), qtx, created.ID, &in); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan batas/saksi")
		return
	}
	if err := tx.Commit(); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan berkas")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": created.ID, "nomor_berkas": created.NomorBerkas})
}

// UpdateBerkas: PUT /api/berkas/{id} — edit penuh; dup_key dihitung ulang (SPEC 7.2).
func (h *Handler) UpdateBerkas(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	if _, err := h.q.GetBerkasDetail(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "berkas tidak ditemukan")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}

	var in berkasInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	rtNomor, lurahID, err := h.resolveRefs(r.Context(), &in)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.cekInduk(r.Context(), &in, id); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	batas := in.batas()
	dup, err := h.cekDuplikat(r.Context(), in.KelurahanID, in.JalanGang, rtNomor,
		[4]string{batas[0].Nama, batas[1].Nama, batas[2].Nama, batas[3].Nama}, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal mengecek duplikat")
		return
	}
	if dup.Tingkat == "merah" && strings.TrimSpace(in.OverrideAlasan) == "" {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":    "terdeteksi duplikat persis — wajib isi alasan untuk melanjutkan",
			"duplikat": dup,
		})
		return
	}

	uid, _ := auth.UserIDFromContext(r.Context())
	dupKey := surat.DupKey(in.KelurahanID, in.JalanGang, rtNomor,
		batas[0].Nama, batas[1].Nama, batas[2].Nama, batas[3].Nama)

	params := db.UpdateBerkasParams{
		TanggalSurat: in.TanggalSurat,
		P1Nama:       in.P1Nama, P1Umur: nullIntPtr(in.P1Umur), P1Nik: in.P1Nik,
		P1Pekerjaan: nullStr(in.P1Pekerjaan), P1Alamat: nullStr(in.P1Alamat),
		P2Nama: nullStr(in.P2Nama), P2Alamat: nullStr(in.P2Alamat),
		KelurahanID: in.KelurahanID, JalanGang: in.JalanGang, Rt: rtNomor,
		LuasM2: in.LuasM2, LuasGantiRugiM2: nullFloat(in.LuasGantiRugiM2),
		TglPengukuran: nullStr(in.TglPengukuran),
		JuruUkurKelID: nullIntPtr(in.JuruUkurKelID), JuruUkurKecID: nullIntPtr(in.JuruUkurKecID),
		LurahID: nullIntZero(lurahID), KetuaRtID: nullIntZero(in.KetuaRtID),
		IndukID: nullIntPtr(in.IndukID), ArsipLama: boolToInt(in.ArsipLama),
		DupKey: dupKey, ID: id,
	}
	// Status override mengikuti kondisi terkini: hanya tersimpan bila masih merah.
	if dup.Tingkat == "merah" {
		params.DupOverrideAlasan = nullStr(strings.TrimSpace(in.OverrideAlasan))
		params.DupOverrideBy = nullIntZero(uid)
		params.DupOverrideAt = nullStr(time.Now().Format(time.RFC3339))
	}

	tx, err := h.sqldb.BeginTx(r.Context(), nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	if err := qtx.UpdateBerkas(r.Context(), params); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
		return
	}
	if err := simpanRelasi(r.Context(), qtx, id, &in); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan batas/saksi")
		return
	}
	if err := tx.Commit(); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListBerkas: GET /api/berkas?q=
func (h *Handler) ListBerkas(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, err := h.q.ListBerkas(r.Context(), q)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat daftar berkas")
		return
	}
	type row struct {
		ID            int64   `json:"id"`
		NomorBerkas   string  `json:"nomor_berkas"`
		TanggalSurat  string  `json:"tanggal_surat"`
		P1Nama        string  `json:"p1_nama"`
		P1Nik         string  `json:"p1_nik"`
		KelurahanNama string  `json:"kelurahan_nama"`
		JalanGang     string  `json:"jalan_gang"`
		Rt            string  `json:"rt"`
		LuasM2        float64 `json:"luas_m2"`
		Override      bool    `json:"override"`
		ArsipLama     bool    `json:"arsip_lama"`
	}
	out := make([]row, 0, len(rows))
	for _, b := range rows {
		out = append(out, row{
			ID: b.ID, NomorBerkas: b.NomorBerkas, TanggalSurat: b.TanggalSurat,
			P1Nama: b.P1Nama, P1Nik: b.P1Nik, KelurahanNama: b.KelurahanNama,
			JalanGang: b.JalanGang, Rt: b.Rt, LuasM2: b.LuasM2,
			Override:  b.DupOverrideAlasan.Valid && b.DupOverrideAlasan.String != "",
			ArsipLama: b.ArsipLama != 0,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// GetBerkas: GET /api/berkas/{id}
func (h *Handler) GetBerkas(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	b, err := h.q.GetBerkasDetail(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "berkas tidak ditemukan")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	batasRows, err := h.q.ListBatasByBerkas(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	saksiRows, err := h.q.ListSaksiByBerkas(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}

	batas := make([]batasView, 0, 4)
	for _, bt := range batasRows {
		v := batasView{Arah: bt.Arah, Nama: bt.NamaPemilik}
		if bt.PanjangMeter.Valid {
			v.PanjangMeter = &bt.PanjangMeter.Float64
		}
		batas = append(batas, v)
	}
	saksi := make([]string, 0, 4)
	for _, s := range saksiRows {
		saksi = append(saksi, s.Nama)
	}

	// Pecah bidang: daftar berkas yang menjadikan berkas ini induknya.
	pecahanRows, err := h.q.ListPecahanByInduk(r.Context(), sql.NullInt64{Int64: id, Valid: true})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	type pecahanView struct {
		ID          int64   `json:"id"`
		NomorBerkas string  `json:"nomor_berkas"`
		P1Nama      string  `json:"p1_nama"`
		LuasM2      float64 `json:"luas_m2"`
	}
	pecahan := make([]pecahanView, 0, len(pecahanRows))
	var totalPecahan float64
	for _, p := range pecahanRows {
		pecahan = append(pecahan, pecahanView{ID: p.ID, NomorBerkas: p.NomorBerkas, P1Nama: p.P1Nama, LuasM2: p.LuasM2})
		totalPecahan += p.LuasM2
	}

	out := map[string]any{
		"id": b.ID, "nomor_berkas": b.NomorBerkas, "tahun": b.Tahun,
		"tanggal_surat": b.TanggalSurat,
		"p1_nama":       b.P1Nama, "p1_nik": b.P1Nik,
		"p1_pekerjaan": strOrEmpty(b.P1Pekerjaan), "p1_alamat": strOrEmpty(b.P1Alamat),
		"p2_nama": strOrEmpty(b.P2Nama), "p2_alamat": strOrEmpty(b.P2Alamat),
		"kelurahan_id": b.KelurahanID, "kelurahan_nama": b.KelurahanNama,
		"jalan_gang": b.JalanGang, "rt": b.Rt, "luas_m2": b.LuasM2,
		"tgl_pengukuran": strOrEmpty(b.TglPengukuran),
		"lurah_nama":     strOrEmpty(b.LurahNama), "lurah_nip": strOrEmpty(b.LurahNip),
		"ketua_rt_nama": strOrEmpty(b.KetuaRtNama), "ketua_rt_nomor": strOrEmpty(b.KetuaRtNomor),
		"juru_kel_nama": strOrEmpty(b.JuruKelNama), "juru_kel_nip": strOrEmpty(b.JuruKelNip),
		"juru_kec_nama": strOrEmpty(b.JuruKecNama), "juru_kec_nip": strOrEmpty(b.JuruKecNip),
		"batas": batas, "saksi": saksi,
		"arsip_lama": b.ArsipLama != 0,
		"pecahan": pecahan, "total_luas_pecahan": totalPecahan,
		"override_alasan": strOrEmpty(b.DupOverrideAlasan),
		"override_at":     strOrEmpty(b.DupOverrideAt),
		"created_at":      b.CreatedAt, "updated_at": b.UpdatedAt,
	}
	if b.IndukID.Valid {
		out["induk_id"] = b.IndukID.Int64
		out["induk_nomor"] = strOrEmpty(b.IndukNomor)
		if b.IndukLuas.Valid {
			out["induk_luas"] = b.IndukLuas.Float64
		}
	}
	if b.P1Umur.Valid {
		out["p1_umur"] = b.P1Umur.Int64
	}
	if b.LuasGantiRugiM2.Valid {
		out["luas_ganti_rugi_m2"] = b.LuasGantiRugiM2.Float64
	}
	if b.KetuaRtID.Valid {
		out["ketua_rt_id"] = b.KetuaRtID.Int64
	}
	if b.JuruUkurKelID.Valid {
		out["juru_ukur_kel_id"] = b.JuruUkurKelID.Int64
	}
	if b.JuruUkurKecID.Valid {
		out["juru_ukur_kec_id"] = b.JuruUkurKecID.Int64
	}
	writeJSON(w, http.StatusOK, out)
}
