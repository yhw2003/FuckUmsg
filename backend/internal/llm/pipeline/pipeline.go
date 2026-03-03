package pipeline

import (
	"context"
	"errors"
)

type Decision int

const (
	DecisionContinue Decision = iota
	DecisionStop
)

type Step[S any] interface {
	Name() string
	Run(ctx context.Context, state *S) (any, error)
}

type SuccessFn[S any] func(ctx context.Context, step string, state *S, out any) Decision
type FailFn[S any] func(ctx context.Context, step string, state *S, err error) Decision

type stepEntry[S any] struct {
	step      Step[S]
	onSuccess SuccessFn[S]
	onFail    FailFn[S]
}

type Pipeline[S any] struct {
	steps []stepEntry[S]
}

func Create[S any]() *Pipeline[S] {
	return &Pipeline[S]{}
}

func (p *Pipeline[S]) AppendStep(step Step[S], onSuccess SuccessFn[S], onFail FailFn[S]) *Pipeline[S] {
	p.steps = append(p.steps, stepEntry[S]{
		step:      step,
		onSuccess: onSuccess,
		onFail:    onFail,
	})
	return p
}

func (p *Pipeline[S]) Run(ctx context.Context, state *S) error {
	if state == nil {
		return errors.New("state is nil")
	}
	for _, entry := range p.steps {
		if entry.step == nil {
			return errors.New("step is nil")
		}
		stepName := entry.step.Name()
		out, err := entry.step.Run(ctx, state)
		if err != nil {
			if entry.onFail == nil {
				return err
			}
			if entry.onFail(ctx, stepName, state, err) == DecisionStop {
				return err
			}
			continue
		}
		if entry.onSuccess == nil {
			continue
		}
		if entry.onSuccess(ctx, stepName, state, out) == DecisionStop {
			return nil
		}
	}
	return nil
}
