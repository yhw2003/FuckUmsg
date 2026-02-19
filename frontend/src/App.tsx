import { useEffect, useMemo, useState } from 'react'
import './App.css'

type Todo = {
  id: number
  title: string
  detail: string
  source_type: 'group' | 'private'
  source_id: string
  source_name: string
  sender_id: number
  sender_name: string
  raw_message: string
  message_id: string
  created_at: number
  deadline_at: number
  completed_at: number | null
  status: 'open' | 'done'
}

type FailedMessage = {
  id: number
  user_id: number
  source_type: 'group' | 'private'
  source_id: string
  message_id: string
  raw_message: string
  fail_stage: 'classify' | 'summarize' | string
  error_text: string
  created_at: number
}

type LoginResponse = {
  token: string
  expires_at: number
  expires_in: number
}

const TOKEN_KEY = 'todo_token'
const TOKEN_EXPIRES_KEY = 'todo_token_expires'

const loadToken = () => {
  const token = localStorage.getItem(TOKEN_KEY) || ''
  const expires = Number(localStorage.getItem(TOKEN_EXPIRES_KEY) || '0')
  if (!token || !expires) {
    return ''
  }
  if (Date.now() / 1000 > expires) {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(TOKEN_EXPIRES_KEY)
    return ''
  }
  return token
}

const saveToken = (token: string, expiresAt: number) => {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(TOKEN_EXPIRES_KEY, String(expiresAt))
}

const clearToken = () => {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(TOKEN_EXPIRES_KEY)
}

const formatTime = (timestamp: number) => {
  if (!timestamp) return '未知'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(timestamp * 1000))
}

const formatDeadline = (timestamp: number) => {
  if (!timestamp) return '未设截止（具体时刻未定）'
  return formatTime(timestamp)
}

const sourceLabel = (todo: Todo) => {
  const name = todo.source_name || todo.source_id
  if (todo.source_type === 'group') {
    return `群聊 ${name}`
  }
  return `私聊 ${name}`
}

const senderLabel = (todo: Todo) => {
  if (todo.sender_name) {
    return `发送者 ${todo.sender_name}`
  }
  if (todo.sender_id) {
    return `发送者 ${todo.sender_id}`
  }
  return '发送者未知'
}

const failedSourceLabel = (item: FailedMessage) => {
  if (item.source_type === 'group') {
    return `群聊 ${item.source_id}`
  }
  return `私聊 ${item.source_id}`
}

