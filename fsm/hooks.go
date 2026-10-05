package fsm

import "context"

// Hooks observes state transitions without affecting execution.
// Hook failures must be handled internally and never propagate.
type Hooks[T any] struct {
	// OnEnter runs before each handler with the current state.
	OnEnter func(ctx context.Context, data *T, state State)
	// OnExit runs after a successful transition.
	OnExit func(ctx context.Context, data *T, currentState, nextState State)
	// OnTransition runs after a successful transition with its hop.
	OnTransition func(ctx context.Context, data *T, hop Transition)
}
