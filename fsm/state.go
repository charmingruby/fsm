package fsm

import (
	"context"
	"time"
)

// State identifies a node in the state machine.
type State string

// EmptyState is the zero value for State and never names a real state.
// Handlers return it together with a non-nil error to signal failure.
const EmptyState State = ""

// StateFunc resolves the next State from data.
//
// Return the next state and a nil error to continue, or EmptyState and a
// non-nil error to fail the current state and take its OnFail fallback.
// A nil error with a terminal state stops Run successfully.
type StateFunc[T any] func(ctx context.Context, data *T) (State, error)

type state[T any] struct {
	handler StateFunc[T]
	key     State
	hooks   []StateHooks[T]
}

// Transition describes a single state change.
type Transition struct {
	// Err is the handler failure that triggered a fallback, if any.
	Err error
	// From is the state that just ran.
	From State
	// To is the state returned by the handler, or EmptyState on failure.
	To State
	// FallbackUsed is the fallback taken after a failure, if any.
	FallbackUsed State
	// Duration measures the handler execution time.
	Duration time.Duration
}
