package log

import (
	"fmt"
	"log"
	"os"
)

func DefaultLogger() Logger {
	return &defaultLogger{log.New(os.Stdout, "[messagebus] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)}
}

type defaultLogger struct {
	*log.Logger
}

func (l defaultLogger) Log(level Level, v ...interface{}) {
	l.Output(3, fmt.Sprint(v...))
}

func (l defaultLogger) Logf(level Level, template string, args ...interface{}) {
	l.Output(3, fmt.Sprintf(template, args...))
}
