package handler

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"surat-tanah/internal/db"
)

// Handler menampung dependency untuk semua HTTP handler API + halaman cetak.
type Handler struct {
	sqldb *sql.DB
	q     *db.Queries
	tmpl  *template.Template // template cetak (html/template)
}

func New(sqldb *sql.DB, q *db.Queries, tmpl *template.Template) *Handler {
	return &Handler{sqldb: sqldb, q: q, tmpl: tmpl}
}

// TemplateFuncs dipakai template cetak.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func idParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// nullStr membungkus string ke sql.NullString (kosong = NULL).
func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// strOrEmpty membuka sql.NullString menjadi string biasa untuk JSON.
func strOrEmpty(n sql.NullString) string {
	if n.Valid {
		return n.String
	}
	return ""
}

// isConstraintErr mendeteksi pelanggaran constraint SQLite (FK/unique).
func isConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "constraint failed")
}
