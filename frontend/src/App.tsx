import { useEffect, useMemo } from 'react'
import type { Todo } from './types'
import HeroHeader from './components/HeroHeader'
import LoginPanel from './components/LoginPanel'
import PanelHeader from './components/PanelHeader'
import FailedMessagesView from './components/FailedMessagesView'
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

  const handleLogout = () => {
    logoutAndReset()
  }

  const completionRate = stats.total === 0 ? 0 : Math.round((stats.done / stats.total) * 100)

  return (
    <div className="app">
      <HeroHeader token={token} total={stats.total} done={stats.done} />

      {!token ? (
        <section className="panel bento-login" aria-label="登录面板">
          <LoginPanel
            password={password}
            loading={loading}
            loginError={loginError}
            onPasswordChange={setPassword}
            onLogin={login}
          />
        </section>
      ) : (
        <section className="bento-grid" aria-label="代办看板布局">
          <article className="panel bento-card bento-card-controls">
            <PanelHeader
              activeView={activeView}
              refreshing={refreshing}
              failedLoading={failedLoading}
              onSwitchView={setActiveView}
              onRefreshTodos={fetchTodos}
              onRefreshFailed={fetchFailedMessages}
              onLogout={handleLogout}
            />
          </article>

          <article
            className="panel bento-card bento-card-main"
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
          </article>

          <aside className="bento-side" aria-label="看板概览">
            <article className="panel bento-card bento-stat-card" role="region" aria-label="代办统计">
              <h2 className="bento-card-title">任务进度</h2>
              <div className="bento-stat-grid">
                <div className="bento-kpi">
                  <span className="bento-kpi-label">全部</span>
                  <strong>{stats.total}</strong>
                </div>
                <div className="bento-kpi">
                  <span className="bento-kpi-label">待处理</span>
                  <strong>{openTodos.length}</strong>
                </div>
                <div className="bento-kpi">
                  <span className="bento-kpi-label">已完成</span>
                  <strong>{stats.done}</strong>
                </div>
                <div className="bento-kpi">
                  <span className="bento-kpi-label">失败消息</span>
                  <strong>{failedMessages.length}</strong>
                </div>
              </div>
              <div className="progress-track" aria-label={`完成率 ${completionRate}%`}>
                <div className="progress-fill" style={{ width: `${completionRate}%` }} />
              </div>
              <p className="bento-note">完成率 {completionRate}%</p>
            </article>

            <article className="panel bento-card bento-meta-card" role="region" aria-label="当前视图信息">
              <h2 className="bento-card-title">当前视图</h2>
              <p className="bento-note">{activeView === 'todos' ? '你正在查看待办处理队列。' : '你正在查看 LLM 处理失败消息。'}</p>
              <p className="bento-note muted-note">支持快捷切换、刷新和原地编辑，状态改动会即时同步。</p>
            </article>
          </aside>
        </section>
      )}
    </div>
  )
}

export default App
