import { useEffect, useMemo } from 'react'
import type { Todo } from './types'
import FailedMessagesView from './components/FailedMessagesView'
import LoginPanel from './components/LoginPanel'
import PanelHeader from './components/PanelHeader'
import TodosView from './components/TodosView'
import { useAuthStore } from './store/authStore'
import { useFailedMessagesStore } from './store/failedMessagesStore'
import { useSessionStore } from './store/sessionStore'
import { useTodosStore } from './store/todosStore'
import { useUiStore } from './store/uiStore'
import './App.css'

function App() {
  const { token, password, loading, loginError, setPassword, login } = useAuthStore()
  const { todos, refreshing, error, fetchTodos, toggleTodo, startEdit, cancelEdit, saveEdit, deleteTodo } =
    useTodosStore()
  const { failedMessages, failedLoading, failedError, fetchFailedMessages } = useFailedMessagesStore()
  const { logoutAndReset } = useSessionStore()
  const {
    editingId,
    editingTitle,
    editingDetail,
    showCompleted,
    activeView,
    setEditingTitle,
    setEditingDetail,
    setActiveView,
    toggleShowCompleted,
  } = useUiStore()

  const stats = useMemo(() => {
    const total = todos.length
    const done = todos.filter((todo: Todo) => todo.status === 'done').length
    return { total, done }
  }, [todos])

  const openTodos = useMemo(() => todos.filter((todo: Todo) => todo.status === 'open'), [todos])
  const doneTodos = useMemo(() => todos.filter((todo: Todo) => todo.status === 'done'), [todos])

  useEffect(() => {
    if (token) {
      void fetchTodos()
      void fetchFailedMessages()
    }
  }, [token, fetchTodos, fetchFailedMessages])

  const completionRate = stats.total === 0 ? 0 : Math.round((stats.done / stats.total) * 100)

  return (
    <div className="app-shell">
      <div className="orb orb-one" aria-hidden="true" />
      <div className="orb orb-two" aria-hidden="true" />

      <main className={`app ${token ? 'app-dashboard' : 'app-login'}`.trim()}>
        {!token ? (
          <section className="login-stage" aria-label="登录区域">
            <article className="card login-card" aria-label="口令登录">
              <LoginPanel
                password={password}
                loading={loading}
                loginError={loginError}
                onPasswordChange={setPassword}
                onLogin={login}
              />
            </article>
          </section>
        ) : (
          <section className="ops-shell" aria-label="代办控制台">
            <header className="ops-summary" role="region" aria-label="待处理概览">
              <article className="card task-overview-card">
                <div className="task-overview-head">
                  <h1 className="task-overview-title">任务概览</h1>
                  {activeView === 'failed' ? (
                    <button className="button button-secondary button-sm" onClick={() => setActiveView('todos')}>
                      返回待办
                    </button>
                  ) : null}
                </div>

                <div className="task-overview-grid">
                  <article className="task-kpi">
                    <span>全部任务</span>
                    <strong>{stats.total}</strong>
                  </article>
                  <article className="task-kpi">
                    <span>已完成</span>
                    <strong>{stats.done}</strong>
                  </article>
                  <article className="task-kpi highlight">
                    <span>未完成</span>
                    <strong>{openTodos.length}</strong>
                  </article>
                </div>

                <div className="task-progress-meta">
                  <span>完成进度</span>
                  <strong>{completionRate}%</strong>
                </div>
                <div className="progress-track" aria-label={`完成率 ${completionRate}%`}>
                  <div className="progress-fill" style={{ width: `${completionRate}%` }} />
                </div>
                <p className="helper-note">
                  已完成 {stats.done} / {stats.total}
                </p>
              </article>
            </header>

            <article className="card workspace-card" aria-label="任务工作区">
              <PanelHeader
                activeView={activeView}
                refreshing={refreshing}
                failedLoading={failedLoading}
                todoCount={stats.total}
                failedCount={failedMessages.length}
                onSwitchView={setActiveView}
                onRefreshTodos={fetchTodos}
                onRefreshFailed={fetchFailedMessages}
                onLogout={logoutAndReset}
              />

              <div
                className="workspace-content"
                id={activeView === 'todos' ? 'todos-panel' : 'failed-panel'}
                role="region"
                aria-live="polite"
                aria-label={activeView === 'todos' ? '待办列表区域' : '失败消息区域'}
              >
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
                    onToggleCompleted={toggleShowCompleted}
                  />
                ) : (
                  <FailedMessagesView failedError={failedError} failedMessages={failedMessages} />
                )}
              </div>
            </article>
          </section>
        )}
      </main>
    </div>
  )
}

export default App
