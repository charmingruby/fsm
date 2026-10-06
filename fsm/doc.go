// Package fsm provides a generic finite state machine for Go.
//
// It models a system as a finite set of states with transitions
// driven by handlers, rejecting undeclared transitions with an error.
//
// Basic usage: register a handler per state with On, declare fallbacks
// with OnFail, mark end states with Terminal, then Run from an initial
// state until a terminal, an error, or the hop limit:
//
//	type Data struct{ OK bool }
//
//	const (
//		stateA   State = "a"
//		stateB   State = "b"
//		fallback State = "fallback"
//		end      State = "end"
//	)
//
//	f := New(stateA,
//		WithMaxHops[Data](64),
//		WithLogger[Data](NewStdLogger()),
//	)
//	f.On(stateA, func(ctx context.Context, d *Data) (State, error) {
//		if d.OK {
//			return stateB, nil
//		}
//		return EmptyState, errBoom
//	}).
//		OnFail(stateA, fallback).
//		On(fallback, func(ctx context.Context, d *Data) (State, error) {
//			return end, nil
//		}).
//		Terminal(end)
//
//	if err := f.Validate(); err != nil {
//		// handle misconfiguration
//	}
//
//	trace, err := f.Run(ctx, data)
//
// Registration never panics: misconfiguration (empty state names, nil
// handlers, unknown fallback states, self fallbacks, non-positive hop
// budget, missing terminals) is reported by Validate as an error
// joining the specific causes.
//
// Callers must finish all registration, call Validate, and only then
// share the machine and call Run. Run does not call Validate itself.
// Any further registration requires another Validate before the next Run.
//
// An FSM is safe for concurrent Run and Validate calls once construction
// is finished: after construction every operation is read-only.
// Registration (On, OnFail, Terminal) and options must happen before
// Validate and before the machine is shared; they are not safe to call
// concurrently with Run or Validate. Per-Run data is owned by the caller
// and must not be shared mutably across goroutines.
package fsm
