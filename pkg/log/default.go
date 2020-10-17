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
	if level == PanicLevel {
		panic(v)
	}
}

func (l defaultLogger) Logf(level Level, template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	l.Output(3, fmt.Sprintf(template, args...))
	if level == PanicLevel {
		panic(msg)
	}
}
