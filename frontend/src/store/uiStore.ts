import { create } from 'zustand'
import type { ActiveView } from '../types'

type UiState = {
  editingId: number | null
  editingTitle: string
  editingDetail: string
  showCompleted: boolean
  activeView: ActiveView
  setEditingTitle: (value: string) => void
  setEditingDetail: (value: string) => void
  setActiveView: (view: ActiveView) => void
  toggleShowCompleted: () => void
  resetUiAfterLogout: () => void
}

export const useUiStore = create<UiState>((set) => ({
  editingId: null,
  editingTitle: '',
  editingDetail: '',
  showCompleted: false,
  activeView: 'todos',

  setEditingTitle: (value) => set({ editingTitle: value }),
  setEditingDetail: (value) => set({ editingDetail: value }),
  setActiveView: (view) => set({ activeView: view }),
  toggleShowCompleted: () => set((state) => ({ showCompleted: !state.showCompleted })),
  resetUiAfterLogout: () =>
    set({
      activeView: 'todos',
    }),
}))
