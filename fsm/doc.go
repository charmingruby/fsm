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
//	}, nil).
//		OnFail(a, fallback).
//		On(fallback, func(ctx context.Context, d *Data) (fsm.State, error) {
//			return end, nil
//		}, nil).
//		Terminal(end)
//
//	trace, err := f.Run(ctx, data)
//
// The initial state must have an On handler registered, unless it is
// also a Terminal state (then Run returns an empty trace). Any visited
// state without a handler, including a fallback target, fails with
// ErrNoTransition.
//
// Fallback: when a handler fails and OnFail was declared for that state,
// Run records the hop with FallbackUsed set and continues at the fallback
// state. Without a fallback, Run wraps the handler error with ErrNoFallback.
// Hooks (GlobalHooks, StateHooks) only observe successful transitions and
// never affect the result.
package fsm
