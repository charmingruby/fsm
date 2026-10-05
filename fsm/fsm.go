package fsm

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	defaultMaxHops = 48
)

var (
	// ErrNoTransition is returned when no handler is registered for the current state.
	ErrNoTransition = errors.New("no transition registered for state")
	// ErrMaxHops is returned when the machine exceeds the configured hop limit.
	ErrMaxHops = errors.New("max hops exceeded")
)

// State identifies a node in the state machine.
type State string

// EmptyState represents an empty or unset state.
const EmptyState State = ""

// StateFunc resolves the next State from data.
type StateFunc[T any] func(ctx context.Context, data *T) (State, error)

// Transition describes a single state change.
type Transition struct {
	Err      error
	From     State
	To       State
	Duration time.Duration
}

// FSM is a generic and type-safe finite state machine.
type FSM[T any] struct {
	logger    Logger
	hooks     Hooks[T]
	terminals map[State]bool
	handlers  map[State]StateFunc[T]
	fallbacks map[State]State
	initial   State
	maxHops   int
}

// New creates an FSM starting at initial.
func New[T any](initial State, opts ...Option[T]) *FSM[T] {
	fsm := &FSM[T]{
		logger:    newStdLogger(),
		hooks:     Hooks[T]{},
		maxHops:   defaultMaxHops,
		initial:   initial,
		terminals: make(map[State]bool),
		handlers:  make(map[State]StateFunc[T]),
		fallbacks: make(map[State]State),
	}

	for _, opt := range opts {
		opt(fsm)
	}

	return fsm
}

// On registers fn for state and returns f for chaining.
func (f *FSM[T]) On(state State, fn StateFunc[T]) *FSM[T] {
	f.handlers[state] = fn

	return f
}

// OnFail sets fallback for state when its StateFunc fails and returns f for chaining.
func (f *FSM[T]) OnFail(state State, fallback State) *FSM[T] {
	f.fallbacks[state] = fallback

	return f
}

// Terminal marks states as terminal and returns f for chaining.
func (f *FSM[T]) Terminal(states ...State) *FSM[T] {
	for _, state := range states {
		f.terminals[state] = true
	}

	return f
}

// Trigger runs handlers from the initial state until a terminal, error, or hop limit. Hooks only observe.
func (f *FSM[T]) Trigger(ctx context.Context, data *T) ([]Transition, error) {
	trace := make([]Transition, 0, 8)
	curr := f.initial

	f.logger.Infof("trigger started at %q (maxHops=%d)", f.initial, f.maxHops)

	for range f.maxHops {
		if ctx.Err() != nil {
			err := fmt.Errorf("context canceled at %q: %w", curr, ctx.Err())
			f.logger.Errorf("%v", err)

			return trace, err
		}

		if f.hooks.OnEnter != nil {
			f.hooks.OnEnter(ctx, data, curr)
		}

		stateFn, ok := f.handlers[curr]
		if !ok {
			err := fmt.Errorf("%w: %q", ErrNoTransition, curr)
			f.logger.Errorf("%v", err)

			return trace, err
		}

		now := time.Now()
		next, err := stateFn(ctx, data)
		duration := time.Since(now)

		hop := Transition{
			From:     curr,
			To:       next,
			Duration: duration,
			Err:      err,
		}

		trace = append(trace, hop)

		if err != nil {
			fallback, hasFallback := f.fallbacks[curr]
			if !hasFallback {
				err = fmt.Errorf("state %q: %w", curr, err)
				f.logger.Errorf("%v", err)

				return trace, err
			}

			f.logger.Errorf("state %q failed: %v, falling back to %q", curr, err, fallback)
			curr = fallback

			continue
		}

		f.logger.Infof("transition: from %q -> %q (%s)", curr, next, duration)

		if f.hooks.OnTransition != nil {
			f.hooks.OnTransition(ctx, data, hop)
		}

		if f.hooks.OnExit != nil {
			f.hooks.OnExit(ctx, data, curr, next)
		}

		if f.terminals[next] {
			f.logger.Infof("reached terminal %q after %d hops", next, len(trace))

			return trace, nil
		}

		curr = next
	}

	err := fmt.Errorf("%w: exceeded %d hops starting at %q", ErrMaxHops, f.maxHops, f.initial)

	f.logger.Errorf("%v", err)

	return trace, err
}
