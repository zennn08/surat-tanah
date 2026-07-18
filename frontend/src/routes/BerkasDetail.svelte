<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { notify } from '../lib/stores.js'
  import { fmtDate } from '../lib/format.js'

  export let id

  let d = null

  onMount(async () => {
    try { d = await api.get('/api/berkas/' + id) }
    catch (e) { notify(e.message, 'error') }
  })

  const cap = (s) => (s ? s.charAt(0).toUpperCase() + s.slice(1) : s)
  $: batasMap = d ? Object.fromEntries(d.batas.map((b) => [b.arah, b])) : {}
</script>

{#if !d}
  <div class="spinner">Memuat…</div>
{:else}
  <div style="margin-bottom:0.5rem;">
    <a class="btn btn-sm btn-ghost" href="#/">← Kembali ke Daftar Berkas</a>
  </div>
  <div class="flex between" style="align-items:center; flex-wrap:wrap; gap:0.75rem;">
    <div>
      <h1 class="mb-0">Berkas <span class="reg-chip" style="font-size:1.1rem;">{d.nomor_berkas}</span>
        {#if d.arsip_lama}<span class="badge badge-gray" style="vertical-align:middle;">🗄 arsip surat lama</span>{/if}
      </h1>
      <div class="muted small">Nomor arsip aplikasi — tidak ikut tercetak di surat.
        {#if d.arsip_lama}Berkas ini salinan surat yang sudah terbit sebelum aplikasi dipakai.{/if}</div>
    </div>
    <div class="flex gap">
      <a class="btn" href={'#/berkas/' + d.id + '/edit'}>✎ Ubah data</a>
      <a class="btn btn-primary btn-lg" href={'/berkas/' + d.id + '/cetak'} target="_blank" rel="noreferrer">🖨 Cetak 3 Surat</a>
    </div>
  </div>

  {#if d.override_alasan}
    <div class="alert alert-warn mt-2">
      ⚠ Berkas ini disimpan <strong>walaupun terdeteksi sama dengan berkas lain</strong>.<br>
      Alasan petugas: “{d.override_alasan}”
    </div>
  {/if}

  <div class="card mt-2">
    <div class="card-title">
      <h3 class="mb-0">Pemohon (Pihak Pertama)</h3>
    </div>
    <div class="review-row"><div class="rv-label">Nama</div>
      <div class="rv-value"><strong>{d.p1_nama}</strong>{d.p1_umur ? `, ${d.p1_umur} tahun` : ''}</div></div>
    <div class="review-row"><div class="rv-label">NIK</div>
      <div class="rv-value mono">{d.p1_nik}</div></div>
    <div class="review-row"><div class="rv-label">Pekerjaan &amp; alamat</div>
      <div class="rv-value">{d.p1_pekerjaan || '—'}{d.p1_alamat ? ` · ${d.p1_alamat}` : ''}</div></div>
  </div>

  <div class="card mt-2">
    <h3>Tanah &amp; batas-batasnya</h3>
    <div class="section-sub">{d.jalan_gang}, RT {d.rt}, Kelurahan {d.kelurahan_nama}</div>

    {#if d.induk_id}
      <div class="notice" style="margin-bottom:1rem;">
        Tanah ini <strong>pecahan (sebagian) dari bidang</strong> pada berkas
        <a href={'#/berkas/' + d.induk_id}><span class="reg-chip">{d.induk_nomor}</span></a>{d.induk_luas ? ` — luas induk ${d.induk_luas.toLocaleString('id-ID')} M²` : ''}.
      </div>
    {/if}

    <div class="kompas kompas-view">
      {#each ['utara', 'selatan', 'barat', 'timur'] as a}
        <div class="k-sisi k-{a}">
          <div class="arah">{a}</div>
          <div class="nilai">{batasMap[a]?.nama || '—'}</div>
          <div class="sub">{batasMap[a]?.panjang_meter != null ? batasMap[a].panjang_meter.toLocaleString('id-ID') + ' meter' : 'panjang belum diisi'}</div>
        </div>
      {/each}
      <div class="k-tanah">
        <div>
          <div class="luas">{d.luas_m2.toLocaleString('id-ID')} M²</div>
          <div class="ket">{d.p1_nama}</div>
        </div>
      </div>
    </div>
  </div>

  {#if d.pecahan && d.pecahan.length > 0}
    <div class="card mt-2">
      <h3>Pecahan dari tanah ini</h3>
      <div class="section-sub">Bidang-bidang yang sudah dipecah dari berkas ini —
        total {d.total_luas_pecahan.toLocaleString('id-ID')} M² dari {d.luas_m2.toLocaleString('id-ID')} M²
        (sisa ± {Math.max(0, d.luas_m2 - d.total_luas_pecahan).toLocaleString('id-ID')} M²).
        {#if d.total_luas_pecahan > d.luas_m2}<strong style="color:var(--danger);">Total pecahan melebihi luas induk — periksa kembali.</strong>{/if}
      </div>
      <div class="table-wrap">
      <table>
        <thead><tr><th>No. berkas</th><th>Pemohon</th><th>Luas</th></tr></thead>
        <tbody>
          {#each d.pecahan as p}
            <tr style="cursor:pointer;" on:click={() => (location.hash = '#/berkas/' + p.id)}>
              <td><span class="reg-chip">{p.nomor_berkas}</span></td>
              <td>{p.p1_nama}</td>
              <td>{p.luas_m2.toLocaleString('id-ID')} M²</td>
            </tr>
          {/each}
        </tbody>
      </table>
      </div>
    </div>
  {/if}

  <div class="card mt-2">
    <h3>Pengukuran &amp; ganti rugi</h3>
    <div class="review-row"><div class="rv-label">Tanggal pengukuran</div>
      <div class="rv-value">{d.tgl_pengukuran ? fmtDate(d.tgl_pengukuran) : 'belum diisi — kosong di surat'}</div></div>
    <div class="review-row"><div class="rv-label">Juru ukur</div>
      <div class="rv-value">{d.juru_kel_nama || '—'} (kelurahan) · {d.juru_kec_nama || '—'} (kecamatan)</div></div>
    <div class="review-row"><div class="rv-label">Pihak Kedua</div>
      <div class="rv-value">{d.p2_nama || '—'}{d.p2_alamat ? ` · ${d.p2_alamat}` : ''}{d.luas_ganti_rugi_m2 != null ? ` · ganti rugi ${d.luas_ganti_rugi_m2.toLocaleString('id-ID')} m²` : ''}</div></div>
  </div>

  <div class="card mt-2">
    <h3>Saksi &amp; penandatangan</h3>
    <div class="review-row"><div class="rv-label">Saksi sempadan</div>
      <div class="rv-value">{#each d.saksi as s, i}{i ? ', ' : ''}{s}{/each}</div></div>
    <div class="review-row"><div class="rv-label">Lurah</div>
      <div class="rv-value">{d.lurah_nama ? `${d.lurah_nama} — NIP ${d.lurah_nip}` : '—'}</div></div>
    <div class="review-row"><div class="rv-label">Ketua RT</div>
      <div class="rv-value">{d.ketua_rt_nama ? `${d.ketua_rt_nama} (RT ${d.ketua_rt_nomor})` : '—'}</div></div>
    <div class="review-row"><div class="rv-label">Tanggal surat</div>
      <div class="rv-value">{fmtDate(d.tanggal_surat)}</div></div>
  </div>

  <div class="card mt-2">
    <h3>Yang akan tercetak</h3>
    <div class="section-sub">Tombol "Cetak 3 Surat" membuka halaman baru berisi ketiganya — tinggal tekan Ctrl+P.</div>
    <div class="surat-strip">
      <div class="surat-item"><div class="surat-name">1 · Surat Pernyataan Tidak Bersengketa</div>
        <span class="muted">Pernyataan pemohon, bermaterai 10.000</span></div>
      <div class="surat-item"><div class="surat-name">2 · Berita Acara Pengukuran Tanah</div>
        <span class="muted">Hasil pengukuran + pihak kedua</span></div>
      <div class="surat-item"><div class="surat-name">3 · Sceets Kaart</div>
        <span class="muted">Peta situasi — kotak digambar tangan setelah dicetak</span></div>
    </div>
  </div>
{/if}
