import { useCallback, useEffect, useMemo, useState } from 'react'
import HeroHeader from './components/HeroHeader'
import LoginPanel from './components/LoginPanel'
import PanelHeader from './components/PanelHeader'
import FailedMessagesView from './components/FailedMessagesView'
import TodosView from './components/TodosView'
import type { ActiveView, FailedMessage, LoginResponse, Todo } from './types'
import { clearToken, loadToken, saveToken } from './utils/auth'
import './App.css'

function App() {
  const [token, setToken] = useState<string>(() => loadToken())
  const [password, setPassword] = useState('')
  const [todos, setTodos] = useState<Todo[]>([])
  const [loading, setLoading] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState('')
  const [loginError, setLoginError] = useState('')
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editingTitle, setEditingTitle] = useState('')
  const [editingDetail, setEditingDetail] = useState('')
  const [showCompleted, setShowCompleted] = useState(false)
  const [activeView, setActiveView] = useState<ActiveView>('todos')
  const [failedMessages, setFailedMessages] = useState<FailedMessage[]>([])
  const [failedLoading, setFailedLoading] = useState(false)
  const [failedError, setFailedError] = useState('')

  const stats = useMemo(() => {
    const total = todos.length
    const done = todos.filter((todo) => todo.status === 'done').length
    return { total, done }
  }, [todos])

  const openTodos = useMemo(() => todos.filter((todo) => todo.status === 'open'), [todos])
  const doneTodos = useMemo(() => todos.filter((todo) => todo.status === 'done'), [todos])

  const fetchTodos = useCallback(async () => {
    if (!token) return
    setError('')
    setRefreshing(true)
    try {
      const resp = await fetch('/api/todos', {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (resp.status === 401) {
        clearToken()
        setToken('')
        return
      }
      if (!resp.ok) {
        throw new Error('加载失败')
      }
      const data = await resp.json()
      setTodos(Array.isArray(data.todos) ? data.todos : [])
    } catch {
      setError('无法获取待办列表，请检查后端服务')
    } finally {
      setRefreshing(false)
    }
  }, [token])

  const fetchFailedMessages = useCallback(async () => {
    if (!token) return
    setFailedError('')
    setFailedLoading(true)
    try {
      const resp = await fetch('/api/failed-messages', {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (resp.status === 401) {
        clearToken()
        setToken('')
        return
      }
      if (!resp.ok) {
        throw new Error('加载失败')
      }
      const data = await resp.json()
      setFailedMessages(Array.isArray(data.failed_messages) ? data.failed_messages : [])
    } catch {
      setFailedError('无法获取失败消息，请检查后端服务')
    } finally {
      setFailedLoading(false)
    }
  }, [token])

  useEffect(() => {
    if (token) {
      void fetchTodos()
      void fetchFailedMessages()
    }
  }, [token, fetchTodos, fetchFailedMessages])

  const handleLogin = async () => {
    setLoginError('')
    setLoading(true)
    try {
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
      setToken(data.token)
      setPassword('')
    } catch {
      setLoginError('口令不正确或服务不可用')
    } finally {
      setLoading(false)
    }
  }

  const toggleTodo = async (todo: Todo) => {
    if (!token) return
    const nextAction = todo.status === 'done' ? 'reopen' : 'complete'
    try {
      const resp = await fetch(`/api/todos/${todo.id}/${nextAction}`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      })
      if (resp.status === 401) {
        clearToken()
        setToken('')
        return
      }
      if (!resp.ok) {
        throw new Error('更新失败')
      }
      setTodos((prev) =>
        prev.map((item) =>
          item.id === todo.id
            ? {
                ...item,
                status: todo.status === 'done' ? 'open' : 'done',
                completed_at:
                  todo.status === 'done' ? null : Math.floor(Date.now() / 1000),
              }
            : item,
        ),
      )
    } catch {
      setError('更新待办状态失败')
    }
  }

  const startEdit = (todo: Todo) => {
    setError('')
    setEditingId(todo.id)
    setEditingTitle(todo.title)
    setEditingDetail(todo.detail || todo.raw_message || '')
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditingTitle('')
    setEditingDetail('')
  }

  const saveEdit = async (todo: Todo) => {
    if (!token) return
    const nextTitle = editingTitle.trim()
    const nextDetail = editingDetail.trim()
    if (!nextTitle) {
      setError('标题不能为空')
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
        clearToken()
        setToken('')
        return
      }
      if (!resp.ok) {
        throw new Error('更新失败')
      }
      setTodos((prev) =>
        prev.map((item) =>
          item.id === todo.id ? { ...item, title: nextTitle, detail: nextDetail } : item,
        ),
      )
      cancelEdit()
    } catch {
      setError('更新待办失败')
    }
  }

  const deleteTodo = async (todo: Todo) => {
    if (!token) return
    const confirmed = window.confirm(`确认删除「${todo.title}」吗？此操作不可撤销。`)
    if (!confirmed) return
    try {
      const resp = await fetch(`/api/todos/${todo.id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      })
      if (resp.status === 401) {
        clearToken()
        setToken('')
        return
      }
      if (!resp.ok) {
        throw new Error('删除失败')
      }
      setTodos((prev) => prev.filter((item) => item.id !== todo.id))
      if (editingId === todo.id) {
        cancelEdit()
      }
    } catch {
      setError('删除待办失败')
    }
  }

  const handleLogout = () => {
    clearToken()
    setToken('')
    setTodos([])
    setFailedMessages([])
    setActiveView('todos')
  }

  return (
    <div className="app">
      <HeroHeader token={token} total={stats.total} done={stats.done} />

      <section className="panel">
        {!token ? (
          <LoginPanel
            password={password}
            loading={loading}
            loginError={loginError}
            onPasswordChange={setPassword}
            onLogin={handleLogin}
          />
        ) : (
          <>
            <PanelHeader
              activeView={activeView}
              refreshing={refreshing}
              failedLoading={failedLoading}
              onSwitchView={setActiveView}
              onRefreshTodos={fetchTodos}
              onRefreshFailed={fetchFailedMessages}
              onLogout={handleLogout}
            />

            {activeView === 'todos' ? (
              <TodosView
                error={error}
                openTodos={openTodos}
                doneTodos={doneTodos}
                showCompleted={showCompleted}
                editingId={editingId}
                editingTitle={editingTitle}
                editingDetail={editingDetail}
                onEditingTitleChange={setEditingTitle}
                onEditingDetailChange={setEditingDetail}
                onSaveEdit={saveEdit}
                onCancelEdit={cancelEdit}
                onToggle={toggleTodo}
                onStartEdit={startEdit}
                onDelete={deleteTodo}
                onToggleCompleted={() => setShowCompleted((prev) => !prev)}
              />
            ) : (
              <FailedMessagesView failedError={failedError} failedMessages={failedMessages} />
            )}
          </>
        )}
      </section>
    </div>
  )
}

export default App
