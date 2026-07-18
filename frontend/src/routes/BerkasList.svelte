<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { notify } from '../lib/stores.js'
  import { fmtDate } from '../lib/format.js'

  let items = []
  let loading = true
  let q = ''
  let timer

  // Checklist persiapan: tampil selama data pendukung belum lengkap.
  let siap = null // null = belum dicek; {lurah, rt, juru} = jumlah data aktif

  async function load() {
    loading = true
    try { items = await api.get('/api/berkas?q=' + encodeURIComponent(q)) }
    catch (e) { notify(e.message, 'error') } finally { loading = false }
  }

  async function cekPersiapan() {
    try {
      const [lurah, rt, juru] = await Promise.all([
        api.get('/api/lurah'), api.get('/api/ketua-rt'), api.get('/api/juru-ukur'),
      ])
      siap = {
        lurah: lurah.filter((x) => x.aktif).length,
        rt: rt.filter((x) => x.aktif).length,
        juru: juru.filter((x) => x.aktif).length,
      }
    } catch { /* biarkan; checklist tidak tampil */ }
  }

  function onSearch() {
    clearTimeout(timer)
    timer = setTimeout(load, 300)
  }

  $: perluPersiapan = siap && (siap.lurah === 0 || siap.rt === 0 || siap.juru === 0)

  onMount(() => { load(); cekPersiapan() })
</script>

<div class="flex between" style="align-items:center; flex-wrap:wrap; gap:0.75rem;">
  <div>
    <h1 class="mb-0">Daftar Berkas</h1>
    <div class="muted small">Satu berkas = satu bidang tanah = 3 surat siap cetak.</div>
  </div>
  <a class="btn btn-primary btn-lg" href="#/berkas/baru">+ Buat Berkas Baru</a>
</div>

{#if perluPersiapan}
  <div class="card todo-card mt-2">
    <h3 style="margin-bottom:0.25rem;">Sebelum membuat berkas pertama</h3>
    <div class="section-sub">Isi dulu nama-nama pejabat berikut supaya bisa dipilih saat mengisi berkas.
      Cukup sekali; nanti tinggal pakai.</div>
    <div class="todo-item" class:beres={siap.lurah > 0}>
      <span class="tik">{siap.lurah > 0 ? '✓' : '1'}</span>
      <span class="grow"><strong>Lurah</strong> — penandatangan surat, satu per kelurahan</span>
      {#if siap.lurah === 0}<a class="btn btn-sm" href="#/master">Isi sekarang</a>{/if}
    </div>
    <div class="todo-item" class:beres={siap.rt > 0}>
      <span class="tik">{siap.rt > 0 ? '✓' : '2'}</span>
      <span class="grow"><strong>Ketua RT</strong> — ikut menandatangani, sesuai RT lokasi tanah</span>
      {#if siap.rt === 0}<a class="btn btn-sm" href="#/master">Isi sekarang</a>{/if}
    </div>
    <div class="todo-item" class:beres={siap.juru > 0}>
      <span class="tik">{siap.juru > 0 ? '✓' : '3'}</span>
      <span class="grow"><strong>Juru Ukur</strong> — petugas pengukuran, untuk Berita Acara</span>
      {#if siap.juru === 0}<a class="btn btn-sm" href="#/master">Isi sekarang</a>{/if}
    </div>
  </div>
{/if}

<div class="card mt-2">
  <div class="field mb-0">
    <label for="cari">Cari berkas</label>
    <input id="cari" placeholder="ketik nama pemohon, NIK, jalan, atau nama pemilik tanah sebelah…"
      bind:value={q} on:input={onSearch} />
    <div class="help">Tips: ketik nama tetangga (pemilik tanah sebelah) untuk mengecek apakah tanah di sekitarnya sudah pernah dibuatkan surat.</div>
  </div>
</div>

<div class="card mt-2">
  {#if loading}
    <div class="spinner">Memuat…</div>
  {:else if items.length === 0}
    <div class="empty">
      {#if q}
        Tidak ada berkas yang cocok dengan "{q}".
      {:else}
        Belum ada berkas tersimpan.<br>
        Klik <strong>+ Buat Berkas Baru</strong> di kanan atas untuk membuat yang pertama.
      {/if}
    </div>
  {:else}
    <div class="table-wrap">
    <table>
      <thead><tr><th>No. berkas</th><th>Tanggal</th><th>Pemohon</th><th>Lokasi tanah</th><th>Luas</th><th></th></tr></thead>
      <tbody>
        {#each items as b}
          <tr style="cursor:pointer;" on:click={() => (location.hash = '#/berkas/' + b.id)}
              title="Klik untuk membuka berkas">
            <td><span class="reg-chip">{b.nomor_berkas}</span></td>
            <td>{fmtDate(b.tanggal_surat)}</td>
            <td><strong>{b.p1_nama}</strong></td>
            <td>{b.jalan_gang}, RT {b.rt}<br><span class="muted small">{b.kelurahan_nama}</span></td>
            <td>{b.luas_m2.toLocaleString('id-ID')} M²</td>
            <td>
              {#if b.arsip_lama}<span class="badge badge-gray" title="Salinan surat lama yang diinput ke aplikasi sebagai arsip">🗄 arsip lama</span>{/if}
              {#if b.override}<span class="badge" style="background:#fdecea;color:#c62828;" title="Disimpan melewati peringatan duplikat — periksa alasannya di halaman berkas">⚠ perlu perhatian</span>{/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
    </div>
  {/if}
</div>
