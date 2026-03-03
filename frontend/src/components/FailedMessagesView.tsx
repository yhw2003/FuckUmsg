import type { FailedMessage } from '../types'
import FailedMessageItem from './FailedMessageItem'

interface FailedMessagesViewProps {
  failedError: string
  failedMessages: FailedMessage[]
}

function FailedMessagesView({ failedError, failedMessages }: FailedMessagesViewProps) {
  return (
    <div className="section-stack">
      {failedError ? <div className="alert-error">{failedError}</div> : null}

      {failedMessages.length === 0 ? (
        <div className="empty-state">
          <h3>暂无失败消息</h3>
          <p>当 LLM 抽取过程异常时，对应消息会显示在这里。</p>
        </div>
      ) : (
        <>
          <div className="section-heading">
            <div>
              <h3>待排查失败样本</h3>
            </div>
            <span className="count-pill">{failedMessages.length} 条</span>
          </div>
          <div className="list-stack compact">
            {failedMessages.map((item, index) => (
              <FailedMessageItem item={item} index={index} key={item.id} />
            ))}
          </div>
        </>
      )}
    </div>
  )
}

export default FailedMessagesView
