package surat

import "testing"

func TestNormalisasiNama(t *testing.T) {
	cases := map[string]string{
		"H. Ahmad Yani":   "ahmad yani", // contoh SPEC 7.1
		"HAJI AHMAD YANI": "ahmad yani",
		"h.ahmad  yani":   "ahmad yani",
		"Hj. Siti Aminah": "siti aminah",
		"Bpk H. Udin":     "udin", // gelar bertumpuk
		"Almarhum Karim":  "karim",
		"O'neil":          "o neil",
		"  Budi ":         "budi",
		"":                "",
		"H.":              "",        // hanya gelar
		"Jalan Harapan":   "harapan", // batas berupa jalan
		"Jln. Harapan":    "harapan",
	}
	for in, want := range cases {
		if got := NormalisasiNama(in); got != want {
			t.Errorf("NormalisasiNama(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalisasiJalanRT(t *testing.T) {
	if got := NormalisasiJalan("Jl. Harapan Raya"); got != "harapan raya" {
		t.Errorf("jalan: %q", got)
	}
	if got := NormalisasiJalan("JALAN HARAPAN  RAYA"); got != "harapan raya" {
		t.Errorf("jalan2: %q", got)
	}
	if got := NormalisasiRT("RT 007"); got != "7" {
		t.Errorf("rt: %q", got)
	}
	if got := NormalisasiRT("07"); got != "7" {
		t.Errorf("rt2: %q", got)
	}
}

func TestDupKey(t *testing.T) {
	a := DupKey(3, "Jl. Harapan", "007", "H. Ahmad Yani", "Siti", "Budi", "Karim")
	b := DupKey(3, "jalan harapan", "RT 7", "haji ahmad yani", "SITI", "budi.", "karim")
	if a != b {
		t.Errorf("dup_key harus sama:\n%s\n%s", a, b)
	}
	if a == DupKey(4, "Jl. Harapan", "007", "H. Ahmad Yani", "Siti", "Budi", "Karim") {
		t.Error("kelurahan beda harus beda dup_key")
	}
	// urutan arah tetap: menukar batas harus mengubah kunci
	if a == DupKey(3, "Jl. Harapan", "007", "Siti", "H. Ahmad Yani", "Budi", "Karim") {
		t.Error("urutan arah harus berpengaruh")
	}

	want := "3|harapan|7|ahmad yani|siti|budi|karim"
	if a != want {
		t.Errorf("dup_key = %q, want %q", a, want)
	}

	if p := LokasiPrefix(3, "Jl. Harapan", "007"); p != "3|harapan|7|" {
		t.Errorf("prefix = %q", p)
	}
	if batas := BatasDariDupKey(a); batas != [4]string{"ahmad yani", "siti", "budi", "karim"} {
		t.Errorf("batas dari dup_key = %v", batas)
	}
}

func TestTanggalIndonesia(t *testing.T) {
	hari, tgl, bulan, tahun, ok := TanggalIndonesia("2026-07-18")
	if !ok || hari != "Sabtu" || tgl != 18 || bulan != "Juli" || tahun != 2026 {
		t.Errorf("got %s %d %s %d ok=%v", hari, tgl, bulan, tahun, ok)
	}
	if _, _, _, _, ok := TanggalIndonesia("bukan-tanggal"); ok {
		t.Error("harus gagal untuk input salah")
	}
	if got := FormatTanggal("2026-07-18"); got != "18 Juli 2026" {
		t.Errorf("FormatTanggal = %q", got)
	}
}
