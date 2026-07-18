<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { notify } from '../lib/stores.js'
  import { digitsOnly } from '../lib/format.js'

  const tabs = [
    { key: 'kelurahan', label: 'Kelurahan', api: '/api/kelurahan',
      ket: 'Daftar kelurahan se-Kecamatan Dumai Timur. Sudah terisi; ubah hanya bila ada pemekaran atau ganti nama.' },
    { key: 'lurah', label: 'Lurah', api: '/api/lurah',
      ket: 'Lurah yang menandatangani surat. Satu lurah aktif per kelurahan — saat lurah berganti, tambahkan yang baru dan yang lama otomatis nonaktif.' },
    { key: 'ketua-rt', label: 'Ketua RT', api: '/api/ketua-rt',
      ket: 'Ketua RT ikut menandatangani surat. Isi sesuai RT tempat tanah berada; bisa ditambah kapan saja.' },
    { key: 'juru-ukur', label: 'Juru Ukur', api: '/api/juru-ukur',
      ket: 'Petugas pengukur tanah untuk Berita Acara: satu dari kelurahan, satu dari kecamatan.' },
  ]
  let tab = tabs[0]
  let items = []
  let kelurahan = [] // untuk dropdown
  let loading = true

  let editingId = null
  let form = blank('kelurahan')
  function blank(key) {
    switch (key) {
      case 'kelurahan': return { nama: '', aktif: true }
      case 'lurah': return { kelurahan_id: 0, nama: '', nip: '', aktif: true }
      case 'ketua-rt': return { kelurahan_id: 0, nomor_rt: '', nama: '', aktif: true }
      case 'juru-ukur': return { tingkat: 'kelurahan', kelurahan_id: 0, nama: '', nip: '', aktif: true }
    }
  }

  async function load() {
    loading = true
    try {
      items = await api.get(tab.api)
      kelurahan = await api.get('/api/kelurahan')
    } catch (e) { notify(e.message, 'error') } finally { loading = false }
  }

  function switchTab(t) {
    tab = t
    cancelEdit()
    load()
  }

  function startEdit(it) {
    editingId = it.id
    form = { ...blank(tab.key), ...it }
    if (form.kelurahan_id === null) form.kelurahan_id = 0
  }
  function cancelEdit() { editingId = null; form = blank(tab.key) }

  async function save() {
    const body = { ...form }
    if (body.kelurahan_id !== undefined) {
      body.kelurahan_id = Number(body.kelurahan_id) || null
      if (tab.key !== 'juru-ukur' && !body.kelurahan_id) {
        notify('Kelurahan wajib dipilih', 'error'); return
      }
    }
    try {
      if (editingId) await api.put(tab.api + '/' + editingId, body)
      else await api.post(tab.api, body)
      cancelEdit()
      await load()
      notify('Data disimpan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  async function del(it) {
    if (!confirm(`Hapus ${tab.label} "${it.nama}"?`)) return
    try { await api.del(tab.api + '/' + it.id); await load(); notify('Data dihapus', 'success') }
    catch (e) { notify(e.message, 'error') }
  }

  onMount(load)
</script>

<h1>Data Pejabat &amp; Wilayah</h1>
<div class="page-sub">Nama-nama yang tinggal dipilih saat membuat berkas: kelurahan, lurah, ketua RT, dan juru ukur.
Isi sekali, dipakai terus.</div>

<div class="flex gap" style="margin-bottom:1rem; flex-wrap:wrap;">
  {#each tabs as t}
    <button class="btn {tab.key === t.key ? 'btn-primary' : ''}" on:click={() => switchTab(t)}>{t.label}</button>
  {/each}
</div>
<div class="notice" style="margin-bottom:1rem;">{tab.ket}</div>

<div class="card">
  <h3>{editingId ? `Edit ${tab.label}` : `Tambah ${tab.label}`}</h3>
  <form on:submit|preventDefault={save}>
    {#if tab.key === 'kelurahan'}
      <div class="row row-2">
        <div class="field"><label>Nama Kelurahan</label><input bind:value={form.nama} required /></div>
        <div class="field"><label>&nbsp;</label>
          <label class="flex items-center gap" style="font-weight:500;color:var(--text);text-transform:none;">
            <input type="checkbox" style="width:auto;" bind:checked={form.aktif} /> Aktif
          </label>
        </div>
      </div>
    {:else if tab.key === 'lurah'}
      <div class="row row-3">
        <div class="field"><label>Kelurahan</label>
          <select bind:value={form.kelurahan_id} required>
            <option value={0} disabled>— pilih —</option>
            {#each kelurahan as k}<option value={k.id}>{k.nama}</option>{/each}
          </select>
        </div>
        <div class="field"><label>Nama</label><input bind:value={form.nama} required /></div>
        <div class="field"><label>NIP</label><input bind:value={form.nip} use:digitsOnly class="mono" inputmode="numeric" required /></div>
      </div>
      <div class="field">
        <label class="flex items-center gap" style="font-weight:500;color:var(--text);text-transform:none;">
          <input type="checkbox" style="width:auto;" bind:checked={form.aktif} /> Jadikan lurah aktif (lurah lain se-kelurahan otomatis non-aktif)
        </label>
      </div>
    {:else if tab.key === 'ketua-rt'}
      <div class="row row-3">
        <div class="field"><label>Kelurahan</label>
          <select bind:value={form.kelurahan_id} required>
            <option value={0} disabled>— pilih —</option>
            {#each kelurahan as k}<option value={k.id}>{k.nama}</option>{/each}
          </select>
        </div>
        <div class="field"><label>Nomor RT</label><input bind:value={form.nomor_rt} class="mono" placeholder="mis. 007" required /></div>
        <div class="field"><label>Nama</label><input bind:value={form.nama} required /></div>
      </div>
      <div class="field">
        <label class="flex items-center gap" style="font-weight:500;color:var(--text);text-transform:none;">
          <input type="checkbox" style="width:auto;" bind:checked={form.aktif} /> Aktif
        </label>
      </div>
    {:else}
      <div class="row row-2">
        <div class="field"><label>Tingkat</label>
          <select bind:value={form.tingkat}>
            <option value="kelurahan">Kelurahan</option>
            <option value="kecamatan">Kecamatan</option>
          </select>
        </div>
        {#if form.tingkat === 'kelurahan'}
          <div class="field"><label>Kelurahan</label>
            <select bind:value={form.kelurahan_id} required>
              <option value={0} disabled>— pilih —</option>
              {#each kelurahan as k}<option value={k.id}>{k.nama}</option>{/each}
            </select>
          </div>
        {/if}
      </div>
      <div class="row row-2">
        <div class="field"><label>Nama</label><input bind:value={form.nama} required /></div>
        <div class="field"><label>NIP</label><input bind:value={form.nip} use:digitsOnly class="mono" inputmode="numeric" required /></div>
      </div>
      <div class="field">
        <label class="flex items-center gap" style="font-weight:500;color:var(--text);text-transform:none;">
          <input type="checkbox" style="width:auto;" bind:checked={form.aktif} /> Aktif
        </label>
      </div>
    {/if}
    <div class="flex gap">
      <button class="btn btn-primary">{editingId ? 'Simpan Perubahan' : 'Tambah'}</button>
      {#if editingId}<button type="button" class="btn btn-ghost" on:click={cancelEdit}>Batal</button>{/if}
    </div>
  </form>
</div>

<div class="card mt-2">
  {#if loading}
    <div class="spinner">Memuat…</div>
  {:else if items.length === 0}
    <div class="empty">Belum ada data {tab.label}.</div>
  {:else}
    <div class="table-wrap">
    <table>
      <thead>
        <tr>
          {#if tab.key === 'kelurahan'}
            <th>Nama</th>
          {:else if tab.key === 'lurah'}
            <th>Kelurahan</th><th>Nama</th><th>NIP</th>
          {:else if tab.key === 'ketua-rt'}
            <th>Kelurahan</th><th>RT</th><th>Nama</th>
          {:else}
            <th>Tingkat</th><th>Kelurahan</th><th>Nama</th><th>NIP</th>
          {/if}
          <th>Status</th><th style="width:140px;"></th>
        </tr>
      </thead>
      <tbody>
        {#each items as it}
          <tr>
            {#if tab.key === 'kelurahan'}
              <td>{it.nama}</td>
            {:else if tab.key === 'lurah'}
              <td>{it.kelurahan_nama}</td><td>{it.nama}</td><td class="mono">{it.nip}</td>
            {:else if tab.key === 'ketua-rt'}
              <td>{it.kelurahan_nama}</td><td class="mono">{it.nomor_rt}</td><td>{it.nama}</td>
            {:else}
              <td style="text-transform:capitalize;">{it.tingkat}</td><td>{it.kelurahan_nama || '—'}</td><td>{it.nama}</td><td class="mono">{it.nip}</td>
            {/if}
            <td>{#if it.aktif}<span class="badge badge-green">Aktif</span>{:else}<span class="badge badge-gray">Non-aktif</span>{/if}</td>
            <td><div class="table-actions">
              <button class="btn btn-sm" on:click={() => startEdit(it)}>Edit</button>
              <button class="btn btn-sm btn-danger" on:click={() => del(it)}>Hapus</button>
            </div></td>
          </tr>
        {/each}
      </tbody>
    </table>
    </div>
  {/if}
</div>
