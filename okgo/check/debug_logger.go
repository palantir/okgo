package check

import "fmt"

type DebugLogger struct {
	debugMode bool
}

func NewDebugLogger(debugMode bool) DebugLogger {
	return DebugLogger{
		debugMode: debugMode,
	}
}

func (d DebugLogger) Log(str string) {
	if d.debugMode {
		fmt.Println(str)
	}
}
