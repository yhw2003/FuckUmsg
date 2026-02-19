import type { FailedMessage } from '../types'
import { failedSourceLabel, failedStageLabel, formatTime } from '../utils/format'

interface FailedMessageItemProps {
  item: FailedMessage
  index: number
}

function FailedMessageItem({ item, index }: FailedMessageItemProps) {
  return (
    <div className="todo-item failed-item" key={item.id} style={{ animationDelay: `${index * 40}ms` }}>
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
  )
}

export default FailedMessageItem
