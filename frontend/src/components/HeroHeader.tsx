interface HeroHeaderProps {
  token: string
  total: number
  done: number
  open: number
  failed: number
  completionRate: number
}

function HeroHeader({ token, total, done, open, failed, completionRate }: HeroHeaderProps) {
  return (
    <header className="hero" role="banner">
      <p className="hero-eyebrow">OneBot11 · OpenAI Responses API</p>
      <h1 className="hero-title">消息代办中心</h1>
      <p className="hero-subtitle">把群聊和私聊里交代给你的事情统一收敛到一个轻量面板，按优先顺序处理。</p>

      {token ? (
        <div className="hero-footer">
          <div className="hero-status" role="status" aria-live="polite">
            已登录，队列同步正常
          </div>
          <div className="hero-metrics">
            <div className="hero-metric">
              <span>全部任务</span>
              <strong>{total}</strong>
            </div>
            <div className="hero-metric">
              <span>待处理</span>
              <strong>{open}</strong>
            </div>
            <div className="hero-metric">
              <span>已完成</span>
              <strong>{done}</strong>
            </div>
            <div className="hero-metric">
              <span>失败消息</span>
              <strong>{failed}</strong>
            </div>
            <div className="hero-metric">
              <span>完成率</span>
              <strong>{completionRate}%</strong>
            </div>
          </div>
        </div>
      ) : (
        <div className="hero-status">请输入访问口令后进入管理看板。</div>
      )}
    </header>
  )
}

export default HeroHeader
