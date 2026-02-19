package server

import (
	"context"
	"encoding/json"
	"errors"
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
	"chat-assist-backend/internal/model"
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

func TestListFailedMessages_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	store := storage.New(db)
	if err := store.Init(t.Context()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	tokens := auth.NewTokenStore(10 * time.Minute)

	s := New(config.Config{}, store, tokens, nil, zap.NewNop())
	r := gin.New()
	s.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/failed-messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListFailedMessages_FilterByUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	store := storage.New(db)
	if err := store.Init(t.Context()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	_, err = store.CreateLLMFailedMessage(t.Context(), storage.LLMFailedMessageInput{
		UserID:     1001,
		SourceType: "private",
		SourceID:   "1001",
		MessageID:  "m-u1-a",
		RawMessage: "u1 message a",
		FailStage:  "classify",
		ErrorText:  "timeout",
		CreatedAt:  1700000100,
	})
	if err != nil {
		t.Fatalf("seed failed message 1 failed: %v", err)
	}
	_, err = store.CreateLLMFailedMessage(t.Context(), storage.LLMFailedMessageInput{
		UserID:     1002,
		SourceType: "group",
		SourceID:   "2002",
		MessageID:  "m-u2-a",
		RawMessage: "u2 message",
		FailStage:  "summarize",
		ErrorText:  "parse error",
		CreatedAt:  1700000200,
	})
	if err != nil {
		t.Fatalf("seed failed message 2 failed: %v", err)
	}
	_, err = store.CreateLLMFailedMessage(t.Context(), storage.LLMFailedMessageInput{
		UserID:     1001,
		SourceType: "group",
		SourceID:   "2001",
		MessageID:  "m-u1-b",
		RawMessage: "u1 message b",
		FailStage:  "summarize",
		ErrorText:  "invalid json",
		CreatedAt:  1700000300,
	})
	if err != nil {
		t.Fatalf("seed failed message 3 failed: %v", err)
	}

	tokens := auth.NewTokenStore(10 * time.Minute)
	token, _, err := tokens.IssueForUser(1001)
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}

	s := New(config.Config{}, store, tokens, nil, zap.NewNop())
	r := gin.New()
	s.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/failed-messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		FailedMessages []model.LLMFailedMessage `json:"failed_messages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(payload.FailedMessages) != 2 {
		t.Fatalf("expected 2 failed messages, got %d", len(payload.FailedMessages))
	}
	if payload.FailedMessages[0].MessageID != "m-u1-b" || payload.FailedMessages[1].MessageID != "m-u1-a" {
		t.Fatalf("unexpected order/messages: %+v", payload.FailedMessages)
	}
	for _, item := range payload.FailedMessages {
		if item.UserID != 1001 {
			t.Fatalf("expected only user 1001 messages, got user %d", item.UserID)
		}
	}
}

func TestListFailedMessages_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	store := storage.New(db)
	if err := store.Init(t.Context()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	tokens := auth.NewTokenStore(10 * time.Minute)
	token, _, err := tokens.IssueForUser(1001)
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}

	s := New(config.Config{}, store, tokens, nil, zap.NewNop())
	r := gin.New()
	s.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/failed-messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var payload struct {
		FailedMessages []model.LLMFailedMessage `json:"failed_messages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(payload.FailedMessages) != 0 {
		t.Fatalf("expected empty failed messages, got %d", len(payload.FailedMessages))
	}
}

type serverStoreStub struct {
	*storage.Store
	listFailedMessagesFunc func(ctx context.Context, userID int64) ([]model.LLMFailedMessage, error)
}

func (s *serverStoreStub) ListLLMFailedMessagesByUser(ctx context.Context, userID int64) ([]model.LLMFailedMessage, error) {
	if s.listFailedMessagesFunc != nil {
		return s.listFailedMessagesFunc(ctx, userID)
	}
	return s.Store.ListLLMFailedMessagesByUser(ctx, userID)
}

func TestListFailedMessages_StoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	baseStore := storage.New(db)
	if err := baseStore.Init(t.Context()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	store := &serverStoreStub{
		Store: baseStore,
		listFailedMessagesFunc: func(ctx context.Context, userID int64) ([]model.LLMFailedMessage, error) {
			return nil, errors.New("db down")
		},
	}

	tokens := auth.NewTokenStore(10 * time.Minute)
	token, _, err := tokens.IssueForUser(1001)
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}

	s := New(config.Config{}, store, tokens, nil, zap.NewNop())
	r := gin.New()
	s.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/failed-messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body=%s", w.Code, w.Body.String())
	}
}
