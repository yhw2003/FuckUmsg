package sse

import (
	"testing"
	"time"
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
