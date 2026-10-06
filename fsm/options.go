package fsm

// Option configures an FSM. Options apply during New and must not be used
// after the machine is shared.
type Option[T any] func(*FSM[T])

// WithMaxHops sets the maximum number of hops per Run.
// The default is 48. A non-positive value is rejected by Validate.
func WithMaxHops[T any](n int) Option[T] {
	return func(f *FSM[T]) {
		f.maxHops = n
	}
}

// WithLogger sets the logger used for transitions and failures.
// A nil logger is ignored and keeps the default noop logger.
func WithLogger[T any](logger Logger) Option[T] {
	return func(f *FSM[T]) {
		if logger != nil {
			f.logger = logger
		}
	}
}

// WithGlobalHooks sets observer hooks for every state.
// Hooks only observe; they never affect Run's result.
func WithGlobalHooks[T any](hooks GlobalHooks[T]) Option[T] {
	return func(f *FSM[T]) {
		f.globalHooks = hooks
	}
}
