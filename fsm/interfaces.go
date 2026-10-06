package fsm

// Logger reports FSM transitions and failures.
// Implementations must be safe for concurrent use, as Run may run concurrently.
type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}
