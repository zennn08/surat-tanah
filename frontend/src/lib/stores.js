import { writable } from 'svelte/store'
import { api } from './api.js'

// User yang sedang login (null = belum login, undefined = belum dicek).
export const user = writable(undefined)

export async function loadSession() {
  try {
    const u = await api.get('/api/me')
    user.set(u)
  } catch {
    user.set(null)
  }
}

export async function login(username, password) {
  const u = await api.post('/api/login', { username, password })
  user.set(u)
  return u
}

export async function logout() {
  try {
    await api.post('/api/logout')
  } finally {
    user.set(null)
  }
}

// Toast/notifikasi ringkas.
export const toast = writable(null)
let toastTimer
export function notify(message, type = 'info') {
  toast.set({ message, type })
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => toast.set(null), 4000)
}
