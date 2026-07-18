package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"surat-tanah/internal/db"
)

type pengaturanView struct {
	Kecamatan        string `json:"kecamatan"`
	Kota             string `json:"kota"`
	KantorPertanahan string `json:"kantor_pertanahan"`
}

// GetPengaturan: GET /api/pengaturan
func (h *Handler) GetPengaturan(w http.ResponseWriter, r *http.Request) {
	p, err := h.q.GetPengaturan(r.Context())
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusOK, pengaturanView{})
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat pengaturan")
		return
	}
	writeJSON(w, http.StatusOK, pengaturanView{
		Kecamatan:        strOrEmpty(p.Kecamatan),
		Kota:             strOrEmpty(p.Kota),
		KantorPertanahan: strOrEmpty(p.KantorPertanahan),
	})
}

// UpdatePengaturan: PUT /api/pengaturan
func (h *Handler) UpdatePengaturan(w http.ResponseWriter, r *http.Request) {
	var in pengaturanView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	in.Kecamatan = strings.TrimSpace(in.Kecamatan)
	in.Kota = strings.TrimSpace(in.Kota)
	in.KantorPertanahan = strings.TrimSpace(in.KantorPertanahan)

	if err := h.q.UpdatePengaturan(r.Context(), db.UpdatePengaturanParams{
		Kecamatan:        nullStr(in.Kecamatan),
		Kota:             nullStr(in.Kota),
		KantorPertanahan: nullStr(in.KantorPertanahan),
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan pengaturan")
		return
	}
	writeJSON(w, http.StatusOK, in)
}
