package auth

import (
	"context"
	"encoding/json"
	"net/http"
)

type ctxKey int

const userIDKey ctxKey = 0

// RequireAuth adalah middleware chi: menolak request tanpa sesi valid (401 JSON)
// dan menyisipkan userID ke context untuk handler di bawahnya.
func (m *Manager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := m.UserID(r)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "sesi tidak valid, silakan login"})
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext mengambil userID yang disisipkan RequireAuth.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	uid, ok := ctx.Value(userIDKey).(int64)
	return uid, ok
}
