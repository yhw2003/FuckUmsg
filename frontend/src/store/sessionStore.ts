import { create } from 'zustand'
import { useAuthStore } from './authStore'
import { useFailedMessagesStore } from './failedMessagesStore'
import { useTodosStore } from './todosStore'
import { useUiStore } from './uiStore'

interface SessionState {
  logoutAndReset: () => void
}

export const useSessionStore = create<SessionState>(() => ({
  logoutAndReset: () => {
    useAuthStore.getState().logout()
    useTodosStore.getState().resetTodosState()
    useFailedMessagesStore.getState().resetFailedMessagesState()
    useUiStore.getState().resetUiAfterLogout()
  },
}))
