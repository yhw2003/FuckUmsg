import { useEffect, useMemo } from 'react'
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
    const done = todos.filter((todo) => todo.status === 'done').length
    return { total, done }
  }, [todos])

  const openTodos = useMemo(() => todos.filter((todo) => todo.status === 'open'), [todos])
  const doneTodos = useMemo(() => todos.filter((todo) => todo.status === 'done'), [todos])

  useEffect(() => {
    if (token) {
      void fetchTodos()
      void fetchFailedMessages()
    }
  }, [token, fetchTodos, fetchFailedMessages])

  const handleLogout = () => {
    logoutAndReset()
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
            onLogin={login}
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
                onToggleCompleted={toggleShowCompleted}
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
