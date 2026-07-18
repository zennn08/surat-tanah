package surat

import (
	"strconv"
	"strings"
)

// Gelar/sapaan yang dibuang dari AWAL nama saat normalisasi (SPEC 7.1).
// ponytail: daftar dari spec + pasangan wajarnya; tambah di sini bila kantor menemukan varian lain.
var gelarDepan = map[string]bool{
	"h": true, "hj": true, "haji": true, "hajjah": true,
	"bpk": true, "bapak": true, "ibu": true,
	"tn": true, "ny": true,
	"alm": true, "almarhum": true, "almh": true, "almarhumah": true,
	"sdr": true, "sdri": true,
}

// tokens memecah string jadi kata: huruf kecil, semua non-alfanumerik jadi spasi.
// Lebih ketat dari spec (bukan cuma titik/koma/apostrof) — aman untuk pencocokan
// dan menjamin dup_key bebas karakter wildcard/pemisah.
func tokens(s string) []string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}

// NormalisasiNama menormalkan nama pemilik batas untuk deteksi duplikat (SPEC 7.1):
// huruf kecil, buang gelar/sapaan di awal, buang tanda baca, rapatkan spasi.
// Batas juga bisa berupa jalan ("Jalan Harapan" vs "Jln Harapan"), jadi prefiks
// jalan ikut dibuang agar variannya cocok.
// "H. Ahmad Yani" / "HAJI AHMAD YANI" / "h.ahmad  yani" → "ahmad yani".
func NormalisasiNama(s string) string {
	t := tokens(s)
	for len(t) > 0 && (gelarDepan[t[0]] || prefiksJalan[t[0]]) {
		t = t[1:]
	}
	return strings.Join(t, " ")
}

// prefiksJalan yang dibuang dari awal nama jalan ("Jl. Harapan" = "Jalan Harapan").
var prefiksJalan = map[string]bool{"jl": true, "jln": true, "jalan": true, "gg": true, "gang": true}

// NormalisasiJalan menormalkan nama jalan/gang untuk kunci duplikat.
func NormalisasiJalan(s string) string {
	t := tokens(s)
	for len(t) > 0 && prefiksJalan[t[0]] {
		t = t[1:]
	}
	return strings.Join(t, " ")
}

// NormalisasiRT menormalkan nomor RT: "RT 007" / "007" / "7" → "7".
func NormalisasiRT(s string) string {
	t := tokens(s)
	if len(t) > 0 && t[0] == "rt" {
		t = t[1:]
	}
	j := strings.Join(t, "")
	if trimmed := strings.TrimLeft(j, "0"); trimmed != "" {
		return trimmed
	}
	return j
}

// DupKey membentuk kunci duplikat (SPEC 7.1):
// kelurahan_id | jalan | rt | utara | selatan | barat | timur (semua ternormalisasi).
func DupKey(kelurahanID int64, jalan, rt, utara, selatan, barat, timur string) string {
	return strings.Join([]string{
		strconv.FormatInt(kelurahanID, 10),
		NormalisasiJalan(jalan),
		NormalisasiRT(rt),
		NormalisasiNama(utara),
		NormalisasiNama(selatan),
		NormalisasiNama(barat),
		NormalisasiNama(timur),
	}, "|")
}

// LokasiPrefix mengembalikan prefiks dup_key untuk pencarian "lokasi sama"
// (kelurahan + jalan + RT), diakhiri "|".
func LokasiPrefix(kelurahanID int64, jalan, rt string) string {
	return strconv.FormatInt(kelurahanID, 10) + "|" + NormalisasiJalan(jalan) + "|" + NormalisasiRT(rt) + "|"
}

// BatasDariDupKey mengurai 4 nama batas ternormalisasi dari dup_key
// (urutan: utara, selatan, barat, timur).
func BatasDariDupKey(dupKey string) [4]string {
	parts := strings.SplitN(dupKey, "|", 7)
	var out [4]string
	for i := 0; i < 4 && 3+i < len(parts); i++ {
		out[i] = parts[3+i]
	}
	return out
}
