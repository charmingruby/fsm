package fsm

// Option configures an FSM.
type Option[T any] func(*FSM[T])

// WithMaxHops sets the maximum number of hops.
func WithMaxHops[T any](n int) Option[T] {
	return func(f *FSM[T]) {
		f.maxHops = n
	}
}

// WithStorage sets the state storage.
func WithStorage[T any](stateStorage StateStorage) Option[T] {
	return func(f *FSM[T]) {
		f.stateStorage = stateStorage
	}
}
