package fsm

import (
	"errors"
	"fmt"
)

// Validate checks the machine configuration without running it.
// A misconfigured machine returns an error matching ErrInvalidFSM
// joined with the specific causes.
//
// Prefer validating once before running the workflow: the result is
// cached, so repeated Runs reuse it instead of revalidating every
// time. Any registration (On, OnFail, Terminal, WithMaxHops)
// invalidates the cache, and Run validates lazily when no cached
// result exists.
func (f *FSM[T]) Validate() error {
	f.validated = true

	err := errors.Join(
		f.validateConfig(),
		f.validateStates(),
		f.validateFallbacks(),
	)
	if err == nil {
		f.validationErr = nil

		return nil
	}

	f.validationErr = fmt.Errorf("%w: %w", ErrInvalidFSM, err)

	return f.validationErr
}

// invalidate drops a previous validation result after registration changes.
func (f *FSM[T]) invalidate() {
	f.validated = false
	f.validationErr = nil
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
