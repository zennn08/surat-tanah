package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"surat-tanah/internal/surat"
)

// titik placeholder untuk isian yang kosong (diisi tangan setelah dicetak).
const titik = "…………………"

// batasCetak satu baris batas pada surat.
type batasCetak struct {
	Arah    string // "Utara" dst
	Nama    string
	Panjang string
}

type cetakData struct {
	NomorBerkas string

	P1Nama, P1Umur, P1Nik, P1Pekerjaan, P1Alamat string
	P2Nama                                       string

	Kelurahan, KelurahanUpper, Kecamatan, Kota, KantorPertanahan string
	JalanGang, RT                                                string
	Luas, LuasGantiRugi                                          string

	Hari, Tanggal, Bulan, Tahun string // tanggal pengukuran (Berita Acara)
	TanggalSurat                string // "18 Juli 2026"

	Batas []batasCetak
	Saksi []string

	LurahNama, LurahNip     string
	KetuaRTNama             string
	JuruKelNama, JuruKelNip string
	JuruKecNama, JuruKecNip string
}

// angkaID memformat angka dengan pemisah ribuan Indonesia: 12345.5 → "12.345,5".
func angkaID(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	intPart, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, d := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(d)
	}
	if frac != "" {
		return b.String() + "," + frac
	}
	return b.String()
}

// atauTitik mengembalikan nilai, atau titik-titik bila kosong (isi tangan).
func atauTitik(s string) string {
	if strings.TrimSpace(s) == "" {
		return titik
	}
	return s
}

// Cetak: GET /berkas/{id}/cetak — 3 surat A4 siap Ctrl+P (SPEC Bagian 10).
func (h *Handler) Cetak(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		http.Error(w, "id tidak valid", http.StatusBadRequest)
		return
	}
	b, err := h.q.GetBerkasDetail(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "berkas tidak ditemukan", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "kesalahan server", http.StatusInternalServerError)
		return
	}
	peng, err := h.q.GetPengaturan(r.Context())
	if err != nil {
		http.Error(w, "kesalahan server", http.StatusInternalServerError)
		return
	}
	batasRows, err := h.q.ListBatasByBerkas(r.Context(), id)
	if err != nil {
		http.Error(w, "kesalahan server", http.StatusInternalServerError)
		return
	}
	saksiRows, err := h.q.ListSaksiByBerkas(r.Context(), id)
	if err != nil {
		http.Error(w, "kesalahan server", http.StatusInternalServerError)
		return
	}

	d := cetakData{
		NomorBerkas: b.NomorBerkas,
		P1Nama:      b.P1Nama,
		P1Umur:      titik,
		P1Nik:       b.P1Nik,
		P1Pekerjaan: atauTitik(strOrEmpty(b.P1Pekerjaan)),
		P1Alamat:    atauTitik(strOrEmpty(b.P1Alamat)),
		P2Nama:      atauTitik(strOrEmpty(b.P2Nama)),

		Kelurahan: b.KelurahanNama, KelurahanUpper: strings.ToUpper(b.KelurahanNama),
		Kecamatan:        atauTitik(strOrEmpty(peng.Kecamatan)),
		Kota:             atauTitik(strOrEmpty(peng.Kota)),
		KantorPertanahan: atauTitik(strOrEmpty(peng.KantorPertanahan)),
		JalanGang:        b.JalanGang, RT: b.Rt,
		Luas:          angkaID(b.LuasM2),
		LuasGantiRugi: titik,

		Hari: titik, Tanggal: titik, Bulan: titik, Tahun: titik,
		TanggalSurat: surat.FormatTanggal(b.TanggalSurat),

		LurahNama: atauTitik(strOrEmpty(b.LurahNama)), LurahNip: atauTitik(strOrEmpty(b.LurahNip)),
		KetuaRTNama: atauTitik(strOrEmpty(b.KetuaRtNama)),
		JuruKelNama: atauTitik(strOrEmpty(b.JuruKelNama)), JuruKelNip: atauTitik(strOrEmpty(b.JuruKelNip)),
		JuruKecNama: atauTitik(strOrEmpty(b.JuruKecNama)), JuruKecNip: atauTitik(strOrEmpty(b.JuruKecNip)),
	}
	if b.P1Umur.Valid {
		d.P1Umur = strconv.FormatInt(b.P1Umur.Int64, 10)
	}
	if b.LuasGantiRugiM2.Valid {
		d.LuasGantiRugi = angkaID(b.LuasGantiRugiM2.Float64)
	}
	if b.TglPengukuran.Valid {
		if hari, tgl, bulan, tahun, ok := surat.TanggalIndonesia(b.TglPengukuran.String); ok {
			d.Hari, d.Bulan = hari, bulan
			d.Tanggal = strconv.Itoa(tgl)
			d.Tahun = strconv.Itoa(tahun)
		}
	}
	for _, bt := range batasRows {
		row := batasCetak{
			Arah:    strings.ToUpper(bt.Arah[:1]) + bt.Arah[1:],
			Nama:    bt.NamaPemilik,
			Panjang: titik,
		}
		if bt.PanjangMeter.Valid {
			row.Panjang = angkaID(bt.PanjangMeter.Float64)
		}
		d.Batas = append(d.Batas, row)
	}
	for _, s := range saksiRows {
		d.Saksi = append(d.Saksi, s.Nama)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "cetak", d); err != nil {
		http.Error(w, "gagal merender surat: "+err.Error(), http.StatusInternalServerError)
	}
}
