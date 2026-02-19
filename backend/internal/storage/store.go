package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"chat-assist-backend/internal/model"
)

var ErrNotFound = errors.New("not found")

const (
	StatusOpen = "open"
	StatusDone = "done"
)

type Store struct {
	db *gorm.DB
}

type TodoInput struct {
	Title      string
	Detail     string
	SourceType string
	SourceID   string
	SourceName string
	SenderID   int64
	RawMessage string
	MessageID  string
	CreatedAt  int64
}

func Open(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(path), &gorm.Config{})
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Init(ctx context.Context) error {
	if s.db == nil {
		return errors.New("db is nil")
	}
	if err := s.db.WithContext(ctx).AutoMigrate(&model.Todo{}); err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Exec(
		"CREATE INDEX IF NOT EXISTS idx_todos_status_created ON todos(status, created_at DESC)",
	).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) Create(ctx context.Context, input TodoInput) (int64, error) {
	if input.Title == "" {
		return 0, errors.New("title is required")
	}
	todo := model.Todo{
		Title:      input.Title,
		Detail:     input.Detail,
		SourceType: input.SourceType,
		SourceID:   input.SourceID,
		SourceName: input.SourceName,
		SenderID:   input.SenderID,
		RawMessage: input.RawMessage,
		MessageID:  input.MessageID,
		CreatedAt:  input.CreatedAt,
		Status:     StatusOpen,
	}
	if err := s.db.WithContext(ctx).Create(&todo).Error; err != nil {
		return 0, err
	}
	return todo.ID, nil
}

func (s *Store) List(ctx context.Context) ([]model.Todo, error) {
	var todos []model.Todo
	if err := s.db.WithContext(ctx).
		Order("created_at DESC, id DESC").
		Find(&todos).Error; err != nil {
		return nil, err
	}
	return todos, nil
}

func (s *Store) UpdateStatus(ctx context.Context, id int64, status string, completedAt *int64) error {
	if status != StatusOpen && status != StatusDone {
		return fmt.Errorf("invalid status: %s", status)
	}
	result := s.db.WithContext(ctx).
		Model(&model.Todo{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       status,
			"completed_at": completedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateContent(ctx context.Context, id int64, title string, detail string) error {
	if title == "" {
		return errors.New("title is required")
	}
	result := s.db.WithContext(ctx).
		Model(&model.Todo{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"title":  title,
			"detail": detail,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	result := s.db.WithContext(ctx).Delete(&model.Todo{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
