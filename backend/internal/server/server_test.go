package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"chat-assist-backend/internal/auth"
	"chat-assist-backend/internal/config"
	"chat-assist-backend/internal/storage"
)

func TestListTodosIncludesDeadlineConflictFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	store := storage.New(db)
	if err := store.Init(t.Context()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	_, err = store.Create(t.Context(), storage.TodoInput{
		Title:             "冲突日期待办",
		Detail:            "需要人工确认日期",
		SourceType:        "private",
		SourceID:          "123",
		SourceName:        "tester",
		SenderID:          123,
		RawMessage:        "后天还是下周一？",
		MessageID:         "msg-1",
		CreatedAt:         time.Now().Unix(),
		DeadlineAt:        0,
		DeadlineLLMAt:     1700086400,
		DeadlineClassicAt: 1700172800,
		DeadlineConflict:  true,
		DeadlineNote:      "未能确认的日期：1月2日(llm) 或 1月3日(classic)",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	tokens := auth.NewTokenStore(10 * time.Minute)
	token, _, err := tokens.Issue()
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}

	s := New(config.Config{}, store, tokens, nil, zap.NewNop())
	r := gin.New()
	s.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/todos", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		Todos []map[string]any `json:"todos"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(payload.Todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(payload.Todos))
	}
	todo := payload.Todos[0]
	if _, ok := todo["deadline_llm_at"]; !ok {
		t.Fatalf("missing deadline_llm_at")
	}
	if _, ok := todo["deadline_classic_at"]; !ok {
		t.Fatalf("missing deadline_classic_at")
	}
	if _, ok := todo["deadline_conflict"]; !ok {
		t.Fatalf("missing deadline_conflict")
	}
	if _, ok := todo["deadline_note"]; !ok {
		t.Fatalf("missing deadline_note")
	}
}
