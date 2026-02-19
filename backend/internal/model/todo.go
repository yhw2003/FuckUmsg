package model

type Todo struct {
	ID                int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Title             string `json:"title" gorm:"not null"`
	Detail            string `json:"detail" gorm:"type:text"`
	SourceType        string `json:"source_type" gorm:"column:source_type;not null"`
	SourceID          string `json:"source_id" gorm:"column:source_id;not null"`
	SourceName        string `json:"source_name" gorm:"column:source_name"`
	SenderID          int64  `json:"sender_id" gorm:"column:sender_id"`
	SenderName        string `json:"sender_name" gorm:"-"`
	RawMessage        string `json:"raw_message" gorm:"column:raw_message;not null"`
	MessageID         string `json:"message_id" gorm:"column:message_id"`
	CreatedAt         int64  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime:false"`
	DeadlineAt        int64  `json:"deadline_at" gorm:"column:deadline_at;not null;default:0"`
	DeadlineLLMAt     int64  `json:"deadline_llm_at" gorm:"column:deadline_llm_at;not null;default:0"`
	DeadlineClassicAt int64  `json:"deadline_classic_at" gorm:"column:deadline_classic_at;not null;default:0"`
	DeadlineConflict  bool   `json:"deadline_conflict" gorm:"column:deadline_conflict;not null;default:false"`
	DeadlineNote      string `json:"deadline_note" gorm:"column:deadline_note"`
	CompletedAt       *int64 `json:"completed_at" gorm:"column:completed_at"`
	Status            string `json:"status" gorm:"column:status;not null"`
}

func (Todo) TableName() string {
	return "todos"
}

type LLMFailedMessage struct {
	ID         int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int64  `json:"user_id" gorm:"column:user_id;not null"`
	SourceType string `json:"source_type" gorm:"column:source_type;not null"`
	SourceID   string `json:"source_id" gorm:"column:source_id;not null"`
	MessageID  string `json:"message_id" gorm:"column:message_id"`
	RawMessage string `json:"raw_message" gorm:"column:raw_message;not null"`
	FailStage  string `json:"fail_stage" gorm:"column:fail_stage;not null"`
	ErrorText  string `json:"error_text" gorm:"column:error_text;type:text;not null"`
	CreatedAt  int64  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime:false"`
}

func (LLMFailedMessage) TableName() string {
	return "llm_failed_messages"
}
