import { create } from 'zustand'
import type { LoginResponse } from '../types'
import { clearToken, loadToken, saveToken } from '../utils/auth'

interface AuthState {
  token: string
  password: string
  loading: boolean
  loginError: string
  setPassword: (value: string) => void
  login: () => Promise<void>
  logout: () => void
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: loadToken(),
  password: '',
  loading: false,
  loginError: '',

  setPassword: (value) => set({ password: value }),

  login: async () => {
    set({ loginError: '', loading: true })
    try {
      const { password } = get()
      const resp = await fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      })
      if (!resp.ok) {
        throw new Error('密码错误')
      }
      const data: LoginResponse = await resp.json()
      saveToken(data.token, data.expires_at)
      set({ token: data.token, password: '' })
    } catch {
      set({ loginError: '口令不正确或服务不可用' })
    } finally {
      set({ loading: false })
    }
  },

  logout: () => {
    clearToken()
    set({ token: '' })
  },
}))

export const forceUnauthenticated = () => {
  clearToken()
  useAuthStore.setState({
    token: '',
    password: '',
    loading: false,
    loginError: '',
  })
}
