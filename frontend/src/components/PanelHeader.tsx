import type { ActiveView } from '../types'

interface PanelHeaderProps {
  activeView: ActiveView
  refreshing: boolean
  failedLoading: boolean
  todoCount: number
  failedCount: number
  onSwitchView: (view: ActiveView) => void
  onRefreshTodos: () => Promise<void>
  onRefreshFailed: () => Promise<void>
  onLogout: () => void
}

function PanelHeader({
  activeView,
  refreshing,
  failedLoading,
  todoCount,
  failedCount,
  onSwitchView,
  onRefreshTodos,
  onRefreshFailed,
  onLogout,
}: PanelHeaderProps) {
  const loading = activeView === 'todos' ? refreshing : failedLoading

  return (
    <div className="panel-header controls-header">
      <div className="panel-title-group">
        <h2 className="panel-title">{activeView === 'todos' ? '待办事项' : '失败消息'}</h2>
      </div>

      <div className="panel-toolbar">
        <div className="segmented" role="tablist" aria-label="视图切换">
          <button
            className={`button button-tab button-sm ${activeView === 'todos' ? 'is-active' : ''}`.trim()}
            role="tab"
            id="tab-todos"
            aria-selected={activeView === 'todos'}
            aria-controls="todos-panel"
            onClick={() => onSwitchView('todos')}
          >
            待办
            <span className="count-badge" aria-hidden="true">
              {todoCount}
            </span>
          </button>
          <button
            className={`button button-tab button-sm ${activeView === 'failed' ? 'is-active' : ''}`.trim()}
            role="tab"
            id="tab-failed"
            aria-selected={activeView === 'failed'}
            aria-controls="failed-panel"
            onClick={() => onSwitchView('failed')}
          >
            失败消息
            <span className="count-badge" aria-hidden="true">
              {failedCount}
            </span>
          </button>
        </div>

        <div className="action-group">
          {activeView === 'todos' ? (
            <button className="button button-secondary button-sm" onClick={() => void onRefreshTodos()} disabled={loading}>
              {loading ? '刷新中...' : '刷新'}
            </button>
          ) : (
            <button className="button button-secondary button-sm" onClick={() => void onRefreshFailed()} disabled={loading}>
              {loading ? '刷新中...' : '刷新'}
            </button>
          )}
          <button className="button button-ghost button-sm" onClick={onLogout}>
            退出登录
          </button>
        </div>
      </div>
    </div>
  )
}

export default PanelHeader
