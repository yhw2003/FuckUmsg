import type { Todo } from '../types'
import TodoItem from './TodoItem'

interface TodosViewProps {
  error: string
  openTodos: Todo[]
  doneTodos: Todo[]
  showCompleted: boolean
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
  onToggleCompleted: () => void
}

function TodosView({
  error,
  openTodos,
  doneTodos,
  showCompleted,
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
  onToggleCompleted,
}: TodosViewProps) {
  const total = openTodos.length + doneTodos.length

  return (
    <div className="section-stack">
      {error ? <div className="alert-error">{error}</div> : null}

      {total === 0 ? (
        <div className="empty-state">
          <h3>暂无待办</h3>
          <p>系统会在接收到可抽取消息后自动生成事项。</p>
        </div>
      ) : (
        <>
          <div className="section-heading">
            <div>
              <h3>待处理事项</h3>
            </div>
            <span className="count-pill">{openTodos.length} 条</span>
          </div>

          {openTodos.length === 0 ? (
            <div className="empty-state subtle">
              <h3>没有未完成事项</h3>
              <p>当前队列中的任务都已处理完毕。</p>
            </div>
          ) : (
            <div className="list-stack">
              {openTodos.map((todo, index) => (
                <TodoItem
                  todo={todo}
                  index={index}
                  isDone={false}
                  editingId={editingId}
                  editingTitle={editingTitle}
                  editingDetail={editingDetail}
                  onEditingTitleChange={onEditingTitleChange}
                  onEditingDetailChange={onEditingDetailChange}
                  onSaveEdit={onSaveEdit}
                  onCancelEdit={onCancelEdit}
                  onToggle={onToggle}
                  onStartEdit={onStartEdit}
                  onDelete={onDelete}
                  key={todo.id}
                />
              ))}
            </div>
          )}

          <div className="collapse-row">
            <div>
              <h3>已完成任务</h3>
            </div>
            <button className="button button-secondary button-sm" onClick={onToggleCompleted} disabled={doneTodos.length === 0}>
              {showCompleted ? '收起列表' : '展开列表'}
            </button>
          </div>

          {showCompleted && doneTodos.length > 0 ? (
            <div className="list-stack compact">
              {doneTodos.map((todo, index) => (
                <TodoItem
                  todo={todo}
                  index={index}
                  isDone
                  editingId={editingId}
                  editingTitle={editingTitle}
                  editingDetail={editingDetail}
                  onEditingTitleChange={onEditingTitleChange}
                  onEditingDetailChange={onEditingDetailChange}
                  onSaveEdit={onSaveEdit}
                  onCancelEdit={onCancelEdit}
                  onToggle={onToggle}
                  onStartEdit={onStartEdit}
                  onDelete={onDelete}
                  key={todo.id}
                />
              ))}
            </div>
          ) : null}
        </>
      )}
    </div>
  )
}

export default TodosView
