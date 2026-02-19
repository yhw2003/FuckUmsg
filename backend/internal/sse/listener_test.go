package sse

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"chat-assist-backend/internal/config"
	"chat-assist-backend/internal/llm"
	"chat-assist-backend/internal/onebot"
	"chat-assist-backend/internal/storage"
)

func TestFinalizeDateDecision_MatchedOnFirstExtract(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	matched := time.Date(2026, 2, 21, 0, 0, 0, 0, loc).Unix()

	final, llmDate, classic, conflict, note := finalizeDateDecision(matched, matched, false, 0, loc)
	if final != matched {
		t.Fatalf("expected final=%d, got %d", matched, final)
	}
	if llmDate != matched || classic != matched {
		t.Fatalf("unexpected candidates llm=%d classic=%d", llmDate, classic)
	}
	if conflict {
		t.Fatalf("expected no conflict")
	}
	if note != "" {
		t.Fatalf("expected empty note, got %q", note)
	}
}

func TestFinalizeDateDecision_MatchedAfterRetry(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	classic := time.Date(2026, 2, 21, 0, 0, 0, 0, loc).Unix()
	firstLLM := time.Date(2026, 2, 22, 0, 0, 0, 0, loc).Unix()

	final, llmDate, gotClassic, conflict, note := finalizeDateDecision(classic, firstLLM, true, classic, loc)
	if final != classic {
		t.Fatalf("expected final=%d, got %d", classic, final)
	}
	if llmDate != classic || gotClassic != classic {
		t.Fatalf("unexpected candidates llm=%d classic=%d", llmDate, gotClassic)
	}
	if conflict {
		t.Fatalf("expected no conflict")
	}
	if note != "" {
		t.Fatalf("expected empty note, got %q", note)
	}
}

func TestFinalizeDateDecision_ConflictAfterRetry(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	classic := time.Date(2026, 2, 21, 0, 0, 0, 0, loc).Unix()
	llmRetry := time.Date(2026, 2, 23, 0, 0, 0, 0, loc).Unix()

	final, llmDate, gotClassic, conflict, note := finalizeDateDecision(classic, 0, true, llmRetry, loc)
	if final != 0 {
		t.Fatalf("expected final=0 on conflict, got %d", final)
	}
	if llmDate != llmRetry || gotClassic != classic {
		t.Fatalf("unexpected candidates llm=%d classic=%d", llmDate, gotClassic)
	}
	if !conflict {
		t.Fatalf("expected conflict")
	}
	expectedNote := "未能确认的日期：2月23日(llm) 或 2月21日(classic)"
	if note != expectedNote {
		t.Fatalf("expected note %q, got %q", expectedNote, note)
	}
}

type fakeLLM struct {
	classifyResult bool
	classifyErr    error
	summarizeTitle string
	summarizeDetail string
	summarizeErr   error
}

func (f *fakeLLM) Enabled() bool { return true }

func (f *fakeLLM) ClassifyTodo(ctx context.Context, input string) (bool, error) {
	return f.classifyResult, f.classifyErr
}

func (f *fakeLLM) SummarizeTodo(ctx context.Context, input string) (string, string, error) {
	return f.summarizeTitle, f.summarizeDetail, f.summarizeErr
}

func (f *fakeLLM) ExtractRelativeDate(ctx context.Context, input string, nowContext string) (llm.RelativeDateInfo, error) {
	return llm.RelativeDateInfo{}, nil
}

func (f *fakeLLM) RetryRelativeDate(ctx context.Context, input string, nowContext string, classicDate int64, previousLLM llm.RelativeDateInfo) (llm.RelativeDateInfo, error) {
	return llm.RelativeDateInfo{}, nil
}

type fakeStore struct {
	createdTodos   []storage.TodoInput
	createdFaileds []storage.LLMFailedMessageInput
}

func (s *fakeStore) Create(ctx context.Context, input storage.TodoInput) (int64, error) {
	s.createdTodos = append(s.createdTodos, input)
	return int64(len(s.createdTodos)), nil
}

func (s *fakeStore) CreateLLMFailedMessage(ctx context.Context, input storage.LLMFailedMessageInput) (int64, error) {
	s.createdFaileds = append(s.createdFaileds, input)
	return int64(len(s.createdFaileds)), nil
}

func TestProcessEvent_PersistFailedMessageOnClassifyError(t *testing.T) {
	store := &fakeStore{}
	listener := NewListener(config.Config{}, 0, store, &fakeLLM{classifyErr: errors.New("classify boom")}, zap.NewNop())
	event := onebot.Event{
		PostType:    "message",
		MessageType: "private",
		Time:        1700000001,
		MessageID:   "m-classify",
		RawMessage:  "提醒我明天开会",
		UserID:      12345,
	}

	listener.processEvent(context.Background(), event)

	if len(store.createdTodos) != 0 {
		t.Fatalf("expected no todo created, got %d", len(store.createdTodos))
	}
	if len(store.createdFaileds) != 1 {
		t.Fatalf("expected 1 failed message, got %d", len(store.createdFaileds))
	}
	failed := store.createdFaileds[0]
	if failed.UserID != 12345 || failed.SourceType != "private" || failed.SourceID != "12345" {
		t.Fatalf("unexpected source fields: %+v", failed)
	}
	if failed.MessageID != "m-classify" || failed.RawMessage != "提醒我明天开会" {
		t.Fatalf("unexpected message fields: %+v", failed)
	}
	if failed.FailStage != "classify" {
		t.Fatalf("expected fail_stage classify, got %s", failed.FailStage)
	}
	if failed.ErrorText != "classify boom" {
		t.Fatalf("expected error text classify boom, got %s", failed.ErrorText)
	}
}

func TestProcessEvent_PersistFailedMessageOnSummarizeError(t *testing.T) {
	store := &fakeStore{}
	listener := NewListener(config.Config{}, 0, store, &fakeLLM{
		classifyResult: true,
		summarizeErr:   errors.New("summarize boom"),
	}, zap.NewNop())
	event := onebot.Event{
		PostType:    "message",
		MessageType: "group",
		GroupID:     98765,
		Time:        1700000002,
		MessageID:   "m-summarize",
		RawMessage:  "下周一交报告",
		UserID:      22334,
	}

	listener.processEvent(context.Background(), event)

	if len(store.createdTodos) != 0 {
		t.Fatalf("expected no todo created, got %d", len(store.createdTodos))
	}
	if len(store.createdFaileds) != 1 {
		t.Fatalf("expected 1 failed message, got %d", len(store.createdFaileds))
	}
	failed := store.createdFaileds[0]
	if failed.UserID != 22334 || failed.SourceType != "group" || failed.SourceID != "98765" {
		t.Fatalf("unexpected source fields: %+v", failed)
	}
	if failed.MessageID != "m-summarize" || failed.RawMessage != "下周一交报告" {
		t.Fatalf("unexpected message fields: %+v", failed)
	}
	if failed.FailStage != "summarize" {
		t.Fatalf("expected fail_stage summarize, got %s", failed.FailStage)
	}
	if failed.ErrorText != "summarize boom" {
		t.Fatalf("expected error text summarize boom, got %s", failed.ErrorText)
	}
}
