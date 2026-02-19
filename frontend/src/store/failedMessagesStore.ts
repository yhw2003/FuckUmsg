import { create } from 'zustand'
import type { FailedMessage } from '../types'
import { forceUnauthenticated, useAuthStore } from './authStore'

interface FailedMessagesState {
  failedMessages: FailedMessage[]
  failedLoading: boolean
  failedError: string
  fetchFailedMessages: () => Promise<void>
  resetFailedMessagesState: () => void
}

export const useFailedMessagesStore = create<FailedMessagesState>((set) => ({
  failedMessages: [],
  failedLoading: false,
  failedError: '',

  fetchFailedMessages: async () => {
    const { token } = useAuthStore.getState()
    if (!token) return
    set({ failedError: '', failedLoading: true })
    try {
      const resp = await fetch('/api/failed-messages', {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (resp.status === 401) {
        forceUnauthenticated()
        return
      }
      if (!resp.ok) {
        throw new Error('加载失败')
      }
      const data = await resp.json()
      set({ failedMessages: Array.isArray(data.failed_messages) ? data.failed_messages : [] })
    } catch {
      set({ failedError: '无法获取失败消息，请检查后端服务' })
    } finally {
      set({ failedLoading: false })
    }
  },

  resetFailedMessagesState: () => {
    set({ failedMessages: [] })
  },
}))
