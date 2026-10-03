package fsm

const (
	defaultMaxHops = 48
)

// FSM is a generic and type-safe finite state machine.
type FSM[T any] struct {
	stateStorage StateStorage
	maxHops      int
}

// New creates a new instance of a finite state machine.
func New[T any](opts ...Option[T]) *FSM[T] {
	fsm := &FSM[T]{
		stateStorage: newMemStateStorage(),
		maxHops:      defaultMaxHops,
	}

	for _, opt := range opts {
		opt(fsm)
	}

	return fsm
}
