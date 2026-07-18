// Format tanggal ISO (YYYY-MM-DD) → "12 Juli 2026". Nilai lain dikembalikan apa adanya.
const fmt = new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'long', year: 'numeric' })

// Svelte action: paksa isian hanya angka (untuk NIK/NIP).
export function digitsOnly(node) {
  const clean = () => {
    const v = node.value.replace(/\D/g, '')
    if (v !== node.value) {
      node.value = v
      node.dispatchEvent(new Event('input')) // sinkronkan kembali ke bind:value
    }
  }
  node.addEventListener('input', clean)
  return { destroy: () => node.removeEventListener('input', clean) }
}

export function fmtDate(s) {
  if (!s || !/^\d{4}-\d{2}-\d{2}/.test(s)) return s || '—'
  const d = new Date(s)
  return isNaN(d) ? s : fmt.format(d)
}
