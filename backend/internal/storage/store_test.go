package storage

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	store := New(db)
	if err := store.Init(context.Background()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return store
}

func TestCreateAndList_PersistDualDeadlineFields(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	_, err := store.Create(ctx, TodoInput{
		Title:             "测试待办",
		Detail:            "测试双日期冲突",
		SourceType:        "private",
		SourceID:          "123",
		SourceName:        "tester",
		SenderID:          123,
		RawMessage:        "后天或下周一",
		MessageID:         "m1",
		CreatedAt:         1700000000,
		DeadlineAt:        0,
		DeadlineLLMAt:     1700086400,
		DeadlineClassicAt: 1700172800,
		DeadlineConflict:  true,
		DeadlineNote:      "未能确认的日期：1月2日(llm) 或 1月3日(classic)",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	todos, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	got := todos[0]
	if got.DeadlineAt != 0 || got.DeadlineLLMAt != 1700086400 || got.DeadlineClassicAt != 1700172800 || !got.DeadlineConflict {
		t.Fatalf("unexpected deadline fields: %+v", got)
	}
	if got.DeadlineNote == "" {
		t.Fatalf("expected deadline note")
	}
}
