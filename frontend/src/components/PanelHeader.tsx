import type { ActiveView } from '../types'

interface PanelHeaderProps {
  activeView: ActiveView
  refreshing: boolean
  failedLoading: boolean
  onSwitchView: (view: ActiveView) => void
  onRefreshTodos: () => Promise<void>
  onRefreshFailed: () => Promise<void>
  onLogout: () => void
}

function PanelHeader({
  activeView,
  refreshing,
  failedLoading,
  onSwitchView,
  onRefreshTodos,
  onRefreshFailed,
  onLogout,
}: PanelHeaderProps) {
  return (
    <div className="panel-header">
      <h2 className="panel-title">{activeView === 'todos' ? '待办列表' : 'LLM失败消息'}</h2>
      <div className="panel-actions">
        <div className="view-switch" role="tablist" aria-label="视图切换">
          <button
            className={`button secondary small ${activeView === 'todos' ? 'active' : ''}`}
            role="tab"
            id="tab-todos"
            aria-selected={activeView === 'todos'}
            aria-controls="todos-panel"
            onClick={() => onSwitchView('todos')}
          >
            待办列表
          </button>
          <button
            className={`button secondary small ${activeView === 'failed' ? 'active' : ''}`}
            role="tab"
            id="tab-failed"
            aria-selected={activeView === 'failed'}
            aria-controls="failed-panel"
            onClick={() => onSwitchView('failed')}
          >
            LLM失败消息
          </button>
        </div>
        {activeView === 'todos' ? (
          <button className="button secondary" onClick={() => void onRefreshTodos()} disabled={refreshing}>
            {refreshing ? '刷新中...' : '刷新'}
          </button>
        ) : (
          <button className="button secondary" onClick={() => void onRefreshFailed()} disabled={failedLoading}>
            {failedLoading ? '刷新中...' : '刷新'}
          </button>
        )}
        <button className="button" onClick={onLogout}>
          退出
        </button>
      </div>
    </div>
  )
}

export default PanelHeader
