package fsm

// StateStore stores the FSM state.
type StateStore interface {
	GetState()
	SetState()
}

// Logger reports FSM transitions and failures.
type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}