const failedStageLabel = (stage: string) => {
  if (stage === 'classify') {
    return '分类失败'
  }
  if (stage === 'summarize') {
    return '摘要失败'
  }
  return stage || '未知阶段'
}

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
  const [activeView, setActiveView] = useState<'todos' | 'failed'>('todos')
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

  const fetchTodos = async () => {
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
    } catch (err) {
      setError('无法获取待办列表，请检查后端服务')
    } finally {
      setRefreshing(false)
    }
  }

  const fetchFailedMessages = async () => {
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
    } catch (err) {
      setFailedError('无法获取失败消息，请检查后端服务')
    } finally {
      setFailedLoading(false)
    }
  }

  useEffect(() => {
    if (token) {
      void fetchTodos()
      void fetchFailedMessages()
    }
  }, [token])

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
    } catch (err) {
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
    } catch (err) {
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
    } catch (err) {
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
    } catch (err) {
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
      <header className="hero">
        <div className="hero-title">消息代办看板</div>
        <div className="hero-subtitle">
          监听 OneBot11 消息并抽取他人交代的杂事，集中到一个清晰的待办列表。
        </div>
        {token ? (
          <div className="info-pill">
            已登录 · 共 {stats.total} 条 · 已完成 {stats.done} 条
          </div>
        ) : (
          <div className="info-pill">需要口令登录后查看待办</div>
        )}
      </header>

      <section className="panel">
        {!token ? (
          <>
            <div className="panel-header">
              <div className="panel-title">登录</div>
            </div>
            <div className="login-grid">
              <input
                className="input"
                type="password"
                placeholder="输入访问口令"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' && password) {
                    void handleLogin()
                  }
                }}
              />
              <button className="button" onClick={() => void handleLogin()} disabled={loading || !password}>
                {loading ? '登录中...' : '进入看板'}
              </button>
            </div>
            {loginError ? <div className="error">{loginError}</div> : null}
          </>
        ) : (
          <>
            <div className="panel-header">
              <div className="panel-title">{activeView === 'todos' ? '待办列表' : 'LLM失败消息'}</div>
              <div className="panel-actions">
                <div className="view-switch" role="tablist" aria-label="视图切换">
                  <button
                    className={`button secondary small ${activeView === 'todos' ? 'active' : ''}`}
                    onClick={() => setActiveView('todos')}
                  >
                    待办列表
                  </button>
                  <button
                    className={`button secondary small ${activeView === 'failed' ? 'active' : ''}`}
                    onClick={() => setActiveView('failed')}
                  >
                    LLM失败消息
                  </button>
                </div>
                {activeView === 'todos' ? (
                  <button className="button secondary" onClick={() => void fetchTodos()} disabled={refreshing}>
                    {refreshing ? '刷新中...' : '刷新'}
                  </button>
                ) : (
                  <button className="button secondary" onClick={() => void fetchFailedMessages()} disabled={failedLoading}>
                    {failedLoading ? '刷新中...' : '刷新'}
                  </button>
                )}
                <button className="button" onClick={handleLogout}>
                  退出
                </button>
              </div>
            </div>

            {activeView === 'todos' ? (
              <>
                {error ? <div className="error">{error}</div> : null}
                {openTodos.length === 0 && doneTodos.length === 0 ? (
                  <div className="empty">暂时没有代办，等消息进来再看看。</div>
                ) : (
                  <>
                    <div className="todo-list">
                      {openTodos.length === 0 ? (
                        <div className="empty">暂无未完成事项。</div>
                      ) : (
                        openTodos.map((todo, index) => (
                          <div
                            className="todo-item"
                            key={todo.id}
                            style={{ animationDelay: `${index * 50}ms` }}
                          >
                            <div>
                              {editingId === todo.id ? (
                                <input
                                  className="input todo-title-input"
                                  value={editingTitle}
                                  onChange={(event) => setEditingTitle(event.target.value)}
                                  onKeyDown={(event) => {
                                    if (event.key === 'Enter') {
                                      void saveEdit(todo)
                                    }
                                    if (event.key === 'Escape') {
                                      cancelEdit()
                                    }
                                  }}
                                />
                              ) : (
                                <div className="todo-title">{todo.title}</div>
                              )}
                              {editingId === todo.id ? (
                                <textarea
                                  className="input todo-detail-input"
                                  rows={3}
                                  value={editingDetail}
                                  onChange={(event) => setEditingDetail(event.target.value)}
                                />
                              ) : todo.detail ? (
                                <div className="todo-detail">{todo.detail}</div>
                              ) : todo.raw_message ? (
                                <div className="todo-detail muted">{todo.raw_message}</div>
                              ) : null}
                              <div className="todo-meta">
                                <span className="badge">待处理</span>
                                <span>{sourceLabel(todo)}</span>
                                <span>{senderLabel(todo)}</span>
                                <span>接收时间：{formatTime(todo.created_at)}</span>
                                <span>截止时间：{formatDeadline(todo.deadline_at)}</span>
                              </div>
                            </div>
                            <div className="todo-actions">
                              {editingId === todo.id ? (
                                <>
                                  <button className="button secondary small" onClick={() => void saveEdit(todo)}>
                                    保存
                                  </button>
                                  <button className="button ghost small" onClick={cancelEdit}>
                                    取消
                                  </button>
                                </>
                              ) : (
                                <>
                                  <span className="status-text">完成后可撤销</span>
                                  <button className="button secondary small" onClick={() => void toggleTodo(todo)}>
                                    完成
                                  </button>
                                  <button className="button ghost small" onClick={() => startEdit(todo)}>
                                    编辑
                                  </button>
                                  <button className="button danger small" onClick={() => void deleteTodo(todo)}>
                                    删除
                                  </button>
                                </>
                              )}
                            </div>
                          </div>
                        ))
                      )}
                    </div>
                    <div className="collapse-header">
                      <div className="collapse-title">
                        已完成 {doneTodos.length} 条
                      </div>
                      <button
                        className="button secondary small"
                        onClick={() => setShowCompleted((prev) => !prev)}
                        disabled={doneTodos.length === 0}
                      >
                        {showCompleted ? '收起' : '展开'}
                      </button>
                    </div>
                    {showCompleted && doneTodos.length > 0 ? (
                      <div className="todo-list compact">
                        {doneTodos.map((todo, index) => (
                          <div
                            className="todo-item done"
                            key={todo.id}
                            style={{ animationDelay: `${index * 50}ms` }}
                          >
                            <div>
                              {editingId === todo.id ? (
                                <input
                                  className="input todo-title-input"
                                  value={editingTitle}
                                  onChange={(event) => setEditingTitle(event.target.value)}
                                  onKeyDown={(event) => {
                                    if (event.key === 'Enter') {
                                      void saveEdit(todo)
                                    }
                                    if (event.key === 'Escape') {
                                      cancelEdit()
                                    }
                                  }}
                                />
                              ) : (
                                <div className="todo-title">{todo.title}</div>
                              )}
                              {editingId === todo.id ? (
                                <textarea
                                  className="input todo-detail-input"
                                  rows={3}
                                  value={editingDetail}
                                  onChange={(event) => setEditingDetail(event.target.value)}
                                />
                              ) : todo.detail ? (
                                <div className="todo-detail">{todo.detail}</div>
                              ) : todo.raw_message ? (
                                <div className="todo-detail muted">{todo.raw_message}</div>
                              ) : null}
                              <div className="todo-meta">
                                <span className="badge done">已完成</span>
                                <span>{sourceLabel(todo)}</span>
                                <span>{senderLabel(todo)}</span>
                                <span>接收时间：{formatTime(todo.created_at)}</span>
                                <span>截止时间：{formatDeadline(todo.deadline_at)}</span>
                              </div>
                            </div>
                            <div className="todo-actions">
                              {editingId === todo.id ? (
                                <>
                                  <button className="button secondary small" onClick={() => void saveEdit(todo)}>
                                    保存
                                  </button>
                                  <button className="button ghost small" onClick={cancelEdit}>
                                    取消
                                  </button>
                                </>
                              ) : (
                                <>
                                  <span className="status-text">可以撤销完成</span>
                                  <button className="button secondary small" onClick={() => void toggleTodo(todo)}>
                                    撤销
                                  </button>
                                  <button className="button ghost small" onClick={() => startEdit(todo)}>
                                    编辑
                                  </button>
                                  <button className="button danger small" onClick={() => void deleteTodo(todo)}>
                                    删除
                                  </button>
                                </>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : null}
                  </>
                )}
              </>
            ) : (
              <>
                {failedError ? <div className="error">{failedError}</div> : null}
                {failedMessages.length === 0 ? (
                  <div className="empty">暂无 LLM 失败消息。</div>
                ) : (
                  <div className="todo-list compact">
                    {failedMessages.map((item, index) => (
                      <div
                        className="todo-item failed-item"
                        key={item.id}
                        style={{ animationDelay: `${index * 40}ms` }}
                      >
                        <div>
                          <div className="todo-title">{item.raw_message || '（空消息）'}</div>
                          <div className="todo-detail muted">错误：{item.error_text}</div>
                          <div className="todo-meta">
                            <span className="badge failed">{failedStageLabel(item.fail_stage)}</span>
                            <span>{failedSourceLabel(item)}</span>
                            <span>发送者：{item.user_id}</span>
                            <span>接收时间：{formatTime(item.created_at)}</span>
                            <span>消息ID：{item.message_id || '未知'}</span>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </>
        )}
      </section>
    </div>
  )
}

export default App
