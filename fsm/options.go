package fsm

// Option configures an FSM.
type Option[T any] func(*FSM[T])

// WithMaxHops sets the maximum number of hops.
func WithMaxHops[T any](n int) Option[T] {
	return func(f *FSM[T]) {
		f.maxHops = n
	}
}

// WithStore sets the state storage.
func WithStore[T any](store StateStore) Option[T] {
	return func(f *FSM[T]) {
		f.store = store
	}
}
