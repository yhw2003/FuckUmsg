package todoflow

import (
	"context"
	"time"

	"go.uber.org/zap"

	"chat-assist-backend/internal/llm"
	"chat-assist-backend/internal/llm/pipeline"
	"chat-assist-backend/internal/timeparse"
)

type Client interface {
	ClassifyTodo(ctx context.Context, input string) (bool, error)
	SummarizeTodo(ctx context.Context, input string) (string, string, error)
	ExtractRelativeDate(ctx context.Context, input string, nowContext string) (llm.RelativeDateInfo, error)
	RetryRelativeDate(ctx context.Context, input string, nowContext string, classicDate int64, previousLLM llm.RelativeDateInfo) (llm.RelativeDateInfo, error)
}

type Input struct {
	PromptInput string
	MessageText string
	NowContext  string
	Now         time.Time
	WeekMode    string
	Week1Monday string
	Location    *time.Location
}

type Result struct {
	IsTodo      bool
	Title       string
	Detail      string
	ClassicDate int64
	LLMDate1    int64
	LLMDate2    int64
	HasRetry    bool
}

type Hooks struct {
	OnFailed func(stage string, err error)
}

type state struct {
	Input  Input
	Result Result

	dateCtx   timeparse.Context
	relative1 llm.RelativeDateInfo
	relative2 llm.RelativeDateInfo
}

func Run(ctx context.Context, client Client, input Input, hooks Hooks, logger *zap.Logger) (Result, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	s := state{
		Input: input,
		dateCtx: timeparse.Context{
			Now:         input.Now,
			WeekMode:    input.WeekMode,
			Week1Monday: input.Week1Monday,
			Location:    input.Location,
		},
	}

	pipe := pipeline.Create[state]()
	pipe.AppendStep(
		classifyStep{client: client},
		func(ctx context.Context, _ string, s *state, _ any) pipeline.Decision {
			if !s.Result.IsTodo {
				return pipeline.DecisionStop
			}
			return pipeline.DecisionContinue
		},
		func(ctx context.Context, _ string, _ *state, err error) pipeline.Decision {
			if hooks.OnFailed != nil {
				hooks.OnFailed("classify", err)
			}
			return pipeline.DecisionStop
		},
	)
	pipe.AppendStep(
		summarizeStep{client: client},
		nil,
		func(ctx context.Context, _ string, _ *state, err error) pipeline.Decision {
			if hooks.OnFailed != nil {
				hooks.OnFailed("summarize", err)
			}
			return pipeline.DecisionStop
		},
	)
	pipe.AppendStep(
		extractRelativeDateStep{client: client},
		nil,
		func(ctx context.Context, _ string, s *state, err error) pipeline.Decision {
			logger.Warn("llm relative date extract failed", zap.Error(err))
			s.relative1 = llm.RelativeDateInfo{}
			return pipeline.DecisionContinue
		},
	)
	pipe.AppendStep(
		computeDatesStep{},
		func(ctx context.Context, _ string, s *state, _ any) pipeline.Decision {
			logger.Debug("date dual-track",
				zap.String("date_stage", "extract"),
				zap.Int64("llm_date_1", s.Result.LLMDate1),
				zap.Int64("classic_date_1", s.Result.ClassicDate),
			)
			if s.Result.LLMDate1 != s.Result.ClassicDate {
				logger.Debug("date dual-track",
					zap.String("date_stage", "compare"),
					zap.Int64("llm_date_1", s.Result.LLMDate1),
					zap.Int64("classic_date_1", s.Result.ClassicDate),
				)
			}
			return pipeline.DecisionContinue
		},
		nil,
	)
	pipe.AppendStep(
		retryRelativeDateStep{client: client},
		func(ctx context.Context, _ string, s *state, _ any) pipeline.Decision {
			if s.Result.HasRetry {
				logger.Debug("date dual-track",
					zap.String("date_stage", "retry"),
					zap.Int64("llm_date_1", s.Result.LLMDate1),
					zap.Int64("classic_date_1", s.Result.ClassicDate),
					zap.Int64("llm_date_2", s.Result.LLMDate2),
				)
			}
			return pipeline.DecisionContinue
		},
		func(ctx context.Context, _ string, _ *state, err error) pipeline.Decision {
			logger.Warn("llm relative date retry failed", zap.Error(err))
			return pipeline.DecisionContinue
		},
	)

	if err := pipe.Run(ctx, &s); err != nil {
		return s.Result, err
	}
	return s.Result, nil
}
