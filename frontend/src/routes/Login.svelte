<script>
  import { login } from '../lib/stores.js'
  import kantorCamat from '../assets/kantor-camat.jpg'

  let username = ''
  let password = ''
  let error = ''
  let busy = false

  async function submit() {
    error = ''
    busy = true
    try {
      await login(username.trim(), password)
    } catch (e) {
      error = e.message
    } finally {
      busy = false
    }
  }
</script>

<div class="login-wrap">
  <div class="login-split">
    <div class="login-photo">
      <img src={kantorCamat} alt="Papan nama Kantor Camat Dumai Timur" />
      <div class="login-photo-caption">
        <strong>Kantor Camat Dumai Timur</strong>
        <span>Kota Dumai, Riau</span>
      </div>
    </div>

    <div class="login-form">
      <div class="brand" style="font-size:1.5rem;">SITANAH</div>
      <div class="muted small" style="margin-bottom:1.25rem;">Aplikasi Surat Tanah Garapan</div>

      {#if error}<div class="alert alert-error">{error}</div>{/if}

      <form on:submit|preventDefault={submit}>
        <div class="field">
          <label for="u">Username</label>
          <input id="u" bind:value={username} autocomplete="username" required />
        </div>
        <div class="field">
          <label for="p">Password</label>
          <input id="p" type="password" bind:value={password} autocomplete="current-password" required />
        </div>
        <button class="btn btn-primary" style="width:100%;" disabled={busy}>
          {busy ? 'Masuk…' : 'Masuk'}
        </button>
      </form>
    </div>
  </div>
</div>
