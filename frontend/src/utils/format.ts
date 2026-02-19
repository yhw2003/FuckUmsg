import type { FailedMessage, Todo } from '../types'

export const formatTime = (timestamp: number) => {
  if (!timestamp) return '未知'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(timestamp * 1000))
}

export const formatDeadline = (timestamp: number) => {
  if (!timestamp) return '未设截止（具体时刻未定）'
  return formatTime(timestamp)
}

export const sourceLabel = (todo: Todo) => {
  const name = todo.source_name || todo.source_id
  if (todo.source_type === 'group') {
    return `群聊 ${name}`
  }
  return `私聊 ${name}`
}

export const senderLabel = (todo: Todo) => {
  if (todo.sender_name) {
    return `发送者 ${todo.sender_name}`
  }
  if (todo.sender_id) {
    return `发送者 ${todo.sender_id}`
  }
  return '发送者未知'
}

export const failedSourceLabel = (item: FailedMessage) => {
  if (item.source_type === 'group') {
    return `群聊 ${item.source_id}`
  }
  return `私聊 ${item.source_id}`
}

export const failedStageLabel = (stage: string) => {
  if (stage === 'classify') {
    return '分类失败'
  }
  if (stage === 'summarize') {
    return '摘要失败'
  }
  return stage || '未知阶段'
}
