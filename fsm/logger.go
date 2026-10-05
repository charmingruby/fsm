package fsm

import "fmt"

type stdLogger struct{}

// NewStdLogger reports FSM transitions and failures to stdout.
// Pass it to WithLogger to opt into console output.
// The FSM default logger discards everything.
func NewStdLogger() Logger {
	return &stdLogger{}
}

func (l *stdLogger) Infof(format string, args ...any) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func (l *stdLogger) Errorf(format string, args ...any) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

type noopLogger struct{}

// NewNoopLogger discards all FSM log output.
// It is the default logger used by New.
func NewNoopLogger() Logger {
	return noopLogger{}
}

func (noopLogger) Infof(string, ...any) {}

func (noopLogger) Errorf(string, ...any) {}
