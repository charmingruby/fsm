// Package fsm provides a generic finite state machine for Go.
//
// It models a system as a finite set of states with transitions
// driven by handlers, rejecting undeclared transitions with an error.
//
// Basic usage: register a handler per state with On, declare fallbacks
// with OnFail, mark end states with Terminal, then Run from an initial
// state until a terminal, an error, or the hop limit:
//
//	f := fsm.New(a,
//		fsm.WithMaxHops[Data](64),
//		fsm.WithLogger[Data](fsm.NewStdLogger()),
//	)
//	f.On(a, func(ctx context.Context, d *Data) (fsm.State, error) {
//		if d.OK {
//			return b, nil
//		}
//		return fsm.EmptyState, errBoom
//	}).
//		OnFail(a, fallback).
//		On(fallback, func(ctx context.Context, d *Data) (fsm.State, error) {
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
// Premise: always Validate before Run. Finish all registration, call
// Validate once, then share the machine and Run it as many times as
// needed. Run never validates itself; calling Run without a prior
// successful Validate has undefined behavior — preferably never Run
// without Validate. Any further registration requires a new Validate
// before the next Run.
//
// An FSM is safe for concurrent Run and Validate calls once constructed
// and validated: after construction every operation is read-only.
// Registration (On, OnFail, Terminal) and options must happen before
// Validate and before the machine is shared; they are not safe to call
// concurrently with Run or Validate. Per-Run data is owned by the caller
// and must not be shared mutably across goroutines.
package fsm
