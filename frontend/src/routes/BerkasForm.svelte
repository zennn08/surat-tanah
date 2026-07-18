<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { notify } from '../lib/stores.js'
  import { navigate } from '../lib/router.js'
  import { digitsOnly, fmtDate } from '../lib/format.js'
  import DupPanel from './DupPanel.svelte'

  export let id = null // null = buat baru, terisi = edit

  const arah = ['utara', 'selatan', 'barat', 'timur']
  const steps = [
    { n: 1, label: 'Pemohon' },
    { n: 2, label: 'Lokasi & Batas' },
    { n: 3, label: 'Pengukuran' },
    { n: 4, label: 'Saksi & Tanggal' },
    { n: 5, label: 'Periksa & Simpan' },
  ]
  let step = 1
  let maxStep = 1 // langkah terjauh yang sudah pernah dicapai

  let kelurahan = []
  let ketuaRT = []
  let juruUkur = []
  let loading = true
  let busy = false

  let f = {
    tanggal_surat: new Date().toISOString().slice(0, 10),
    p1_nama: '', p1_umur: '', p1_nik: '', p1_pekerjaan: '', p1_alamat: '',
    p2_nama: '', p2_alamat: '', luas_ganti_rugi_m2: '',
    kelurahan_id: 0, jalan_gang: '', ketua_rt_id: 0, luas_m2: '',
    tgl_pengukuran: '', juru_ukur_kel_id: 0, juru_ukur_kec_id: 0,
    batas: { utara: { nama: '', panjang_meter: '' }, selatan: { nama: '', panjang_meter: '' }, barat: { nama: '', panjang_meter: '' }, timur: { nama: '', panjang_meter: '' } },
    saksi: ['', '', '', ''],
    induk_id: null, // pecah bidang: id berkas induk
    arsip_lama: false, // salinan surat lama (arsip), bukan surat baru
  }

  // Info berkas induk untuk peringatan luas (dimuat saat induk ditandai).
  let indukInfo = null // {nomor_berkas, luas_m2, total_luas_pecahan, pecahan: []}

  async function setInduk(match) {
    if (!match) { f.induk_id = null; indukInfo = null; return }
    f.induk_id = match.id
    try { indukInfo = await api.get('/api/berkas/' + match.id) }
    catch (e) { notify(e.message, 'error') }
  }

  // Sisa luas induk = luas induk - total pecahan lain (tidak menghitung berkas ini sendiri saat edit).
  $: sisaInduk = (() => {
    if (!indukInfo) return null
    let total = indukInfo.total_luas_pecahan || 0
    if (id) {
      const sendiri = (indukInfo.pecahan || []).find((p) => p.id === Number(id))
      if (sendiri) total -= sendiri.luas_m2
    }
    return indukInfo.luas_m2 - total
  })()
  $: luasMelebihi = sisaInduk != null && Number(f.luas_m2) > sisaInduk

  // Panel duplikat
  let dup = null
  let dupTimer
  let overrideAlasan = ''

  $: rtOptions = ketuaRT.filter((r) => r.kelurahan_id === Number(f.kelurahan_id) && r.aktif)
  $: jukelOptions = juruUkur.filter((j) => j.tingkat === 'kelurahan' && j.aktif && (!j.kelurahan_id || j.kelurahan_id === Number(f.kelurahan_id)))
  $: jukecOptions = juruUkur.filter((j) => j.tingkat === 'kecamatan' && j.aktif)
  $: rtNomor = (ketuaRT.find((r) => r.id === Number(f.ketua_rt_id)) || {}).nomor_rt || ''
  $: kelurahanNama = (kelurahan.find((k) => k.id === Number(f.kelurahan_id)) || {}).nama || ''
  $: jukelNama = (juruUkur.find((j) => j.id === Number(f.juru_ukur_kel_id)) || {}).nama || ''
  $: jukecNama = (juruUkur.find((j) => j.id === Number(f.juru_ukur_kec_id)) || {}).nama || ''

  // SPEC 7.1: cek duplikat begitu lokasi + keempat batas terisi.
  $: lengkap = Boolean(Number(f.kelurahan_id) && f.jalan_gang.trim() && rtNomor &&
    arah.every((a) => f.batas[a].nama.trim()))
  $: if (!loading) jadwalkanCek(lengkap, f.jalan_gang, rtNomor, f.kelurahan_id,
    f.batas.utara.nama, f.batas.selatan.nama, f.batas.barat.nama, f.batas.timur.nama)
  function jadwalkanCek(ok) {
    clearTimeout(dupTimer)
    if (!ok) { dup = null; return }
    dupTimer = setTimeout(cekDuplikat, 400)
  }

  async function cekDuplikat() {
    try {
      dup = await api.post('/api/berkas/cek-duplikat', {
        kelurahan_id: Number(f.kelurahan_id),
        jalan_gang: f.jalan_gang,
        rt: rtNomor,
        batas_utara: { nama: f.batas.utara.nama },
        batas_selatan: { nama: f.batas.selatan.nama },
        batas_barat: { nama: f.batas.barat.nama },
        batas_timur: { nama: f.batas.timur.nama },
        exclude_id: id ? Number(id) : 0,
      })
    } catch (e) { notify(e.message, 'error') }
  }

  // Validasi per langkah; kembalikan pesan kesalahan pertama, atau '' bila lolos.
  function periksaLangkah(n) {
    if (n === 1) {
      if (!f.p1_nama.trim()) return 'Nama pemohon belum diisi.'
      if (!f.p1_nik.trim()) return 'NIK pemohon belum diisi.'
    }
    if (n === 2) {
      if (!Number(f.kelurahan_id)) return 'Pilih kelurahan dulu.'
      if (!f.jalan_gang.trim()) return 'Nama jalan/gang belum diisi.'
      if (!Number(f.ketua_rt_id)) return 'Pilih RT dulu.'
      if (!Number(f.luas_m2)) return 'Luas tanah belum diisi.'
      for (const a of arah) {
        if (!f.batas[a].nama.trim()) return `Batas sebelah ${a} belum diisi.`
      }
    }
    if (n === 4) {
      for (let i = 0; i < 4; i++) {
        if (!f.saksi[i].trim()) return `Nama saksi ke-${i + 1} belum diisi.`
      }
      if (!f.tanggal_surat) return 'Tanggal surat belum diisi.'
    }
    return ''
  }

  function lanjut() {
    const salah = periksaLangkah(step)
    if (salah) { notify(salah, 'error'); return }
    step += 1
    maxStep = Math.max(maxStep, step)
    window.scrollTo({ top: 0 })
  }
  function kembali() { step -= 1; window.scrollTo({ top: 0 }) }
  function batal() {
    if (!confirm('Batalkan? Isian yang belum disimpan akan hilang.')) return
    navigate(id ? '/berkas/' + id : '/')
  }
  function keLangkah(n) {
    if (n > maxStep && !id) return
    // maju lewat klik tetap divalidasi langkah demi langkah saat buat baru
    step = n
    maxStep = Math.max(maxStep, n)
    window.scrollTo({ top: 0 })
  }

  function payload() {
    const num = (v) => (v === '' || v === null ? null : Number(v))
    const b = (a) => ({ nama: f.batas[a].nama, panjang_meter: num(f.batas[a].panjang_meter) })
    return {
      tanggal_surat: f.tanggal_surat,
      p1_nama: f.p1_nama, p1_umur: num(f.p1_umur), p1_nik: f.p1_nik,
      p1_pekerjaan: f.p1_pekerjaan, p1_alamat: f.p1_alamat,
      p2_nama: f.p2_nama, p2_alamat: f.p2_alamat, luas_ganti_rugi_m2: num(f.luas_ganti_rugi_m2),
      kelurahan_id: Number(f.kelurahan_id), jalan_gang: f.jalan_gang,
      ketua_rt_id: Number(f.ketua_rt_id), luas_m2: num(f.luas_m2),
      tgl_pengukuran: f.tgl_pengukuran,
      juru_ukur_kel_id: Number(f.juru_ukur_kel_id) || null,
      juru_ukur_kec_id: Number(f.juru_ukur_kec_id) || null,
      batas_utara: b('utara'), batas_selatan: b('selatan'), batas_barat: b('barat'), batas_timur: b('timur'),
      saksi: f.saksi,
      induk_id: f.induk_id ? Number(f.induk_id) : null,
      arsip_lama: f.arsip_lama,
      override_alasan: overrideAlasan,
    }
  }

  async function simpan() {
    for (const n of [1, 2, 4]) {
      const salah = periksaLangkah(n)
      if (salah) { notify(salah, 'error'); step = n; return }
    }
    if (dup && dup.tingkat === 'merah' && !overrideAlasan.trim()) {
      notify('Tanah ini terdeteksi sudah pernah dibuatkan surat. Isi alasan dulu untuk tetap menyimpan.', 'error')
      return
    }
    busy = true
    try {
      if (id) {
        await api.put('/api/berkas/' + id, payload())
        notify('Perubahan berkas disimpan', 'success')
        navigate('/berkas/' + id)
      } else {
        const res = await api.post('/api/berkas', payload())
        notify('Berkas ' + res.nomor_berkas + ' tersimpan', 'success')
        navigate('/berkas/' + res.id)
      }
    } catch (e) {
      if (e.status === 409) cekDuplikat()
      notify(e.message, 'error')
    } finally { busy = false }
  }

  // Isi seluruh form dengan data contoh — untuk uji coba / latihan.
  function isiContoh() {
    const kel = kelurahan.find((k) => k.aktif && ketuaRT.some((r) => r.kelurahan_id === k.id && r.aktif)) || kelurahan.find((k) => k.aktif)
    const kelId = kel ? kel.id : 0
    const rt = ketuaRT.find((r) => r.kelurahan_id === kelId && r.aktif)
    const jukel = juruUkur.find((j) => j.tingkat === 'kelurahan' && j.aktif && (!j.kelurahan_id || j.kelurahan_id === kelId))
    const jukec = juruUkur.find((j) => j.tingkat === 'kecamatan' && j.aktif)
    const hariIni = new Date().toISOString().slice(0, 10)
    f = {
      tanggal_surat: hariIni,
      p1_nama: 'H. Ahmad Yani', p1_umur: 52, p1_nik: '1471012345678901',
      p1_pekerjaan: 'Petani', p1_alamat: 'Jl. Harapan RT 007 ' + (kel ? kel.nama : ''),
      p2_nama: 'Muhammad Rizky', p2_alamat: 'Jl. Sudirman No. 12 Dumai', luas_ganti_rugi_m2: 1250,
      kelurahan_id: kelId, jalan_gang: 'Jl. Harapan', ketua_rt_id: rt ? rt.id : 0, luas_m2: 2000,
      tgl_pengukuran: hariIni, juru_ukur_kel_id: jukel ? jukel.id : 0, juru_ukur_kec_id: jukec ? jukec.id : 0,
      batas: {
        utara: { nama: 'Siti Aminah', panjang_meter: 40 },
        selatan: { nama: 'Budi Santoso', panjang_meter: 40 },
        barat: { nama: 'H. Karim', panjang_meter: 50 },
        timur: { nama: 'Jalan Harapan', panjang_meter: 50 },
      },
      saksi: ['Abdul Rahman', 'Joko Susilo', 'Mislan', 'Hendra Gunawan'],
    }
    maxStep = 5
    notify('Form terisi data contoh — silakan telusuri tiap langkah, lalu simpan bila mau.', 'success')
    if (!rt) notify('Ketua RT belum ada di Data Pejabat & Wilayah — RT masih kosong.', 'error')
  }

  onMount(async () => {
    try {
      ;[kelurahan, ketuaRT, juruUkur] = await Promise.all([
        api.get('/api/kelurahan'), api.get('/api/ketua-rt'), api.get('/api/juru-ukur'),
      ])
      if (id) {
        const d = await api.get('/api/berkas/' + id)
        f = {
          tanggal_surat: d.tanggal_surat,
          p1_nama: d.p1_nama, p1_umur: d.p1_umur ?? '', p1_nik: d.p1_nik,
          p1_pekerjaan: d.p1_pekerjaan, p1_alamat: d.p1_alamat,
          p2_nama: d.p2_nama, p2_alamat: d.p2_alamat, luas_ganti_rugi_m2: d.luas_ganti_rugi_m2 ?? '',
          kelurahan_id: d.kelurahan_id, jalan_gang: d.jalan_gang,
          ketua_rt_id: d.ketua_rt_id ?? 0, luas_m2: d.luas_m2,
          tgl_pengukuran: d.tgl_pengukuran, juru_ukur_kel_id: d.juru_ukur_kel_id ?? 0, juru_ukur_kec_id: d.juru_ukur_kec_id ?? 0,
          batas: Object.fromEntries(arah.map((a) => {
            const bt = d.batas.find((x) => x.arah === a) || {}
            return [a, { nama: bt.nama || '', panjang_meter: bt.panjang_meter ?? '' }]
          })),
          saksi: [...d.saksi, '', '', '', ''].slice(0, 4),
          induk_id: d.induk_id ?? null,
          arsip_lama: Boolean(d.arsip_lama),
        }
        overrideAlasan = d.override_alasan || ''
        maxStep = 5 // saat edit, semua langkah boleh dibuka langsung
        if (f.induk_id) {
          try { indukInfo = await api.get('/api/berkas/' + f.induk_id) } catch { /* info saja */ }
        }
      }
    } catch (e) { notify(e.message, 'error') } finally { loading = false }
  })
