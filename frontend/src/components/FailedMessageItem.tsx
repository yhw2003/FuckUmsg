import type { FailedMessage } from '../types'
import { failedSourceLabel, failedStageLabel, formatTime } from '../utils/format'

interface FailedMessageItemProps {
  item: FailedMessage
  index: number
}

function FailedMessageItem({ item, index }: FailedMessageItemProps) {
  return (
    <article className="todo-item failed-item" style={{ animationDelay: `${index * 35}ms` }}>
      <div className="todo-main">
        <div className="todo-headline">
          <span className="tag tag-failed">{failedStageLabel(item.fail_stage)}</span>
          <span className="todo-number">失败 #{item.id}</span>
        </div>

        <h4 className="todo-title">{item.raw_message || '（空消息）'}</h4>
        <p className="todo-detail muted">错误详情：{item.error_text || '未知错误'}</p>

        <div className="todo-meta">
          <span className="meta-chip">{failedSourceLabel(item)}</span>
          <span className="meta-chip">发送者：{item.user_id}</span>
          <span className="meta-chip">接收时间：{formatTime(item.created_at)}</span>
          <span className="meta-chip">消息 ID：{item.message_id || '未知'}</span>
        </div>
      </div>
    </article>
  )
}

export default FailedMessageItem
