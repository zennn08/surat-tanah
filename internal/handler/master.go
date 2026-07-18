package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"surat-tanah/internal/db"
)

// ===== KELURAHAN =====

type kelurahanView struct {
	ID    int64  `json:"id"`
	Nama  string `json:"nama"`
	Aktif bool   `json:"aktif"`
}

func toKelurahanView(k db.Kelurahan) kelurahanView {
	return kelurahanView{ID: k.ID, Nama: k.Nama, Aktif: k.Aktif != 0}
}

// ListKelurahan: GET /api/kelurahan
func (h *Handler) ListKelurahan(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListKelurahan(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat kelurahan")
		return
	}
	out := make([]kelurahanView, 0, len(rows))
	for _, k := range rows {
		out = append(out, toKelurahanView(k))
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateKelurahan: POST /api/kelurahan  {nama}
func (h *Handler) CreateKelurahan(w http.ResponseWriter, r *http.Request) {
	var in kelurahanView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	in.Nama = strings.TrimSpace(in.Nama)
	if in.Nama == "" {
		writeErr(w, http.StatusBadRequest, "nama wajib diisi")
		return
	}
	created, err := h.q.CreateKelurahan(r.Context(), in.Nama)
	if isConstraintErr(err) {
		writeErr(w, http.StatusBadRequest, "kelurahan dengan nama itu sudah ada")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan kelurahan")
		return
	}
	writeJSON(w, http.StatusCreated, toKelurahanView(created))
}

// UpdateKelurahan: PUT /api/kelurahan/{id}  {nama, aktif}
func (h *Handler) UpdateKelurahan(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	var in kelurahanView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	in.Nama = strings.TrimSpace(in.Nama)
	if in.Nama == "" {
		writeErr(w, http.StatusBadRequest, "nama wajib diisi")
		return
	}
	if _, err := h.q.GetKelurahan(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "kelurahan tidak ditemukan")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if err := h.q.UpdateKelurahan(r.Context(), db.UpdateKelurahanParams{
		Nama: in.Nama, Aktif: boolToInt(in.Aktif), ID: id,
	}); err != nil {
		if isConstraintErr(err) {
			writeErr(w, http.StatusBadRequest, "kelurahan dengan nama itu sudah ada")
			return
		}
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
		return
	}
	in.ID = id
	writeJSON(w, http.StatusOK, in)
}

// DeleteKelurahan: DELETE /api/kelurahan/{id}
func (h *Handler) DeleteKelurahan(w http.ResponseWriter, r *http.Request) {
	h.deleteRow(w, r, "kelurahan", h.q.DeleteKelurahan)
}

// deleteRow menghapus baris master by id; FK violation → pesan ramah.
func (h *Handler) deleteRow(w http.ResponseWriter, r *http.Request, label string, del func(context.Context, int64) error) {
	id, err := idParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	if err := del(r.Context(), id); err != nil {
		if isConstraintErr(err) {
			writeErr(w, http.StatusBadRequest, label+" masih dipakai oleh data lain, tidak bisa dihapus. Nonaktifkan saja.")
			return
		}
		writeErr(w, http.StatusInternalServerError, "gagal menghapus "+label)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ===== LURAH =====

type lurahView struct {
	ID            int64  `json:"id"`
	KelurahanID   int64  `json:"kelurahan_id"`
	KelurahanNama string `json:"kelurahan_nama,omitempty"`
	Nama          string `json:"nama"`
	Nip           string `json:"nip"`
	Aktif         bool   `json:"aktif"`
}

func (in *lurahView) validate() error {
	in.Nama = strings.TrimSpace(in.Nama)
	in.Nip = strings.TrimSpace(in.Nip)
	if in.KelurahanID == 0 {
		return errors.New("kelurahan wajib dipilih")
	}
	if in.Nama == "" {
		return errors.New("nama wajib diisi")
	}
	if in.Nip == "" {
		return errors.New("NIP wajib diisi")
	}
	return nil
}

// ListLurah: GET /api/lurah
func (h *Handler) ListLurah(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListLurah(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat lurah")
		return
	}
	out := make([]lurahView, 0, len(rows))
	for _, l := range rows {
		out = append(out, lurahView{
			ID: l.ID, KelurahanID: l.KelurahanID, KelurahanNama: l.KelurahanNama,
			Nama: l.Nama, Nip: l.Nip, Aktif: l.Aktif != 0,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateLurah: POST /api/lurah — bila aktif, lurah lain se-kelurahan dinonaktifkan
// (1 aktif per kelurahan) dalam satu transaksi.
func (h *Handler) CreateLurah(w http.ResponseWriter, r *http.Request) {
	var in lurahView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.sqldb.BeginTx(r.Context(), nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	if in.Aktif {
		if err := qtx.DeactivateLurahByKelurahan(r.Context(), in.KelurahanID); err != nil {
			writeErr(w, http.StatusInternalServerError, "gagal menyimpan lurah")
			return
		}
	}
	created, err := qtx.CreateLurah(r.Context(), db.CreateLurahParams{
		KelurahanID: in.KelurahanID, Nama: in.Nama, Nip: in.Nip, Aktif: boolToInt(in.Aktif),
	})
	if err != nil || tx.Commit() != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan lurah")
		return
	}
	in.ID = created.ID
	writeJSON(w, http.StatusCreated, in)
}

// UpdateLurah: PUT /api/lurah/{id}
func (h *Handler) UpdateLurah(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	var in lurahView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.q.GetLurah(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "lurah tidak ditemukan")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}

	tx, err := h.sqldb.BeginTx(r.Context(), nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	if in.Aktif {
		if err := qtx.DeactivateLurahByKelurahan(r.Context(), in.KelurahanID); err != nil {
			writeErr(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
			return
		}
	}
	if err := qtx.UpdateLurah(r.Context(), db.UpdateLurahParams{
		KelurahanID: in.KelurahanID, Nama: in.Nama, Nip: in.Nip,
		Aktif: boolToInt(in.Aktif), ID: id,
	}); err != nil || tx.Commit() != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
		return
	}
	in.ID = id
	writeJSON(w, http.StatusOK, in)
}

// DeleteLurah: DELETE /api/lurah/{id}
func (h *Handler) DeleteLurah(w http.ResponseWriter, r *http.Request) {
	h.deleteRow(w, r, "lurah", h.q.DeleteLurah)
}

// ===== KETUA RT =====

type ketuaRTView struct {
	ID            int64  `json:"id"`
	KelurahanID   int64  `json:"kelurahan_id"`
	KelurahanNama string `json:"kelurahan_nama,omitempty"`
	NomorRt       string `json:"nomor_rt"`
	Nama          string `json:"nama"`
	Aktif         bool   `json:"aktif"`
}

func (in *ketuaRTView) validate() error {
	in.NomorRt = strings.TrimSpace(in.NomorRt)
	in.Nama = strings.TrimSpace(in.Nama)
	if in.KelurahanID == 0 {
		return errors.New("kelurahan wajib dipilih")
	}
	if in.NomorRt == "" {
		return errors.New("nomor RT wajib diisi")
	}
	if in.Nama == "" {
		return errors.New("nama wajib diisi")
	}
	return nil
}

// ListKetuaRT: GET /api/ketua-rt
func (h *Handler) ListKetuaRT(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListKetuaRT(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat ketua RT")
		return
	}
	out := make([]ketuaRTView, 0, len(rows))
	for _, k := range rows {
		out = append(out, ketuaRTView{
			ID: k.ID, KelurahanID: k.KelurahanID, KelurahanNama: k.KelurahanNama,
			NomorRt: k.NomorRt, Nama: k.Nama, Aktif: k.Aktif != 0,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateKetuaRT: POST /api/ketua-rt
func (h *Handler) CreateKetuaRT(w http.ResponseWriter, r *http.Request) {
	var in ketuaRTView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	created, err := h.q.CreateKetuaRT(r.Context(), db.CreateKetuaRTParams{
		KelurahanID: in.KelurahanID, NomorRt: in.NomorRt, Nama: in.Nama, Aktif: boolToInt(in.Aktif),
	})
	if isConstraintErr(err) {
		writeErr(w, http.StatusBadRequest, "Ketua RT untuk nomor RT itu sudah ada di kelurahan tersebut")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan ketua RT")
		return
	}
	in.ID = created.ID
	writeJSON(w, http.StatusCreated, in)
}

// UpdateKetuaRT: PUT /api/ketua-rt/{id}
func (h *Handler) UpdateKetuaRT(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	var in ketuaRTView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.q.GetKetuaRT(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "ketua RT tidak ditemukan")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if err := h.q.UpdateKetuaRT(r.Context(), db.UpdateKetuaRTParams{
		KelurahanID: in.KelurahanID, NomorRt: in.NomorRt, Nama: in.Nama,
		Aktif: boolToInt(in.Aktif), ID: id,
	}); err != nil {
		if isConstraintErr(err) {
			writeErr(w, http.StatusBadRequest, "Ketua RT untuk nomor RT itu sudah ada di kelurahan tersebut")
			return
		}
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
		return
	}
	in.ID = id
	writeJSON(w, http.StatusOK, in)
}

// DeleteKetuaRT: DELETE /api/ketua-rt/{id}
func (h *Handler) DeleteKetuaRT(w http.ResponseWriter, r *http.Request) {
	h.deleteRow(w, r, "ketua RT", h.q.DeleteKetuaRT)
}

// ===== JURU UKUR =====

type juruUkurView struct {
	ID            int64  `json:"id"`
	Tingkat       string `json:"tingkat"`
	KelurahanID   *int64 `json:"kelurahan_id"`
	KelurahanNama string `json:"kelurahan_nama,omitempty"`
	Nama          string `json:"nama"`
	Nip           string `json:"nip"`
	Aktif         bool   `json:"aktif"`
}

func (in *juruUkurView) validate() error {
	in.Tingkat = strings.ToLower(strings.TrimSpace(in.Tingkat))
	in.Nama = strings.TrimSpace(in.Nama)
	in.Nip = strings.TrimSpace(in.Nip)
	if in.Tingkat != "kelurahan" && in.Tingkat != "kecamatan" {
		return errors.New("tingkat harus 'kelurahan' atau 'kecamatan'")
	}
	if in.Tingkat == "kelurahan" && (in.KelurahanID == nil || *in.KelurahanID == 0) {
		return errors.New("kelurahan wajib dipilih untuk juru ukur tingkat kelurahan")
	}
	if in.Tingkat == "kecamatan" {
		in.KelurahanID = nil
	}
	if in.Nama == "" {
		return errors.New("nama wajib diisi")
	}
	if in.Nip == "" {
		return errors.New("NIP wajib diisi")
	}
	return nil
}

func nullIntPtr(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

// ListJuruUkur: GET /api/juru-ukur
func (h *Handler) ListJuruUkur(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListJuruUkur(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat juru ukur")
		return
	}
	out := make([]juruUkurView, 0, len(rows))
	for _, j := range rows {
		v := juruUkurView{
			ID: j.ID, Tingkat: j.Tingkat, KelurahanNama: strOrEmpty(j.KelurahanNama),
			Nama: j.Nama, Nip: j.Nip, Aktif: j.Aktif != 0,
		}
		if j.KelurahanID.Valid {
			v.KelurahanID = &j.KelurahanID.Int64
		}
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, out)
}

// CreateJuruUkur: POST /api/juru-ukur
func (h *Handler) CreateJuruUkur(w http.ResponseWriter, r *http.Request) {
	var in juruUkurView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	created, err := h.q.CreateJuruUkur(r.Context(), db.CreateJuruUkurParams{
		Tingkat: in.Tingkat, KelurahanID: nullIntPtr(in.KelurahanID),
		Nama: in.Nama, Nip: in.Nip, Aktif: boolToInt(in.Aktif),
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan juru ukur")
		return
	}
	in.ID = created.ID
	writeJSON(w, http.StatusCreated, in)
}

// UpdateJuruUkur: PUT /api/juru-ukur/{id}
func (h *Handler) UpdateJuruUkur(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	var in juruUkurView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.q.GetJuruUkur(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "juru ukur tidak ditemukan")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if err := h.q.UpdateJuruUkur(r.Context(), db.UpdateJuruUkurParams{
		Tingkat: in.Tingkat, KelurahanID: nullIntPtr(in.KelurahanID),
		Nama: in.Nama, Nip: in.Nip, Aktif: boolToInt(in.Aktif), ID: id,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
		return
	}
	in.ID = id
	writeJSON(w, http.StatusOK, in)
}

// DeleteJuruUkur: DELETE /api/juru-ukur/{id}
func (h *Handler) DeleteJuruUkur(w http.ResponseWriter, r *http.Request) {
	h.deleteRow(w, r, "juru ukur", h.q.DeleteJuruUkur)
}
