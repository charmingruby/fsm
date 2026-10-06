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
//	trace, err := f.Run(ctx, data)
//
// Registration never panics: misconfiguration (empty state names, nil
// handlers, unknown fallback states, self fallbacks, non-positive hop
// budget, missing terminals) is reported by Validate as an error
// matching ErrInvalidFSM joined with the specific causes.
//
// Validate caches its result, so a machine can be validated once and
// Run many times without revalidating. Any registration (On, OnFail,
// Terminal) invalidates the cache, and Run validates lazily when no
// cached result exists. A misconfigured machine makes Run return the
// cached validation error with an empty trace.
package fsm
