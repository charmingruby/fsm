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

type state[T any] struct {
	handler StateFunc[T]
	hooks   *StateHooks[T]
	key     State
}

// Transition describes a single state change.
type Transition struct {
	Err      error
	From     State
	To       State
	Duration time.Duration
}

// FSM is a generic and type-safe finite state machine.
type FSM[T any] struct {
	logger      Logger
	globalHooks GlobalHooks[T]
	terminals   map[State]bool
	states      map[State]state[T]
	fallbacks   map[State]State
	initial     State
	maxHops     int
}

// New creates an FSM starting at initial.
func New[T any](initial State, opts ...Option[T]) *FSM[T] {
	fsm := &FSM[T]{
		logger:      NewNoopLogger(),
		globalHooks: GlobalHooks[T]{},
		maxHops:     defaultMaxHops,
		initial:     initial,
		terminals:   make(map[State]bool),
		states:      make(map[State]state[T]),
		fallbacks:   make(map[State]State),
	}

	for _, opt := range opts {
		opt(fsm)
	}

	return fsm
}

// On registers fn for state with optional per-state hooks and returns f for chaining.
// A nil hooks is valid and disables per-state observation for that state.
func (f *FSM[T]) On(stateName State, fn StateFunc[T], hooks *StateHooks[T]) *FSM[T] {
	f.states[stateName] = state[T]{
		handler: fn,
		hooks:   hooks,
		key:     stateName,
	}

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

// Run runs handlers from the initial state until a terminal, error, or hop limit. Hooks only observe.
func (f *FSM[T]) Run(ctx context.Context, data *T) ([]Transition, error) {
	trace := make([]Transition, 0, 8)
	curr := f.initial

	if f.terminals[curr] {
		return trace, nil
	}

	f.logger.Infof("run started at %q (maxHops=%d)", f.initial, f.maxHops)

	for range f.maxHops {
		if ctx.Err() != nil {
			err := fmt.Errorf("context canceled at %q: %w", curr, ctx.Err())
			f.logger.Errorf("%v", err)

			return trace, err
		}

		if f.globalHooks.OnEnter != nil {
			f.globalHooks.OnEnter(ctx, data, curr)
		}

		state, ok := f.states[curr]
		if !ok {
			err := fmt.Errorf("%w: %q", ErrNoTransition, curr)
			f.logger.Errorf("%v", err)

			return trace, err
		}

		now := time.Now()
		next, err := state.handler(ctx, data)
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

		if f.globalHooks.OnTransition != nil {
			f.globalHooks.OnTransition(ctx, data, hop)
		}

		if state.hooks != nil && state.hooks.OnTransition != nil {
			state.hooks.OnTransition(ctx, data, hop)
		}

		if f.globalHooks.OnExit != nil {
			f.globalHooks.OnExit(ctx, data, curr, next)
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
