import type { Todo } from '../types'
import { formatDeadline, formatTime, senderLabel, sourceLabel } from '../utils/format'

type TodoItemProps = {
  todo: Todo
  index: number
  isDone: boolean
  editingId: number | null
  editingTitle: string
  editingDetail: string
  onEditingTitleChange: (value: string) => void
  onEditingDetailChange: (value: string) => void
  onSaveEdit: (todo: Todo) => Promise<void>
  onCancelEdit: () => void
  onToggle: (todo: Todo) => Promise<void>
  onStartEdit: (todo: Todo) => void
  onDelete: (todo: Todo) => Promise<void>
}

function TodoItem({
  todo,
  index,
  isDone,
  editingId,
  editingTitle,
  editingDetail,
  onEditingTitleChange,
  onEditingDetailChange,
  onSaveEdit,
  onCancelEdit,
  onToggle,
  onStartEdit,
  onDelete,
}: TodoItemProps) {
  const isEditing = editingId === todo.id

  return (
    <div className={`todo-item ${isDone ? 'done' : ''}`.trim()} style={{ animationDelay: `${index * 50}ms` }}>
      <div>
        {isEditing ? (
          <input
            className="input todo-title-input"
            value={editingTitle}
            onChange={(event) => onEditingTitleChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                void onSaveEdit(todo)
              }
              if (event.key === 'Escape') {
                onCancelEdit()
              }
            }}
          />
        ) : (
          <div className="todo-title">{todo.title}</div>
        )}
        {isEditing ? (
          <textarea
            className="input todo-detail-input"
            rows={3}
            value={editingDetail}
            onChange={(event) => onEditingDetailChange(event.target.value)}
          />
        ) : todo.detail ? (
          <div className="todo-detail">{todo.detail}</div>
        ) : todo.raw_message ? (
          <div className="todo-detail muted">{todo.raw_message}</div>
        ) : null}
        <div className="todo-meta">
          <span className={`badge ${isDone ? 'done' : ''}`.trim()}>{isDone ? '已完成' : '待处理'}</span>
          <span>{sourceLabel(todo)}</span>
          <span>{senderLabel(todo)}</span>
          <span>接收时间：{formatTime(todo.created_at)}</span>
          <span>截止时间：{formatDeadline(todo.deadline_at)}</span>
        </div>
      </div>
      <div className="todo-actions">
        {isEditing ? (
          <>
            <button className="button secondary small" onClick={() => void onSaveEdit(todo)}>
              保存
            </button>
            <button className="button ghost small" onClick={onCancelEdit}>
              取消
            </button>
          </>
        ) : (
          <>
            <span className="status-text">{isDone ? '可以撤销完成' : '完成后可撤销'}</span>
            <button className="button secondary small" onClick={() => void onToggle(todo)}>
              {isDone ? '撤销' : '完成'}
            </button>
            <button className="button ghost small" onClick={() => onStartEdit(todo)}>
              编辑
            </button>
            <button className="button danger small" onClick={() => void onDelete(todo)}>
              删除
            </button>
          </>
        )}
      </div>
    </div>
  )
}

export default TodoItem
