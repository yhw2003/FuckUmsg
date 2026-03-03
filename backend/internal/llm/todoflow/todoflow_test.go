package todoflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"chat-assist-backend/internal/llm"
)

type fakeClient struct {
	classifyOut bool
	classifyErr error

	summarizeTitle  string
	summarizeDetail string
	summarizeErr    error

	classifyCalls  int
	summarizeCalls int
}

func (f *fakeClient) ClassifyTodo(ctx context.Context, input string) (bool, error) {
	f.classifyCalls++
	return f.classifyOut, f.classifyErr
}

func (f *fakeClient) SummarizeTodo(ctx context.Context, input string) (string, string, error) {
	f.summarizeCalls++
	return f.summarizeTitle, f.summarizeDetail, f.summarizeErr
}

func (f *fakeClient) ExtractRelativeDate(ctx context.Context, input string, nowContext string) (llm.RelativeDateInfo, error) {
	return llm.RelativeDateInfo{}, nil
}

func (f *fakeClient) RetryRelativeDate(ctx context.Context, input string, nowContext string, classicDate int64, previousLLM llm.RelativeDateInfo) (llm.RelativeDateInfo, error) {
	return llm.RelativeDateInfo{}, nil
}

func TestRun_StopWhenNotTodo(t *testing.T) {
	client := &fakeClient{classifyOut: false}
	result, err := Run(context.Background(), client, Input{
		PromptInput: "x",
		MessageText: "x",
		NowContext:  "x",
		Now:         time.Now(),
	}, Hooks{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsTodo {
		t.Fatalf("expected not todo")
	}
	if client.classifyCalls != 1 || client.summarizeCalls != 0 {
		t.Fatalf("unexpected calls classify=%d summarize=%d", client.classifyCalls, client.summarizeCalls)
	}
}

func TestRun_ClassifyErrorTriggersHook(t *testing.T) {
	client := &fakeClient{classifyErr: errors.New("classify boom")}
	var stages []string
	_, err := Run(context.Background(), client, Input{
		PromptInput: "x",
		MessageText: "x",
		NowContext:  "x",
		Now:         time.Now(),
	}, Hooks{
		OnFailed: func(stage string, err error) {
			stages = append(stages, stage+":"+err.Error())
		},
	}, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(stages) != 1 || stages[0] != "classify:classify boom" {
		t.Fatalf("unexpected stages: %+v", stages)
	}
}

func TestRun_SummarizeErrorTriggersHook(t *testing.T) {
	client := &fakeClient{
		classifyOut:  true,
		summarizeErr: errors.New("summarize boom"),
	}
	var stages []string
	_, err := Run(context.Background(), client, Input{
		PromptInput: "x",
		MessageText: "x",
		NowContext:  "x",
		Now:         time.Now(),
	}, Hooks{
		OnFailed: func(stage string, err error) {
			stages = append(stages, stage+":"+err.Error())
		},
	}, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(stages) != 1 || stages[0] != "summarize:summarize boom" {
		t.Fatalf("unexpected stages: %+v", stages)
	}
}
