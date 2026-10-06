package fsm

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultMaxHops = 48
)

// FSM is a generic and type-safe finite state machine.
type FSM[T any] struct {
	globalHooks   GlobalHooks[T]
	logger        Logger
	validationErr error
	terminals     map[State]bool
	states        map[State]state[T]
	fallbacks     map[State]State
	initial       State
	maxHops       int
	validated     bool
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
// Invalid registrations (empty name, nil handler) are reported by Validate, not here.
func (f *FSM[T]) On(stateName State, fn StateFunc[T], hooks ...StateHooks[T]) *FSM[T] {
	f.states[stateName] = state[T]{
		handler: fn,
		hooks:   hooks,
		key:     stateName,
	}

	f.invalidate()

	return f
}

// OnFail sets fallback for state when its StateFunc fails and returns f for chaining.
// Invalid mappings (empty states, self fallback, unknown states) are reported by Validate.
func (f *FSM[T]) OnFail(state State, fallback State) *FSM[T] {
	f.fallbacks[state] = fallback

	f.invalidate()

	return f
}

// Terminal marks states as terminal and returns f for chaining.
// An empty state name is reported by Validate.
func (f *FSM[T]) Terminal(states ...State) *FSM[T] {
	for _, s := range states {
		f.terminals[s] = true
	}

	f.invalidate()

	return f
}

// Run runs handlers from the initial state until a terminal, error, or hop limit. Hooks only observe.
// Run validates first, reusing a cached Validate result when present, and returns
// the validation error with an empty trace for a misconfigured machine.
func (f *FSM[T]) Run(ctx context.Context, data *T) ([]Transition, error) {
	trace := make([]Transition, 0, 8)
	currStateKey := f.initial

	if !f.validated {
		if err := f.Validate(); err != nil {
			return trace, err
		}
	}

	if f.validationErr != nil {
		return trace, f.validationErr
	}

	if f.terminals[currStateKey] {
		return trace, nil
	}

	f.logger.Infof("run started at %q (maxHops=%d)", f.initial, f.maxHops)

	for range f.maxHops {
		if ctx.Err() != nil {
			err := fmt.Errorf("context canceled at %q: %w", currStateKey, ctx.Err())
			f.logger.Errorf("%v", err)

			return trace, err
		}

		f.triggerOnEnterHooks(ctx, data, currStateKey)

		currState, ok := f.states[currStateKey]
		if !ok {
			err := fmt.Errorf("%w: %q", ErrNoTransition, currStateKey)
			f.logger.Errorf("%v", err)

			return trace, err
		}

		hop := f.exec(ctx, data, currState)

		trace = append(trace, hop)

		if hop.Err != nil {
			fallbackStateKey, err := f.resolveFallback(&hop, currState)
			if err != nil {
				err := fmt.Errorf("state %q failed: %w", currState.key, err)
				f.logger.Errorf("%v", err)

				return trace, err
			}

			f.logger.Errorf("state %q failed: %v, falling back to %q", currState.key, hop.Err, fallbackStateKey)

			trace[len(trace)-1] = hop
			currStateKey = fallbackStateKey

			continue
		}

		f.triggerOnTransitionHooks(ctx, data, currState, hop)

		f.logger.Infof("transition: from %q -> %q (%s)", currStateKey, hop.To, hop.Duration)

		f.triggerOnExitHooks(ctx, data, hop)

		if f.terminals[hop.To] {
			f.logger.Infof("reached terminal %q after %d hops", hop.To, len(trace))

			return trace, nil
		}

		currStateKey = hop.To
	}

	err := fmt.Errorf("%w: exceeded %d hops starting at %q", ErrMaxHops, f.maxHops, f.initial)

	f.logger.Errorf("%v", err)

	return trace, err
}

func (f *FSM[T]) exec(ctx context.Context, data *T, currState state[T]) Transition {
	now := time.Now()
	next, err := currState.handler(ctx, data)

	return Transition{
		From:     currState.key,
		To:       next,
		Duration: time.Since(now),
		Err:      err,
	}
}

func (f *FSM[T]) resolveFallback(hop *Transition, currState state[T]) (State, error) {
	fallbackStateKey, hasFallback := f.fallbacks[currState.key]
	if !hasFallback {
		return EmptyState, fmt.Errorf("%w: %w", ErrNoFallback, hop.Err)
	}

	hop.FallbackUsed = fallbackStateKey

	return fallbackStateKey, nil
}

func (f *FSM[T]) triggerOnEnterHooks(ctx context.Context, data *T, currState State) {
	if f.globalHooks.OnEnter != nil {
		f.globalHooks.OnEnter(ctx, data, currState)
	}
}

func (f *FSM[T]) triggerOnExitHooks(ctx context.Context, data *T, hop Transition) {
	if f.globalHooks.OnExit != nil {
		f.globalHooks.OnExit(ctx, data, hop.From, hop.To)
	}
}

func (f *FSM[T]) triggerOnTransitionHooks(ctx context.Context, data *T, currState state[T], hop Transition) {
	if f.globalHooks.OnTransition != nil {
		f.globalHooks.OnTransition(ctx, data, hop)
	}

	for _, hk := range currState.hooks {
		if hk.OnTransition != nil {
			hk.OnTransition(ctx, data, hop)
		}
	}
}
