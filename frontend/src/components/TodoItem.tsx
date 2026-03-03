import type { Todo } from '../types'
import { formatDeadline, formatTime, senderLabel, sourceLabel } from '../utils/format'

interface TodoItemProps {
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
    <article className={`todo-item ${isDone ? 'is-done' : ''}`.trim()} style={{ animationDelay: `${index * 45}ms` }}>
      <div className="todo-main">
        <div className="todo-headline">
          <span className={`tag ${isDone ? 'tag-done' : 'tag-open'}`.trim()}>{isDone ? '已完成' : '待处理'}</span>
          <span className="todo-number">任务 #{todo.id}</span>
        </div>

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
          <h4 className="todo-title">{todo.title}</h4>
        )}

        {isEditing ? (
          <textarea
            className="input todo-detail-input"
            rows={3}
            value={editingDetail}
            onChange={(event) => onEditingDetailChange(event.target.value)}
          />
        ) : todo.detail ? (
          <p className="todo-detail">{todo.detail}</p>
        ) : todo.raw_message ? (
          <p className="todo-detail muted">{todo.raw_message}</p>
        ) : null}

        <div className="todo-meta">
          <span className="meta-chip">{sourceLabel(todo)}</span>
          <span className="meta-chip">{senderLabel(todo)}</span>
          <span className="meta-chip">接收时间：{formatTime(todo.created_at)}</span>
          <span className="meta-chip">截止时间：{formatDeadline(todo.deadline_at)}</span>
        </div>
      </div>

      <div className="todo-actions">
        {isEditing ? (
          <>
            <button className="button button-primary button-sm" onClick={() => void onSaveEdit(todo)}>
              保存修改
            </button>
            <button className="button button-ghost button-sm" onClick={onCancelEdit}>
              取消
            </button>
          </>
        ) : (
          <>
            <p className="status-hint">{isDone ? '可撤销为待处理' : '完成后仍可回退'}</p>
            <button className="button button-secondary button-sm" onClick={() => void onToggle(todo)}>
              {isDone ? '撤销完成' : '标记完成'}
            </button>
            <button className="button button-ghost button-sm" onClick={() => onStartEdit(todo)}>
              编辑
            </button>
            <button className="button button-danger button-sm" onClick={() => void onDelete(todo)}>
              删除
            </button>
          </>
        )}
      </div>
    </article>
  )
}

export default TodoItem
