import type { Todo } from '../types'
import TodoItem from './TodoItem'

type TodosViewProps = {
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
  return (
    <>
      {error ? <div className="error">{error}</div> : null}
      {openTodos.length === 0 && doneTodos.length === 0 ? (
        <div className="empty">暂时没有代办，等消息进来再看看。</div>
      ) : (
        <>
          <div className="todo-list">
            {openTodos.length === 0 ? (
              <div className="empty">暂无未完成事项。</div>
            ) : (
              openTodos.map((todo, index) => (
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
              ))
            )}
          </div>
          <div className="collapse-header">
            <div className="collapse-title">已完成 {doneTodos.length} 条</div>
            <button className="button secondary small" onClick={onToggleCompleted} disabled={doneTodos.length === 0}>
              {showCompleted ? '收起' : '展开'}
            </button>
          </div>
          {showCompleted && doneTodos.length > 0 ? (
            <div className="todo-list compact">
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
    </>
  )
}

export default TodosView
