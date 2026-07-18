<script>
  import { api } from '../lib/api.js'
  import { user, notify } from '../lib/stores.js'

  export let forced = false

  let oldPassword = ''
  let newPassword = ''
  let confirm = ''
  let error = ''
  let busy = false

  async function submit() {
    error = ''
    if (newPassword.length < 6) { error = 'Password baru minimal 6 karakter'; return }
    if (newPassword !== confirm) { error = 'Konfirmasi password tidak cocok'; return }
    busy = true
    try {
      await api.post('/api/change-password', { old_password: oldPassword, new_password: newPassword })
      // refresh user (must_change_password jadi false)
      const me = await api.get('/api/me')
      user.set(me)
      notify('Password berhasil diubah', 'success')
    } catch (e) {
      error = e.message
    } finally {
      busy = false
    }
  }
</script>

{#if forced}
<div class="login-wrap">
  <div class="card login-card">
    <h2>Ganti Password</h2>
    <div class="alert alert-warn">Ini login pertama Anda. Demi keamanan, wajib ganti password default terlebih dahulu.</div>
    {#if error}<div class="alert alert-error">{error}</div>{/if}

    <form on:submit|preventDefault={submit}>
      <div class="field">
        <label for="op">Password Lama</label>
        <input id="op" type="password" bind:value={oldPassword} required />
      </div>
      <div class="field">
        <label for="np">Password Baru</label>
        <input id="np" type="password" bind:value={newPassword} required />
      </div>
      <div class="field">
        <label for="cp">Ulangi Password Baru</label>
        <input id="cp" type="password" bind:value={confirm} required />
      </div>
      <button class="btn btn-primary" style="width:100%;" disabled={busy}>
        {busy ? 'Menyimpan…' : 'Simpan Password'}
      </button>
    </form>
  </div>
</div>
{:else}
  {#if error}<div class="alert alert-error">{error}</div>{/if}
  <form on:submit|preventDefault={submit}>
    <div class="field">
      <label for="op2">Password Lama</label>
      <input id="op2" type="password" bind:value={oldPassword} required />
    </div>
    <div class="field">
      <label for="np2">Password Baru</label>
      <input id="np2" type="password" bind:value={newPassword} required />
    </div>
    <div class="field">
      <label for="cp2">Ulangi Password Baru</label>
      <input id="cp2" type="password" bind:value={confirm} required />
    </div>
    <button class="btn btn-primary" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan Password'}</button>
  </form>
{/if}
