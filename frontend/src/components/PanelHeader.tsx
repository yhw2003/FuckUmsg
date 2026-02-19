import type { ActiveView } from '../types'

type PanelHeaderProps = {
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
      <div className="panel-title">{activeView === 'todos' ? '待办列表' : 'LLM失败消息'}</div>
      <div className="panel-actions">
        <div className="view-switch" role="tablist" aria-label="视图切换">
          <button
            className={`button secondary small ${activeView === 'todos' ? 'active' : ''}`}
            onClick={() => onSwitchView('todos')}
          >
            待办列表
          </button>
          <button
            className={`button secondary small ${activeView === 'failed' ? 'active' : ''}`}
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
