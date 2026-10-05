package fsm

import "context"

// OnEnterHook runs before each handler with the current state.
type OnEnterHook[T any] func(ctx context.Context, data *T, state State)

// OnExitHook runs after a successful transition.
type OnExitHook[T any] func(ctx context.Context, data *T, currentState, nextState State)

// OnTransitionHook runs after a successful transition with its hop.
type OnTransitionHook[T any] func(ctx context.Context, data *T, hop Transition)

// GlobalHooks observes every state transition without affecting execution.
// Hook failures must be handled internally and never propagate.
type GlobalHooks[T any] struct {
	// OnEnter runs before each handler with the current state.
	OnEnter OnEnterHook[T]
	// OnExit runs after a successful transition.
	OnExit OnExitHook[T]
	// OnTransition runs after a successful transition with its hop.
	OnTransition OnTransitionHook[T]
}

// StateHooks observes transitions of a single state without affecting execution.
// Hook failures must be handled internally and never propagate.
type StateHooks[T any] struct {
	// OnTransition runs after a successful transition out of the attached state.
	OnTransition OnTransitionHook[T]
}
