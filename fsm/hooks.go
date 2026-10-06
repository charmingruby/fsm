package fsm

import "context"

// OnEnterHook runs before each handler with the current state.
type OnEnterHook[T any] func(ctx context.Context, data *T, state State)

// OnExitHook runs after a successful hop with the states it left and entered.
type OnExitHook[T any] func(ctx context.Context, data *T, currentState, nextState State)

// OnTransitionHook runs after a successful hop with the recorded Transition.
type OnTransitionHook[T any] func(ctx context.Context, data *T, hop Transition)

// GlobalHooks observes every state transition without affecting execution.
// Hooks run synchronously inside Run and must be fast and side-effect free
// beyond observation.
type GlobalHooks[T any] struct {
	OnEnter      OnEnterHook[T]
	OnExit       OnExitHook[T]
	OnTransition OnTransitionHook[T]
}

// StateHooks observes transitions of a single state without affecting execution.
// Only OnEnter and OnTransition apply per state; OnExit is global only.
type StateHooks[T any] struct {
	OnEnter      OnEnterHook[T]
	OnTransition OnTransitionHook[T]
}
