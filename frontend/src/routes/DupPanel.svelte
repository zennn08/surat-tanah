<script>
  import { fmtDate } from '../lib/format.js'
  export let dup = null
  export let lengkap = false
  export let indukId = null // id berkas yang ditandai sebagai induk pecah bidang
  export let onInduk = null // dipanggil dengan (match|null) saat menandai/membatalkan induk

  const info = {
    merah: {
      cls: 'dup-merah', ikon: '⛔', judul: 'Tanah ini kemungkinan SAMA dengan berkas lama',
      ket: 'Lokasi dan keempat batasnya persis sama. Membuat surat kedua untuk tanah yang sama bisa menimbulkan sengketa. Biasanya yang dibutuhkan adalah mencetak ulang berkas lama — buka berkas di bawah ini.',
    },
    kuning: {
      cls: 'dup-kuning', ikon: '⚠️', judul: 'Mirip dengan berkas lama — periksa dulu',
      ket: 'Lokasinya sama dan sebagian batasnya cocok. Bandingkan dengan berkas di bawah ini sebelum melanjutkan. Kalau tanah ini memang pecahan (dijual sebagian) dari salah satu berkas di bawah, tandai berkas itu sebagai induknya.',
    },
    abu: {
      cls: 'dup-abu', ikon: 'ℹ️', judul: 'Ada berkas lain di jalan & RT yang sama',
      ket: 'Batasnya berbeda, kemungkinan memang tanah lain. Lihat sekilas untuk memastikan.',
    },
    aman: {
      cls: 'dup-aman', ikon: '✔️', judul: 'Aman — belum pernah dibuatkan surat',
      ket: 'Tidak ditemukan berkas lain dengan lokasi atau batas yang mirip.',
    },
  }
  $: indukTertandai = indukId && dup && dup.matches.some((m) => m.id === indukId)
</script>

{#if lengkap && dup}
  {@const s = info[dup.tingkat]}
  <div class="dup-panel {s.cls}" role="status">
    <div class="dup-judul"><span>{s.ikon}</span> {s.judul}</div>
    <div class="dup-ket">{s.ket}</div>
    {#if indukTertandai && dup.tingkat !== 'merah'}
      <div class="dup-ket"><strong>Berkas induk sudah ditandai</strong> — kemiripan dengan induknya memang wajar untuk pecah bidang.</div>
    {/if}

    {#each dup.matches as m}
      <div class="dup-match">
        <div class="m-head">
          <div>
            <span class="reg-chip">{m.nomor_berkas}</span>
            <strong> {m.p1_nama}</strong> · {fmtDate(m.tanggal_surat)}
            {#if m.id === indukId}<span class="badge badge-green">✓ induk bidang</span>{/if}
            <br>
            <span class="small">{m.jalan_gang}, RT {m.rt}, {m.kelurahan_nama} · {m.luas_m2.toLocaleString('id-ID')} M² ·
            <strong>{m.cocok_batas} dari 4 batas sama</strong></span>
          </div>
          <div class="flex gap">
            {#if onInduk}
              {#if m.id === indukId}
                <button type="button" class="btn btn-sm" on:click={() => onInduk(null)}>Batalkan induk</button>
              {:else}
                <button type="button" class="btn btn-sm" on:click={() => onInduk(m)}
                  title="Tanah yang sedang diisi adalah pecahan/sebagian dari bidang pada berkas ini">Tandai sebagai induk</button>
              {/if}
            {/if}
            <a class="btn btn-sm" href={'#/berkas/' + m.id} target="_blank" rel="noreferrer">Buka</a>
          </div>
        </div>
        <div class="m-batas">
          Batasnya: {#each m.batas as b, i}{i ? ' · ' : ''}{b.arah} <strong>{b.nama}</strong>{/each}
        </div>
      </div>
    {/each}
  </div>
{/if}
