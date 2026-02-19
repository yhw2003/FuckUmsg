package timeparse

import (
	"testing"
	"time"

	"chat-assist-backend/internal/llm"
)

func TestComputeClassicDate_BasicRelative(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	ctx := Context{Now: time.Date(2026, 2, 19, 10, 0, 0, 0, loc), WeekMode: "academic", Week1Monday: "2026-02-16", Location: loc}

	result := ComputeClassicDate("提醒我后天交报告", ctx)
	if !result.Matched {
		t.Fatalf("expected matched")
	}
	expected := time.Date(2026, 2, 21, 0, 0, 0, 0, loc).Unix()
	if result.DateAt != expected {
		t.Fatalf("expected %d, got %d", expected, result.DateAt)
	}
}

func TestComputeClassicDate_NextWeekday(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	ctx := Context{Now: time.Date(2026, 2, 19, 10, 0, 0, 0, loc), WeekMode: "academic", Week1Monday: "2026-02-16", Location: loc}

	result := ComputeClassicDate("下周五之前完成", ctx)
	if !result.Matched {
		t.Fatalf("expected matched")
	}
	expected := time.Date(2026, 2, 27, 0, 0, 0, 0, loc).Unix()
	if result.DateAt != expected {
		t.Fatalf("expected %d, got %d", expected, result.DateAt)
	}
}

func TestComputeClassicDate_AcademicWeekExpr(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	ctx := Context{Now: time.Date(2026, 3, 1, 9, 0, 0, 0, loc), WeekMode: "academic", Week1Monday: "2026-02-16", Location: loc}

	result := ComputeClassicDate("第13周周二提交", ctx)
	if !result.Matched {
		t.Fatalf("expected matched")
	}
	expected := time.Date(2026, 5, 12, 0, 0, 0, 0, loc).Unix()
	if result.DateAt != expected {
		t.Fatalf("expected %d, got %d", expected, result.DateAt)
	}
}

func TestComputeClassicDate_NaturalWeekExpr(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	ctx := Context{Now: time.Date(2026, 3, 1, 9, 0, 0, 0, loc), WeekMode: "natural", Location: loc}

	result := ComputeClassicDate("第13周周二提交", ctx)
	if !result.Matched {
		t.Fatalf("expected matched")
	}
	expected := time.Date(2026, 3, 24, 0, 0, 0, 0, loc).Unix()
	if result.DateAt != expected {
		t.Fatalf("expected %d, got %d", expected, result.DateAt)
	}
}

func TestComputeFromRelativeInfo(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	ctx := Context{Now: time.Date(2026, 2, 19, 10, 0, 0, 0, loc), WeekMode: "academic", Week1Monday: "2026-02-16", Location: loc}

	result := ComputeFromRelativeInfo(llm.RelativeDateInfo{BaseDate: "2026-02-20", OffsetDays: 2}, ctx)
	if !result.Matched {
		t.Fatalf("expected matched")
	}
	expected := time.Date(2026, 2, 22, 0, 0, 0, 0, loc).Unix()
	if result.DateAt != expected {
		t.Fatalf("expected %d, got %d", expected, result.DateAt)
	}
}
