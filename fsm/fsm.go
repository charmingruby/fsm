package fsm

// FSM is a generic and type-safe finite state machine.
type FSM[T any] struct{}

// New creates a new instance of a finite state machine.
func New[T any]() *FSM[T] {
	fsm := &FSM[T]{}

	return fsm
}
