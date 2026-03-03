package todoflow

import (
	"context"

	"chat-assist-backend/internal/timeparse"
)

type classifyStep struct {
	client Client
}

func (s classifyStep) Name() string { return "classify" }

func (s classifyStep) Run(ctx context.Context, state *state) (any, error) {
	isTodo, err := s.client.ClassifyTodo(ctx, state.Input.PromptInput)
	if err != nil {
		return nil, err
	}
	state.Result.IsTodo = isTodo
	return nil, nil
}

type summarizeStep struct {
	client Client
}

func (s summarizeStep) Name() string { return "summarize" }

func (s summarizeStep) Run(ctx context.Context, state *state) (any, error) {
	title, detail, err := s.client.SummarizeTodo(ctx, state.Input.PromptInput)
	if err != nil {
		return nil, err
	}
	state.Result.Title = title
	state.Result.Detail = detail
	return nil, nil
}

type extractRelativeDateStep struct {
	client Client
}

func (s extractRelativeDateStep) Name() string { return "relative_date_extract" }

func (s extractRelativeDateStep) Run(ctx context.Context, state *state) (any, error) {
	info, err := s.client.ExtractRelativeDate(ctx, state.Input.MessageText, state.Input.NowContext)
	if err != nil {
		return nil, err
	}
	state.relative1 = info
	return nil, nil
}

type computeDatesStep struct{}

func (s computeDatesStep) Name() string { return "compute_dates" }

func (s computeDatesStep) Run(ctx context.Context, state *state) (any, error) {
	state.Result.LLMDate1 = timeparse.ComputeFromRelativeInfo(state.relative1, state.dateCtx).DateAt
	state.Result.ClassicDate = timeparse.ComputeClassicDate(state.Input.MessageText, state.dateCtx).DateAt
	return nil, nil
}

type retryRelativeDateStep struct {
	client Client
}

func (s retryRelativeDateStep) Name() string { return "relative_date_retry" }

func (s retryRelativeDateStep) Run(ctx context.Context, state *state) (any, error) {
	if state.Result.LLMDate1 == state.Result.ClassicDate {
		return nil, nil
	}
	info, err := s.client.RetryRelativeDate(
		ctx,
		state.Input.MessageText,
		state.Input.NowContext,
		state.Result.ClassicDate,
		state.relative1,
	)
	if err != nil {
		return nil, err
	}
	state.Result.HasRetry = true
	state.relative2 = info
	state.Result.LLMDate2 = timeparse.ComputeFromRelativeInfo(info, state.dateCtx).DateAt
	return nil, nil
}
