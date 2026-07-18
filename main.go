package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"surat-tanah/internal/auth"
	"surat-tanah/internal/db"
	"surat-tanah/internal/handler"
)

const (
	defaultPort = "8080"
	dbFileName  = "surat-tanah.db"
)

// version diisi saat build via -ldflags "-X main.version=v1.x.x" (lihat Makefile/CI).
var version = "dev"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	seedOnly := flag.Bool("seed", false, "jalankan seeder lalu keluar")
	seedContoh := flag.Bool("seed-contoh", false, "isi data contoh Lurah/Ketua RT/Juru Ukur (untuk latihan) lalu keluar")
	flag.Parse()

	sqldb, err := openDB()
	if err != nil {
		log.Fatalf("gagal membuka database: %v", err)
	}
	defer sqldb.Close()

	if err := db.Migrate(sqldb); err != nil {
		log.Fatalf("gagal migrasi: %v", err)
	}

	q := db.New(sqldb)

	if err := auth.Seed(context.Background(), q); err != nil {
		log.Fatalf("gagal seed: %v", err)
	}
	if *seedContoh {
		if err := auth.SeedContoh(context.Background(), q); err != nil {
			log.Fatalf("gagal seed contoh: %v", err)
		}
		log.Println("data contoh selesai diisi")
		return
	}
	if *seedOnly {
		log.Println("seeder selesai")
		return
	}

	mgr := auth.NewManager()
	r := newRouter(sqldb, q, mgr)

	ln, port, err := listen(defaultPort)
	if err != nil {
		log.Fatalf("gagal membuka listener: %v", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%s", port)
	srv := &http.Server{Handler: r}

	go func() {
		log.Printf("Surat Tanah berjalan di %s", url)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	time.Sleep(300 * time.Millisecond)
	openBrowser(url)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("server berhenti")
}

func newRouter(sqldb *sql.DB, q *db.Queries, mgr *auth.Manager) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	authH := auth.NewHandler(q, mgr)
	apiH := handler.New(sqldb, q, parseTemplates())

	// Publik
	r.Post("/api/login", authH.Login)
	r.Get("/api/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"version":%q}`, version)
	})
	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		if err := sqldb.PingContext(req.Context()); err != nil {
			http.Error(w, "db down", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// Butuh sesi
	r.Group(func(pr chi.Router) {
		pr.Use(mgr.RequireAuth)
		pr.Post("/api/logout", authH.Logout)
		pr.Get("/api/me", authH.Me)
		pr.Post("/api/change-password", authH.ChangePassword)

		// Master data — Kelurahan
		pr.Get("/api/kelurahan", apiH.ListKelurahan)
		pr.Post("/api/kelurahan", apiH.CreateKelurahan)
		pr.Put("/api/kelurahan/{id}", apiH.UpdateKelurahan)
		pr.Delete("/api/kelurahan/{id}", apiH.DeleteKelurahan)

		// Master data — Lurah
		pr.Get("/api/lurah", apiH.ListLurah)
		pr.Post("/api/lurah", apiH.CreateLurah)
		pr.Put("/api/lurah/{id}", apiH.UpdateLurah)
		pr.Delete("/api/lurah/{id}", apiH.DeleteLurah)

		// Master data — Ketua RT
		pr.Get("/api/ketua-rt", apiH.ListKetuaRT)
		pr.Post("/api/ketua-rt", apiH.CreateKetuaRT)
		pr.Put("/api/ketua-rt/{id}", apiH.UpdateKetuaRT)
		pr.Delete("/api/ketua-rt/{id}", apiH.DeleteKetuaRT)

		// Master data — Juru Ukur
		pr.Get("/api/juru-ukur", apiH.ListJuruUkur)
		pr.Post("/api/juru-ukur", apiH.CreateJuruUkur)
		pr.Put("/api/juru-ukur/{id}", apiH.UpdateJuruUkur)
		pr.Delete("/api/juru-ukur/{id}", apiH.DeleteJuruUkur)

		// Pengaturan
		pr.Get("/api/pengaturan", apiH.GetPengaturan)
		pr.Put("/api/pengaturan", apiH.UpdatePengaturan)

		// Berkas tanah (inti) + deteksi duplikat
		pr.Get("/api/berkas", apiH.ListBerkas)
		pr.Post("/api/berkas", apiH.CreateBerkas)
		pr.Post("/api/berkas/cek-duplikat", apiH.CekDuplikat)
		pr.Get("/api/berkas/{id}", apiH.GetBerkas)
		pr.Put("/api/berkas/{id}", apiH.UpdateBerkas)

		// Halaman cetak (html/template, A4) — dibuka di tab baru.
		pr.Get("/berkas/{id}/cetak", apiH.Cetak)
	})

	// UI Svelte (embed) — publik; auth ditangani di dalam SPA via /api/me.
	r.Handle("/*", spaHandler())

	return r
}

// openDB membuka/membuat surat-tanah.db di direktori exe (WAL, foreign keys).
func openDB() (*sql.DB, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(filepath.Dir(exe), dbFileName)
	sqldb, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	log.Printf("database siap: %s (WAL)", path)
	return sqldb, nil
}

// listen wajib bind 127.0.0.1 saja (SPEC Bagian 2) — jangan pernah 0.0.0.0.
func listen(preferred string) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:"+preferred)
	if err == nil {
		return ln, preferred, nil
	}
	log.Printf("port %s terpakai, mencari port bebas...", preferred)
	ln, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		ln.Close()
		return nil, "", err
	}
	return ln, port, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("tidak bisa membuka browser otomatis (%v). Buka manual: %s", err, url)
	}
}
