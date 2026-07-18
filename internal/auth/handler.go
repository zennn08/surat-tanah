package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"surat-tanah/internal/db"
)

// Handler menyediakan endpoint auth: login, logout, me, change-password.
type Handler struct {
	q   *db.Queries
	mgr *Manager
}

func NewHandler(q *db.Queries, mgr *Manager) *Handler {
	return &Handler{q: q, mgr: mgr}
}

type userView struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	Nama               string `json:"nama"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"`
}

func toUserView(u db.User) userView {
	return userView{
		ID:                 u.ID,
		Username:           u.Username,
		Nama:               u.Nama,
		Role:               u.Role,
		MustChangePassword: u.MustChangePassword != 0,
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

// Login: POST /api/login  {username, password}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeErr(w, http.StatusBadRequest, "username dan password wajib diisi")
		return
	}

	u, err := h.q.GetUserByUsername(r.Context(), req.Username)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusUnauthorized, "username atau password salah")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if !CheckPassword(u.PasswordHash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "username atau password salah")
		return
	}
	if err := h.mgr.Create(w, u.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal membuat sesi")
		return
	}
	writeJSON(w, http.StatusOK, toUserView(u))
}

// Logout: POST /api/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.mgr.Destroy(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Me: GET /api/me  (butuh sesi)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "sesi tidak valid")
		return
	}
	u, err := h.q.GetUserByID(r.Context(), uid)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "sesi tidak valid")
		return
	}
	writeJSON(w, http.StatusOK, toUserView(u))
}

// ChangePassword: POST /api/change-password  {old_password, new_password}
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "sesi tidak valid")
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if len(req.NewPassword) < 6 {
		writeErr(w, http.StatusBadRequest, "password baru minimal 6 karakter")
		return
	}

	u, err := h.q.GetUserByID(r.Context(), uid)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "sesi tidak valid")
		return
	}
	if !CheckPassword(u.PasswordHash, req.OldPassword) {
		writeErr(w, http.StatusBadRequest, "password lama salah")
		return
	}
	hash, err := HashPassword(req.NewPassword)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memproses password")
		return
	}
	if err := h.q.UpdateUserPassword(r.Context(), db.UpdateUserPasswordParams{
		PasswordHash: hash,
		ID:           uid,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan password")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
