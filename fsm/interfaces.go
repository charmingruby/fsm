package fsm

// StateStorage stores the FSM state.
type StateStorage interface {
	GetState()
	SetState()
}
