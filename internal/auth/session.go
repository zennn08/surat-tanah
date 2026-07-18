package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	cookieName     = "st_session"
	sessionTTL     = 12 * time.Hour
	tokenNumBytes  = 32
)

type sessionEntry struct {
	userID    int64
	expiresAt time.Time
}

// Manager menyimpan sesi di memori. Cocok untuk aplikasi desktop lokal
// single-process: restart exe = semua sesi hilang (user login ulang).
type Manager struct {
	mu       sync.Mutex
	sessions map[string]sessionEntry
	secure   bool // set true jika dilayani via HTTPS
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[string]sessionEntry)}
}

func newToken() (string, error) {
	b := make([]byte, tokenNumBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create membuat sesi baru untuk userID dan menuliskan cookie ke response.
func (m *Manager) Create(w http.ResponseWriter, userID int64) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	exp := time.Now().Add(sessionTTL)

	m.mu.Lock()
	m.sessions[token] = sessionEntry{userID: userID, expiresAt: exp}
	m.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
		Expires:  exp,
	})
	return nil
}

// UserID mengembalikan userID dari cookie sesi request, atau (0,false) jika
// tidak ada / kedaluwarsa. Sesi valid diperpanjang (sliding expiration).
func (m *Manager) UserID(r *http.Request) (int64, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return 0, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.sessions[c.Value]
	if !ok {
		return 0, false
	}
	if time.Now().After(e.expiresAt) {
		delete(m.sessions, c.Value)
		return 0, false
	}
	// perpanjang
	e.expiresAt = time.Now().Add(sessionTTL)
	m.sessions[c.Value] = e
	return e.userID, true
}

// Destroy menghapus sesi request dan menghapus cookie di browser.
func (m *Manager) Destroy(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		m.mu.Lock()
		delete(m.sessions, c.Value)
		m.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
		MaxAge:   -1,
	})
}