</script>

<h1>{id ? 'Ubah Berkas' : 'Buat Berkas Baru'}</h1>
<div class="page-sub">Isi data langkah demi langkah. Dari satu berkas, aplikasi membuat 3 surat siap cetak:
Surat Pernyataan Tidak Bersengketa, Berita Acara Pengukuran, dan Sceets Kaart.</div>

{#if loading}
  <div class="spinner">Memuat…</div>
{:else}

<div class="wiz-steps">
  {#each steps as s}
    <button type="button" class="wiz-step"
      class:active={step === s.n} class:done={step > s.n}
      class:clickable={s.n <= maxStep || id}
      on:click={() => keLangkah(s.n)}>
      <span class="num">{step > s.n ? '✓' : s.n}</span>
      <span class="lbl">{s.label}</span>
    </button>
  {/each}
</div>

<form on:submit|preventDefault={step === 5 ? simpan : lanjut}>

{#if step === 1}
  <div class="card">
    <div class="flex between" style="align-items:flex-start; gap:0.75rem;">
      <div>
        <div class="step-title">Siapa pemohonnya?</div>
        <div class="step-sub">Orang yang menggarap tanah dan namanya tercantum di ketiga surat (Pihak Pertama).</div>
      </div>
      {#if !id}
        <button type="button" class="btn btn-sm" on:click={isiContoh}
          title="Mengisi semua langkah dengan data contoh, untuk latihan atau uji coba">✨ Isi data contoh</button>
      {/if}
    </div>
    <div class="row row-2">
      <div class="field">
        <label for="p1nama">Nama lengkap</label>
        <input id="p1nama" bind:value={f.p1_nama} placeholder="sesuai KTP, boleh dengan gelar" required />
      </div>
      <div class="field">
        <label for="p1nik">NIK</label>
        <input id="p1nik" bind:value={f.p1_nik} use:digitsOnly class="mono" inputmode="numeric" maxlength="16" placeholder="16 angka di KTP" required />
      </div>
    </div>
    <div class="row row-3">
      <div class="field">
        <label for="p1umur">Umur</label>
        <input id="p1umur" type="number" min="1" max="130" bind:value={f.p1_umur} placeholder="tahun" />
      </div>
      <div class="field">
        <label for="p1kerja">Pekerjaan</label>
        <input id="p1kerja" bind:value={f.p1_pekerjaan} placeholder="mis. Petani" />
      </div>
      <div class="field">
        <label for="p1alamat">Alamat tempat tinggal</label>
        <input id="p1alamat" bind:value={f.p1_alamat} placeholder="alamat pemohon sekarang" />
      </div>
    </div>

    <div class="divider"></div>
    <label class="flex gap" style="align-items:flex-start; cursor:pointer;">
      <input type="checkbox" style="width:auto; margin-top:4px;" bind:checked={f.arsip_lama} />
      <span>
        <strong>Ini salinan surat lama (arsip)</strong><br>
        <span class="muted small">Centang bila data ini disalin dari surat yang sudah terbit sebelum aplikasi dipakai —
        misalnya surat induk yang mau dipecah. Isi tanggal surat sesuai tanggal di surat lamanya.
        Berkas akan diberi tanda "arsip lama" dan tidak perlu dicetak ulang.</span>
      </span>
    </label>
  </div>

{:else if step === 2}
  <div class="card">
    <div class="step-title">Di mana tanahnya?</div>
    <div class="step-sub">Lokasi dan batas-batas inilah yang dipakai aplikasi untuk memeriksa
      apakah tanah yang sama sudah pernah dibuatkan surat.</div>

    <div class="row row-2">
      <div class="field">
        <label for="kel">Kelurahan</label>
        <select id="kel" bind:value={f.kelurahan_id} on:change={() => { f.ketua_rt_id = 0; f.juru_ukur_kel_id = 0 }} required>
          <option value={0} disabled>— pilih kelurahan —</option>
          {#each kelurahan.filter((k) => k.aktif) as k}<option value={k.id}>{k.nama}</option>{/each}
        </select>
      </div>
      <div class="field">
        <label for="jalan">Jalan / gang</label>
        <input id="jalan" bind:value={f.jalan_gang} placeholder="mis. Jl. Harapan" required />
      </div>
    </div>
    <div class="row row-2">
      <div class="field">
        <label for="rt">RT</label>
        <select id="rt" bind:value={f.ketua_rt_id} required disabled={!Number(f.kelurahan_id)}>
          <option value={0} disabled>{Number(f.kelurahan_id) ? '— pilih RT —' : 'pilih kelurahan dulu'}</option>
          {#each rtOptions as r}<option value={r.id}>RT {r.nomor_rt} — Ketua: {r.nama}</option>{/each}
        </select>
        {#if Number(f.kelurahan_id) && rtOptions.length === 0}
          <div class="help">Belum ada Ketua RT untuk kelurahan ini.
            <a href="#/master">Tambahkan dulu di Data Pejabat &amp; Wilayah</a>.</div>
        {/if}
      </div>
      <div class="field">
        <label for="luas">Luas tanah (M²)</label>
        <input id="luas" type="number" step="any" min="0" bind:value={f.luas_m2} placeholder="mis. 2000" required />
      </div>
    </div>

    <div class="divider"></div>
    <h3>Batas-batas tanah</h3>
    <div class="section-sub">Tulis nama pemilik tanah (atau jalan/parit) di tiap sisi, seperti melihat denah dari atas.</div>

    <div class="kompas">
      {#each arah as a}
        <div class="k-sisi k-{a}">
          <div class="arah">{a}</div>
          <input bind:value={f.batas[a].nama} placeholder="berbatas dengan…" aria-label={'Nama batas sebelah ' + a} />
          <div class="meter">
            <input type="number" step="any" min="0" bind:value={f.batas[a].panjang_meter} aria-label={'Panjang batas ' + a} />
            <span>meter</span>
          </div>
        </div>
      {/each}
      <div class="k-tanah">
        <div>
          <div class="luas">{f.luas_m2 ? Number(f.luas_m2).toLocaleString('id-ID') + ' M²' : 'Tanah'}</div>
          <div class="ket">{f.p1_nama || 'milik pemohon'}</div>
        </div>
      </div>
    </div>

    <DupPanel {dup} {lengkap} indukId={f.induk_id} onInduk={setInduk} />

    {#if f.induk_id && indukInfo}
      <div class="notice mt-2">
        <strong>Pecahan dari berkas {indukInfo.nomor_berkas}</strong> ({indukInfo.p1_nama}) —
        luas induk {indukInfo.luas_m2.toLocaleString('id-ID')} M²,
        sisa yang belum dipecah ± {Math.max(0, sisaInduk).toLocaleString('id-ID')} M².
      </div>
      {#if luasMelebihi}
        <div class="alert alert-error mt-1">
          Luas yang diisi ({Number(f.luas_m2).toLocaleString('id-ID')} M²) <strong>melebihi sisa luas induk</strong>
          ({Math.max(0, sisaInduk).toLocaleString('id-ID')} M² dari {indukInfo.luas_m2.toLocaleString('id-ID')} M²).
          Periksa kembali angkanya — tetap boleh disimpan bila memang benar.
        </div>
      {/if}
    {/if}
  </div>

{:else if step === 3}
  <div class="card">
    <div class="step-title">Data pengukuran &amp; ganti rugi</div>
    <div class="step-sub">Dipakai di Berita Acara Pengukuran. Kalau belum ada, boleh dilewati dulu —
      bagian itu akan kosong di surat dan bisa diisi tangan.</div>

    <div class="row row-3">
      <div class="field">
        <label for="tglukur">Tanggal pengukuran</label>
        <input id="tglukur" type="date" bind:value={f.tgl_pengukuran} />
      </div>
      <div class="field">
        <label for="jukel">Juru ukur kelurahan</label>
        <select id="jukel" bind:value={f.juru_ukur_kel_id}>
          <option value={0}>— belum dipilih —</option>
          {#each jukelOptions as j}<option value={j.id}>{j.nama}</option>{/each}
        </select>
      </div>
      <div class="field">
        <label for="jukec">Juru ukur kecamatan</label>
        <select id="jukec" bind:value={f.juru_ukur_kec_id}>
          <option value={0}>— belum dipilih —</option>
          {#each jukecOptions as j}<option value={j.id}>{j.nama}</option>{/each}
        </select>
      </div>
    </div>

    <div class="divider"></div>
    <h3>Pihak Kedua (yang mengganti rugi)</h3>
    <div class="section-sub">Hanya muncul di Berita Acara. Kosongkan bila belum ada.</div>
    <div class="row row-3">
      <div class="field">
        <label for="p2nama">Nama</label>
        <input id="p2nama" bind:value={f.p2_nama} />
      </div>
      <div class="field">
        <label for="p2alamat">Alamat</label>
        <input id="p2alamat" bind:value={f.p2_alamat} />
      </div>
      <div class="field">
        <label for="lgr">Luas yang diganti rugi (m²)</label>
        <input id="lgr" type="number" step="any" min="0" bind:value={f.luas_ganti_rugi_m2} />
      </div>
    </div>
  </div>

{:else if step === 4}
  <div class="card">
    <div class="step-title">Saksi sempadan &amp; tanggal surat</div>
    <div class="step-sub">Empat orang saksi batas tanah — biasanya pemilik tanah sebelah. Nama mereka tercetak di ketiga surat.</div>
    <div class="row row-2">
      {#each f.saksi as _, i}
        <div class="field">
          <label for={'saksi' + i}>Saksi {i + 1}</label>
          <input id={'saksi' + i} bind:value={f.saksi[i]} required />
        </div>
      {/each}
    </div>
    <div class="divider"></div>
    <div class="row row-2">
      <div class="field">
        <label for="tglsurat">Tanggal surat</label>
        <input id="tglsurat" type="date" bind:value={f.tanggal_surat} required />
        <div class="help">Tanggal yang tercetak di blok tanda tangan ketiga surat.</div>
      </div>
    </div>
  </div>

{:else}
  <div class="card">
    <div class="step-title">Periksa dulu sebelum disimpan</div>
    <div class="step-sub">Pastikan semua benar — data ini akan tercetak di surat resmi.
      Klik nama langkah di atas untuk memperbaiki.</div>

    {#if f.arsip_lama}
      <div class="review-row"><div class="rv-label">Jenis berkas</div>
        <div class="rv-value"><span class="badge badge-gray">🗄 arsip surat lama</span> — salinan surat yang sudah terbit, bukan surat baru</div></div>
    {/if}
    <div class="review-row"><div class="rv-label">Pemohon</div>
      <div class="rv-value">{f.p1_nama}{f.p1_umur ? `, ${f.p1_umur} tahun` : ''} · NIK <span class="mono">{f.p1_nik}</span>
        {#if f.p1_pekerjaan}<br>{f.p1_pekerjaan}{/if}{#if f.p1_alamat} — {f.p1_alamat}{/if}</div></div>
    <div class="review-row"><div class="rv-label">Lokasi tanah</div>
      <div class="rv-value">{f.jalan_gang}, RT {rtNomor}, Kelurahan {kelurahanNama} · luas {Number(f.luas_m2 || 0).toLocaleString('id-ID')} M²</div></div>
    {#if f.induk_id && indukInfo}
      <div class="review-row"><div class="rv-label">Pecahan dari</div>
        <div class="rv-value">Berkas <span class="reg-chip">{indukInfo.nomor_berkas}</span> ({indukInfo.p1_nama}, {indukInfo.luas_m2.toLocaleString('id-ID')} M²)
          {#if luasMelebihi}<br><span style="color:var(--danger); font-weight:650;">⚠ luas melebihi sisa luas induk — periksa lagi</span>{/if}
        </div></div>
    {/if}
    <div class="review-row"><div class="rv-label">Batas-batas</div>
      <div class="rv-value">
        {#each arah as a}
          <div>Sebelah {a}: <strong>{f.batas[a].nama}</strong>{f.batas[a].panjang_meter ? ` (${f.batas[a].panjang_meter} meter)` : ''}</div>
        {/each}
      </div></div>
    <div class="review-row"><div class="rv-label">Pengukuran</div>
      <div class="rv-value">{f.tgl_pengukuran ? fmtDate(f.tgl_pengukuran) : 'tanggal belum diisi'} · Juru ukur: {jukelNama || '—'} (kelurahan), {jukecNama || '—'} (kecamatan)</div></div>
    <div class="review-row"><div class="rv-label">Pihak Kedua</div>
      <div class="rv-value">{f.p2_nama || '—'}{f.luas_ganti_rugi_m2 ? ` · ganti rugi ${Number(f.luas_ganti_rugi_m2).toLocaleString('id-ID')} m²` : ''}</div></div>
    <div class="review-row"><div class="rv-label">Saksi sempadan</div>
      <div class="rv-value">{f.saksi.filter(Boolean).join(', ')}</div></div>
    <div class="review-row"><div class="rv-label">Tanggal surat</div>
      <div class="rv-value">{fmtDate(f.tanggal_surat)}</div></div>

    <DupPanel {dup} {lengkap} indukId={f.induk_id} onInduk={setInduk} />

    {#if dup && dup.tingkat === 'merah'}
      <div class="alert alert-error mt-2">
        <strong>Tanah ini kemungkinan besar sudah pernah dibuatkan surat.</strong><br>
        Kalau memang harus tetap dibuat (mis. surat lama hilang), tulis alasannya di bawah.
        Alasan ini tercatat dan berkas akan diberi tanda ⚠.
        <div class="field mt-1 mb-0">
          <textarea rows="2" bind:value={overrideAlasan} placeholder="tulis alasan singkat di sini…"></textarea>
        </div>
      </div>
    {/if}
  </div>
{/if}

<div class="wiz-actions">
  <div class="flex gap">
    {#if step > 1}
      <button type="button" class="btn" on:click={kembali}>← Kembali</button>
    {/if}
    <button type="button" class="btn btn-ghost" on:click={batal}>Batal</button>
  </div>
  <div>
    {#if step < 5}
      <button class="btn btn-primary btn-lg">Lanjut →</button>
    {:else}
      <button class="btn btn-primary btn-lg" disabled={busy}>
        {busy ? 'Menyimpan…' : id ? 'Simpan perubahan' : 'Simpan berkas'}
      </button>
    {/if}
  </div>
</div>

</form>
{/if}
