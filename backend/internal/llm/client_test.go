package llm

import "testing"

func TestParseClassifyOutput(t *testing.T) {
	v, err := parseClassifyOutput(`{"is_todo": true}`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !v {
		t.Fatalf("expected true")
	}
}

func TestParseSummarizeOutput(t *testing.T) {
	title, detail, err := parseSummarizeOutput(`{"title":"明天提交周报","detail":"请明天提交周报给我"}`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if title != "明天提交周报" {
		t.Fatalf("unexpected title: %q", title)
	}
	if detail != "请明天提交周报给我" {
		t.Fatalf("unexpected detail: %q", detail)
	}
}

func TestParseRelativeDateOutput(t *testing.T) {
	output := `{"base_date":"2026-02-19","offset_days":2,"week_expr":"","resolved_date":"2026-02-21","raw_reasoning_tag":"relative+2"}`
	info, err := parseRelativeDateOutput(output)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if info.BaseDate != "2026-02-19" || info.OffsetDays != 2 || info.ResolvedDate != "2026-02-21" {
		t.Fatalf("unexpected relative info: %+v", info)
	}
}

func TestParseRelativeDateOutput_WithCodeFence(t *testing.T) {
	output := "```json\n{\"base_date\":\"\",\"offset_days\":0,\"week_expr\":\"第13周周二\",\"resolved_date\":\"2026-05-12\",\"raw_reasoning_tag\":\"week_expr\"}\n```"
	info, err := parseRelativeDateOutput(output)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if info.WeekExpr != "第13周周二" {
		t.Fatalf("unexpected week_expr: %q", info.WeekExpr)
	}
}
