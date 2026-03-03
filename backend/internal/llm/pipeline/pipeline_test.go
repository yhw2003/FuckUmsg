package pipeline

import (
	"context"
	"errors"
	"testing"
)

type pipelineTestState struct {
	Ran []string
}

type recordStep struct {
	name string
	err  error
}

func (s recordStep) Name() string { return s.name }

func (s recordStep) Run(ctx context.Context, state *pipelineTestState) (any, error) {
	state.Ran = append(state.Ran, s.name)
	return nil, s.err
}

func TestPipeline_StopOnSuccessDecision(t *testing.T) {
	p := Create[pipelineTestState]()
	p.AppendStep(recordStep{name: "a"}, func(ctx context.Context, step string, state *pipelineTestState, out any) Decision {
		return DecisionStop
	}, nil)
	p.AppendStep(recordStep{name: "b"}, nil, nil)

	var state pipelineTestState
	if err := p.Run(context.Background(), &state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.Ran) != 1 || state.Ran[0] != "a" {
		t.Fatalf("unexpected ran steps: %+v", state.Ran)
	}
}

func TestPipeline_ContinueAfterFailureDecision(t *testing.T) {
	p := Create[pipelineTestState]()
	p.AppendStep(recordStep{name: "a", err: errors.New("boom")}, nil, func(ctx context.Context, step string, state *pipelineTestState, err error) Decision {
		return DecisionContinue
	})
	p.AppendStep(recordStep{name: "b"}, nil, nil)

	var state pipelineTestState
	if err := p.Run(context.Background(), &state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.Ran) != 2 || state.Ran[0] != "a" || state.Ran[1] != "b" {
		t.Fatalf("unexpected ran steps: %+v", state.Ran)
	}
}

func TestPipeline_DefaultStopOnFailureWithoutHandler(t *testing.T) {
	p := Create[pipelineTestState]()
	p.AppendStep(recordStep{name: "a", err: errors.New("boom")}, nil, nil)
	p.AppendStep(recordStep{name: "b"}, nil, nil)

	var state pipelineTestState
	if err := p.Run(context.Background(), &state); err == nil {
		t.Fatalf("expected error")
	}
	if len(state.Ran) != 1 || state.Ran[0] != "a" {
		t.Fatalf("unexpected ran steps: %+v", state.Ran)
	}
}
