package fsm

import (
	"context"
	"time"
)

const (
	defaultMaxHops = 48
)

// State identifies a node in the state machine.
type State string

// StateFunc resolves the next State from an event.
type StateFunc[T any] func(ctx context.Context, event *T) (State, error)

// Transition describes a single state change.
type Transition struct {
	Err      error
	From     State
	To       State
	Duration time.Duration
}

// FSM is a generic and type-safe finite state machine.
type FSM[T any] struct {
	store     StateStore
	terminals map[State]bool
	handlers  map[State]StateFunc[T]
	fallbacks map[State]State
	initial   State
	maxHops   int
}

// New creates a new instance of a finite state machine.
func New[T any](initial State, opts ...Option[T]) *FSM[T] {
	fsm := &FSM[T]{
		store:     newMemStateStore(),
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

// Trigger advances the machine with event and returns the applied transitions.
func (f *FSM[T]) Trigger(_ context.Context, _ *T) ([]Transition, error) {
	panic("not implemented")
}
