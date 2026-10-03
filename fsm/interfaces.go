package fsm

// StateStore stores the FSM state.
type StateStore interface {
	GetState()
	SetState()
}
