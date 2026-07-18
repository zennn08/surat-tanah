package surat

import "time"

var hariID = [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

var bulanID = [...]string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// TanggalIndonesia mengurai tanggal ISO (YYYY-MM-DD) menjadi komponen Bahasa
// Indonesia untuk Berita Acara: hari, tanggal, bulan, tahun. ok=false bila format salah.
func TanggalIndonesia(iso string) (hari string, tanggal int, bulan string, tahun int, ok bool) {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return "", 0, "", 0, false
	}
	return hariID[t.Weekday()], t.Day(), bulanID[t.Month()], t.Year(), true
}

// FormatTanggal memformat ISO → "18 Juli 2026" (untuk blok tanda tangan).
// Bila format salah, kembalikan apa adanya.
func FormatTanggal(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return time.Time(t).Format("2") + " " + bulanID[t.Month()] + " " + t.Format("2006")
}
