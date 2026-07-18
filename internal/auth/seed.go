package auth

import (
	"context"
	"database/sql"
	"log"

	"surat-tanah/internal/db"
)

// Default kredensial admin awal. SPEC §9: wajib ganti saat login pertama.
const (
	defaultAdminUsername = "admin"
	defaultAdminPassword = "admin123"
	defaultAdminNama     = "Admin Kecamatan"
	defaultAdminRole     = "admin"
)

// Kelurahan se-Kecamatan Dumai Timur (dikonfirmasi user).
var seedKelurahan = []string{
	"Teluk Binjai",
	"Tanjung Palas",
	"Jaya Mukti",
	"Bukit Batrem",
	"Buluh Kasap",
}

// Seed idempotent: pengaturan (id=1), admin default bila belum ada user,
// daftar kelurahan bila tabel kosong. Aman dipanggil tiap start.
func Seed(ctx context.Context, q *db.Queries) error {
	if err := q.EnsurePengaturanRow(ctx); err != nil {
		return err
	}

	n, err := q.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		hash, err := HashPassword(defaultAdminPassword)
		if err != nil {
			return err
		}
		if _, err := q.CreateUser(ctx, db.CreateUserParams{
			Username:           defaultAdminUsername,
			PasswordHash:       hash,
			Nama:               defaultAdminNama,
			Role:               defaultAdminRole,
			MustChangePassword: 1,
		}); err != nil {
			return err
		}
		log.Printf("seeder: user admin default dibuat (username=%q, password=%q) — WAJIB ganti saat login pertama",
			defaultAdminUsername, defaultAdminPassword)
	}

	nk, err := q.CountKelurahan(ctx)
	if err != nil {
		return err
	}
	if nk == 0 {
		for _, nama := range seedKelurahan {
			if _, err := q.CreateKelurahan(ctx, nama); err != nil {
				return err
			}
		}
		log.Printf("seeder: %d kelurahan Dumai Timur ditambahkan", len(seedKelurahan))
	}
	return nil
}

// SeedContoh mengisi nama CONTOH Lurah/Ketua RT/Juru Ukur untuk latihan/uji coba.
// Hanya jalan bila dipanggil eksplisit (flag --seed-contoh) dan hanya mengisi
// tabel yang masih kosong. Jangan dipakai di kantor — isi nama asli lewat menu.
func SeedContoh(ctx context.Context, q *db.Queries) error {
	kel, err := q.ListKelurahan(ctx)
	if err != nil {
		return err
	}
	idKel := map[string]int64{}
	for _, k := range kel {
		idKel[k.Nama] = k.ID
	}

	if rows, err := q.ListLurah(ctx); err != nil {
		return err
	} else if len(rows) == 0 {
		contoh := []struct{ kel, nama, nip string }{
			{"Teluk Binjai", "Syamsul Bahri, S.Sos", "197203121994031005"},
			{"Tanjung Palas", "Hendra Saputra, S.STP", "198106152000121002"},
			{"Jaya Mukti", "Drs. Bambang Setiawan", "197001011990031001"},
			{"Bukit Batrem", "Rosmawati, S.AP", "197805202005022003"},
			{"Buluh Kasap", "M. Nasir, S.Sos", "196909091992031007"},
		}
		for _, c := range contoh {
			if idKel[c.kel] == 0 {
				continue
			}
			if _, err := q.CreateLurah(ctx, db.CreateLurahParams{
				KelurahanID: idKel[c.kel], Nama: c.nama, Nip: c.nip, Aktif: 1,
			}); err != nil {
				return err
			}
		}
		log.Println("seeder: lurah contoh ditambahkan (data sementara — ganti dengan nama asli)")
	}

	if rows, err := q.ListKetuaRT(ctx); err != nil {
		return err
	} else if len(rows) == 0 {
		contoh := []struct{ kel, rt, nama string }{
			{"Teluk Binjai", "001", "Ramli"}, {"Teluk Binjai", "005", "Darwis"},
			{"Tanjung Palas", "003", "Salim"}, {"Tanjung Palas", "008", "Yusuf Efendi"},
			{"Jaya Mukti", "007", "Sutrisno"}, {"Jaya Mukti", "012", "Amiruddin"},
			{"Bukit Batrem", "002", "Herman"}, {"Bukit Batrem", "009", "Tugiman"},
			{"Buluh Kasap", "004", "Zulkifli"}, {"Buluh Kasap", "006", "Rustam"},
		}
		for _, c := range contoh {
			if idKel[c.kel] == 0 {
				continue
			}
			if _, err := q.CreateKetuaRT(ctx, db.CreateKetuaRTParams{
				KelurahanID: idKel[c.kel], NomorRt: c.rt, Nama: c.nama, Aktif: 1,
			}); err != nil {
				return err
			}
		}
		log.Println("seeder: ketua RT contoh ditambahkan (data sementara — ganti dengan nama asli)")
	}

	if rows, err := q.ListJuruUkur(ctx); err != nil {
		return err
	} else if len(rows) == 0 {
		for _, c := range []struct{ kel, nama, nip string }{
			{"Teluk Binjai", "Andi Prasetyo, A.Md", "198804102012121001"},
			{"Tanjung Palas", "Rio Fernanda, S.T.", "199001222015031002"},
			{"Jaya Mukti", "Sugiharto, S.T.", "198505052010011003"},
			{"Bukit Batrem", "Wahyudi, A.Md", "198711302014021004"},
			{"Buluh Kasap", "Taufik Hidayat, S.T.", "199203152017051005"},
		} {
			if idKel[c.kel] == 0 {
				continue
			}
			if _, err := q.CreateJuruUkur(ctx, db.CreateJuruUkurParams{
				Tingkat: "kelurahan", KelurahanID: sql.NullInt64{Int64: idKel[c.kel], Valid: true},
				Nama: c.nama, Nip: c.nip, Aktif: 1,
			}); err != nil {
				return err
			}
		}
		for _, c := range []struct{ nama, nip string }{
			{"Ir. Dedi Kurniawan", "198001012000121001"},
			{"Rahmat Hidayat, A.Md", "198906172013101006"},
		} {
			if _, err := q.CreateJuruUkur(ctx, db.CreateJuruUkurParams{
				Tingkat: "kecamatan", Nama: c.nama, Nip: c.nip, Aktif: 1,
			}); err != nil {
				return err
			}
		}
		log.Println("seeder: juru ukur contoh ditambahkan (data sementara — ganti dengan nama asli)")
	}
	return nil
}
