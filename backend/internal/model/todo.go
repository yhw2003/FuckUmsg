package model

type Todo struct {
	ID          int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Title       string `json:"title" gorm:"not null"`
	Detail      string `json:"detail" gorm:"type:text"`
	SourceType  string `json:"source_type" gorm:"column:source_type;not null"`
	SourceID    string `json:"source_id" gorm:"column:source_id;not null"`
	SourceName  string `json:"source_name" gorm:"column:source_name"`
	SenderID    int64  `json:"sender_id" gorm:"column:sender_id"`
	SenderName  string `json:"sender_name" gorm:"-"`
	RawMessage  string `json:"raw_message" gorm:"column:raw_message;not null"`
	MessageID   string `json:"message_id" gorm:"column:message_id"`
	CreatedAt   int64  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime:false"`
	CompletedAt *int64 `json:"completed_at" gorm:"column:completed_at"`
	Status      string `json:"status" gorm:"column:status;not null"`
}

func (Todo) TableName() string {
	return "todos"
}
