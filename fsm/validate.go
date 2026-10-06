package fsm

import (
	"errors"
	"fmt"
)

// Validate checks the machine configuration without running it and must be
// called once after construction is finished and before any Run.
// A misconfigured machine returns an error joining the specific causes.
// Any further registration requires a new Validate before the next Run.
//
// Validate performs no writes and is safe for concurrent use with other
// Validate and Run calls.
func (f *FSM[T]) Validate() error {
	return errors.Join(
		f.validateConfig(),
		f.validateStates(),
		f.validateFallbacks(),
	)
}

func (f *FSM[T]) validateConfig() error {
	var errs []error

	if f.initial == EmptyState {
		errs = append(errs, fmt.Errorf("%w: initial state must not be empty", ErrInvalidInitial))
	}

	if f.maxHops <= 0 {
		errs = append(errs, fmt.Errorf("%w: got %d, want > 0", ErrInvalidMaxHops, f.maxHops))
	}

	if len(f.terminals) == 0 {
		errs = append(errs, ErrNoTerminal)
	}

	for name := range f.terminals {
		if name == EmptyState {
			errs = append(errs, fmt.Errorf("%w: terminal must not be empty", ErrEmptyState))
		}
	}

	return errors.Join(errs...)
}

func (f *FSM[T]) validateStates() error {
	var errs []error

	for name, st := range f.states {
		if name == EmptyState {
			errs = append(errs, fmt.Errorf("%w: registered state must not be empty", ErrEmptyState))
		}

		if st.handler == nil {
			errs = append(errs, fmt.Errorf("%w: %q", ErrNilHandler, name))
		}
	}

	if f.initial != EmptyState && !f.terminals[f.initial] {
		if _, ok := f.states[f.initial]; !ok {
			errs = append(errs, fmt.Errorf("%w: %q", ErrNoTransition, f.initial))
		}
	}

	return errors.Join(errs...)
}

func (f *FSM[T]) validateFallbacks() error {
	var errs []error

	for src, dst := range f.fallbacks {
		switch {
		case src == EmptyState || dst == EmptyState:
			errs = append(errs, fmt.Errorf("%w: fallback %q -> %q must not be empty",
				ErrEmptyState, src, dst))
		case src == dst:
			errs = append(errs, fmt.Errorf("%w: fallback %q cannot target itself",
				ErrInvalidFallback, src))
		default:
			if _, ok := f.states[src]; !ok {
				errs = append(errs, fmt.Errorf("%w: fallback source %q has no handler",
					ErrNoTransition, src))
			}

			if _, ok := f.states[dst]; !ok {
				errs = append(errs, fmt.Errorf("%w: fallback target %q has no handler",
					ErrNoTransition, dst))
			}
		}
	}

	return errors.Join(errs...)
}
