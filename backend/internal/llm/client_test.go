package llm

import "testing"

func TestParsePlainTodoOutput_LegacyThreeLines(t *testing.T) {
	text := "IS_TODO: true\nTITLE: 明天提交周报\nDETAIL: 请明天提交周报给我"

	result, err := parsePlainTodoOutput(text)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !result.IsTodo {
		t.Fatalf("expected todo=true")
	}
	if result.Title != "明天提交周报" {
		t.Fatalf("unexpected title: %q", result.Title)
	}
	if result.Detail != "请明天提交周报给我" {
		t.Fatalf("unexpected detail: %q", result.Detail)
	}
	if result.DeadlineAt != 0 {
		t.Fatalf("expected deadline_at=0, got %d", result.DeadlineAt)
	}
}

func TestParsePlainTodoOutput_FourLinesWithDeadline(t *testing.T) {
	text := "IS_TODO: true\nTITLE: 明天14点开会\nDETAIL: 明天14点准时参加会议\nDEADLINE_AT: 1760000000"

	result, err := parsePlainTodoOutput(text)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !result.IsTodo {
		t.Fatalf("expected todo=true")
	}
	if result.DeadlineAt != 1760000000 {
		t.Fatalf("expected deadline_at=1760000000, got %d", result.DeadlineAt)
	}
}

func TestParsePlainTodoOutput_InvalidDeadlineFallback(t *testing.T) {
	text := "IS_TODO: true\nTITLE: 周五前完成\nDETAIL: 本周五之前完成并反馈\nDEADLINE_AT: not-a-number"

	result, err := parsePlainTodoOutput(text)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !result.IsTodo {
		t.Fatalf("expected todo=true")
	}
	if result.DeadlineAt != 0 {
		t.Fatalf("expected deadline_at fallback to 0, got %d", result.DeadlineAt)
	}
}
