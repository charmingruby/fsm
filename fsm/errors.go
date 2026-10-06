package fsm

import "errors"

var (
	// ErrNoTransition is returned when no handler is registered for the current state.
	ErrNoTransition = errors.New("no transition registered for state")
	// ErrNoFallback is returned when a state fails and no fallback is registered for it.
	ErrNoFallback = errors.New("no fallback registered for state")
	// ErrMaxHops is returned when the machine exceeds the configured hop limit.
	ErrMaxHops = errors.New("max hops exceeded")
	// ErrEmptyState is returned when an empty state name is used.
	ErrEmptyState = errors.New("empty state name")
	// ErrNilHandler is returned when a state has no handler.
	ErrNilHandler = errors.New("nil handler")
	// ErrInvalidInitial is returned when the initial state is empty.
	ErrInvalidInitial = errors.New("invalid initial state")
	// ErrInvalidMaxHops is returned when the hop budget is not positive.
	ErrInvalidMaxHops = errors.New("invalid max hops")
	// ErrInvalidFallback is returned when a fallback mapping is self-referential.
	ErrInvalidFallback = errors.New("invalid fallback")
	// ErrNoTerminal is returned when no terminal state is declared.
	ErrNoTerminal = errors.New("no terminal state declared")
)
