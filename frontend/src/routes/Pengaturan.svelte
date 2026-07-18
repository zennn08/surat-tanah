<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { notify } from '../lib/stores.js'
  import ChangePassword from './ChangePassword.svelte'

  let form = { kecamatan: '', kota: '', kantor_pertanahan: '' }
  let loading = true
  let showPw = false

  async function load() {
    loading = true
    try { form = await api.get('/api/pengaturan') }
    catch (e) { notify(e.message, 'error') } finally { loading = false }
  }

  async function save() {
    try { await api.put('/api/pengaturan', form); notify('Pengaturan disimpan', 'success') }
    catch (e) { notify(e.message, 'error') }
  }

  onMount(load)
</script>

<h1>Pengaturan</h1>
<div class="section-sub">Identitas kantor yang dipakai di isi surat (bukan kop/logo — surat dicetak polos).</div>

{#if loading}
  <div class="spinner">Memuat…</div>
{:else}
  <div class="card">
    <h3>Identitas Kantor</h3>
    <form on:submit|preventDefault={save}>
      <div class="row row-2">
        <div class="field"><label>Kecamatan</label><input bind:value={form.kecamatan} placeholder="mis. Dumai Timur" /></div>
        <div class="field"><label>Kota</label><input bind:value={form.kota} placeholder="mis. Dumai" /></div>
      </div>
      <div class="field">
        <label>Kantor Pertanahan (untuk Sceets Kaart)</label>
        <input bind:value={form.kantor_pertanahan} placeholder="mis. Kantor Pertanahan Kota Dumai" />
      </div>
      <button class="btn btn-primary mt-1">Simpan Pengaturan</button>
    </form>
  </div>

  <div class="card mt-2">
    <div class="card-title">
      <h3 class="mb-0">Keamanan Akun</h3>
      <button class="btn btn-sm" on:click={() => (showPw = !showPw)}>{showPw ? 'Tutup' : 'Ganti Password'}</button>
    </div>
    {#if showPw}
      <div style="max-width:420px;"><ChangePassword forced={false} /></div>
    {/if}
  </div>
{/if}
