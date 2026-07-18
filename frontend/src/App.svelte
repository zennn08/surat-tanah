<script>
  import { onMount } from 'svelte'
  import { user, loadSession, logout, toast } from './lib/stores.js'
  import { path, navigate, match } from './lib/router.js'

  import Login from './routes/Login.svelte'
  import ChangePassword from './routes/ChangePassword.svelte'
  import BerkasList from './routes/BerkasList.svelte'
  import BerkasForm from './routes/BerkasForm.svelte'
  import BerkasDetail from './routes/BerkasDetail.svelte'
  import MasterData from './routes/MasterData.svelte'
  import Pengaturan from './routes/Pengaturan.svelte'

  let version = ''
  onMount(() => {
    loadSession()
    fetch('/api/version').then((r) => r.json()).then((v) => (version = v.version)).catch(() => {})
  })

  // Resolusi route → { component, props }
  $: route = resolve($path)
  function resolve(p) {
    let m
    if (p === '/' || p === '') return { c: BerkasList, props: {} }
    if (p === '/berkas/baru') return { c: BerkasForm, props: {} }
    if ((m = match('/berkas/:id/edit', p))) return { c: BerkasForm, props: { id: m.id } }
    if ((m = match('/berkas/:id', p))) return { c: BerkasDetail, props: { id: m.id } }
    if (p === '/master') return { c: MasterData, props: {} }
    if (p === '/pengaturan') return { c: Pengaturan, props: {} }
    return { c: BerkasList, props: {} }
  }

  async function doLogout() {
    await logout()
    navigate('/')
  }

  const links = [
    { to: '/', label: 'Daftar Berkas' },
    { to: '/master', label: 'Data Pejabat & Wilayah' },
    { to: '/pengaturan', label: 'Pengaturan' },
  ]
  function isActive(p, to) {
    if (to === '/') return p === '/' || (p.startsWith('/berkas/') && p !== '/berkas/baru')
    return p === to
  }
</script>

{#if $user === undefined}
  <div class="spinner">Memuat…</div>
{:else if $user === null}
  <Login />
{:else if $user.must_change_password}
  <ChangePassword forced={true} />
{:else}
  <div class="app-shell">
    <header class="topbar">
      <div class="topbar-inner">
        <span class="brand">SITANAH<span class="brand-sub">Surat Tanah · Kec. Dumai Timur</span></span>
        <nav class="nav">
          {#each links as l}
            <a href={'#' + l.to} class:active={isActive($path, l.to)}>{l.label}</a>
          {/each}
        </nav>
        <span class="user">{$user.nama}</span>
        <button class="btn btn-sm btn-ghost" on:click={doLogout}>Keluar</button>
      </div>
    </header>
    <main class="container">
      {#key $path}
        <svelte:component this={route.c} {...route.props} />
      {/key}
    </main>
  </div>
{/if}

{#if $toast}
  <div class="toast {$toast.type}">{$toast.message}</div>
{/if}

<footer class="app-footer">SITANAH{version ? ` ${version}` : ''} · © Kukerta UNRI Kec. Dumai Timur 2026</footer>

<style>
  .brand-sub {
    display: block; font-size: 0.68rem; font-weight: 600; letter-spacing: 0.3px;
    color: var(--muted); margin-top: -2px;
  }
  .app-footer {
    flex-shrink: 0;
    text-align: center;
    padding: 14px 16px 18px;
    margin-top: 8px;
    font-size: 12px;
    color: var(--muted, #64748b);
  }
</style>
