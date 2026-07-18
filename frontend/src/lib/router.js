import { readable } from 'svelte/store'

// Router hash sederhana. Path contoh: #/, #/berkas/baru, #/berkas/12, #/pejabat.
function parse() {
  const raw = location.hash.replace(/^#/, '') || '/'
  return raw
}

export const path = readable(parse(), (set) => {
  const handler = () => set(parse())
  window.addEventListener('hashchange', handler)
  return () => window.removeEventListener('hashchange', handler)
})

export function navigate(to) {
  if (location.hash.replace(/^#/, '') === to) return
  location.hash = to
}

// Cocokkan pola seperti "/berkas/:id" terhadap path aktual.
export function match(pattern, actual) {
  const pp = pattern.split('/').filter(Boolean)
  const ap = actual.split('/').filter(Boolean)
  if (pp.length !== ap.length) return null
  const params = {}
  for (let i = 0; i < pp.length; i++) {
    if (pp[i].startsWith(':')) params[pp[i].slice(1)] = decodeURIComponent(ap[i])
    else if (pp[i] !== ap[i]) return null
  }
  return params
}
