package fsm

import "fmt"

type stdLogger struct{}

func newStdLogger() *stdLogger {
	return &stdLogger{}
}

func (l *stdLogger) Infof(format string, args ...any) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func (l *stdLogger) Errorf(format string, args ...any) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}
