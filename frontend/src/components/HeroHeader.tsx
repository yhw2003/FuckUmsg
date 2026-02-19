interface HeroHeaderProps {
  token: string
  total: number
  done: number
}

function HeroHeader({ token, total, done }: HeroHeaderProps) {
  return (
    <header className="hero">
      <div className="hero-title">消息代办看板</div>
      <div className="hero-subtitle">监听 OneBot11 消息并抽取他人交代的杂事，集中到一个清晰的待办列表。</div>
      {token ? (
        <div className="info-pill">
          已登录 · 共 {total} 条 · 已完成 {done} 条
        </div>
      ) : (
        <div className="info-pill">需要口令登录后查看待办</div>
      )}
    </header>
  )
}

export default HeroHeader
