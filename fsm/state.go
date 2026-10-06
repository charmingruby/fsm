package fsm

import (
	"context"
	"time"
)

// State identifies a node in the state machine.
type State string

// EmptyState represents an empty or unset state.
const EmptyState State = ""

// StateFunc resolves the next State from data.
type StateFunc[T any] func(ctx context.Context, data *T) (State, error)

type state[T any] struct {
	handler StateFunc[T]
	key     State
	hooks   []StateHooks[T]
}

// Transition describes a single state change.
type Transition struct {
	Err          error
	From         State
	To           State
	FallbackUsed State
	Duration     time.Duration
}
