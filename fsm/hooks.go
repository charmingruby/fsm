package fsm

import "context"

// OnEnterHook runs before each handler with the current state.
type OnEnterHook[T any] func(ctx context.Context, data *T, state State)

// OnExitHook runs after a successful transition.
type OnExitHook[T any] func(ctx context.Context, data *T, currentState, nextState State)

// OnTransitionHook runs after a successful transition with its hop.
type OnTransitionHook[T any] func(ctx context.Context, data *T, hop Transition)

// GlobalHooks observes every state transition without affecting execution.
type GlobalHooks[T any] struct {
	OnEnter      OnEnterHook[T]
	OnExit       OnExitHook[T]
	OnTransition OnTransitionHook[T]
}

// StateHooks observes transitions of a single state without affecting execution.
type StateHooks[T any] struct {
	OnEnter      OnEnterHook[T]
	OnTransition OnTransitionHook[T]
}
