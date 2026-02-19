import type { FailedMessage } from '../types'
import FailedMessageItem from './FailedMessageItem'

type FailedMessagesViewProps = {
  failedError: string
  failedMessages: FailedMessage[]
}

function FailedMessagesView({ failedError, failedMessages }: FailedMessagesViewProps) {
  return (
    <>
      {failedError ? <div className="error">{failedError}</div> : null}
      {failedMessages.length === 0 ? (
        <div className="empty">暂无 LLM 失败消息。</div>
      ) : (
        <div className="todo-list compact">
          {failedMessages.map((item, index) => (
            <FailedMessageItem item={item} index={index} key={item.id} />
          ))}
        </div>
      )}
    </>
  )
}

export default FailedMessagesView
