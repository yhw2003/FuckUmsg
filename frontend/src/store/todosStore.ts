import { create } from 'zustand'
import type { Todo } from '../types'
import { forceUnauthenticated, useAuthStore } from './authStore'
import { useUiStore } from './uiStore'

type TodosState = {
  todos: Todo[]
  refreshing: boolean
  error: string
  fetchTodos: () => Promise<void>
  toggleTodo: (todo: Todo) => Promise<void>
  startEdit: (todo: Todo) => void
  cancelEdit: () => void
  saveEdit: (todo: Todo) => Promise<void>
  deleteTodo: (todo: Todo) => Promise<void>
  resetTodosState: () => void
}

export const useTodosStore = create<TodosState>((set) => ({
  todos: [],
  refreshing: false,
  error: '',

  fetchTodos: async () => {
    const { token } = useAuthStore.getState()
    if (!token) return
    set({ error: '', refreshing: true })
    try {
      const resp = await fetch('/api/todos', {
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
      set({ todos: Array.isArray(data.todos) ? data.todos : [] })
    } catch {
      set({ error: '无法获取待办列表，请检查后端服务' })
    } finally {
      set({ refreshing: false })
    }
  },

  toggleTodo: async (todo) => {
    const { token } = useAuthStore.getState()
    if (!token) return
    const nextAction = todo.status === 'done' ? 'reopen' : 'complete'
    try {
      const resp = await fetch(`/api/todos/${todo.id}/${nextAction}`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      })
      if (resp.status === 401) {
        forceUnauthenticated()
        return
      }
      if (!resp.ok) {
        throw new Error('更新失败')
      }
      set((state) => ({
        todos: state.todos.map((item) =>
          item.id === todo.id
            ? {
                ...item,
                status: todo.status === 'done' ? 'open' : 'done',
                completed_at: todo.status === 'done' ? null : Math.floor(Date.now() / 1000),
              }
            : item,
        ),
      }))
    } catch {
      set({ error: '更新待办状态失败' })
    }
  },

  startEdit: (todo) => {
    set({ error: '' })
    useUiStore.setState({
      editingId: todo.id,
      editingTitle: todo.title,
      editingDetail: todo.detail || todo.raw_message || '',
    })
  },

  cancelEdit: () => {
    useUiStore.setState({
      editingId: null,
      editingTitle: '',
      editingDetail: '',
    })
  },

  saveEdit: async (todo) => {
    const { token } = useAuthStore.getState()
    if (!token) return
    const { editingTitle, editingDetail } = useUiStore.getState()
    const nextTitle = editingTitle.trim()
    const nextDetail = editingDetail.trim()
    if (!nextTitle) {
      set({ error: '标题不能为空' })
      return
    }
    try {
      const resp = await fetch(`/api/todos/${todo.id}`, {
        method: 'PATCH',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ title: nextTitle, detail: nextDetail }),
      })
      if (resp.status === 401) {
        forceUnauthenticated()
        return
      }
      if (!resp.ok) {
        throw new Error('更新失败')
      }
      set((state) => ({
        todos: state.todos.map((item) =>
          item.id === todo.id ? { ...item, title: nextTitle, detail: nextDetail } : item,
        ),
      }))
      useUiStore.setState({
        editingId: null,
        editingTitle: '',
        editingDetail: '',
      })
    } catch {
      set({ error: '更新待办失败' })
    }
  },

  deleteTodo: async (todo) => {
    const { token } = useAuthStore.getState()
    if (!token) return
    const confirmed = window.confirm(`确认删除「${todo.title}」吗？此操作不可撤销。`)
    if (!confirmed) return
    try {
      const resp = await fetch(`/api/todos/${todo.id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      })
      if (resp.status === 401) {
        forceUnauthenticated()
        return
      }
      if (!resp.ok) {
        throw new Error('删除失败')
      }
      set((state) => ({
        todos: state.todos.filter((item) => item.id !== todo.id),
      }))
      if (useUiStore.getState().editingId === todo.id) {
        useUiStore.setState({
          editingId: null,
          editingTitle: '',
          editingDetail: '',
        })
      }
    } catch {
      set({ error: '删除待办失败' })
    }
  },

  resetTodosState: () => {
    set({ todos: [] })
  },
}))
