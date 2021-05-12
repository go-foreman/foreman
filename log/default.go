package log

import (
	"fmt"
	"log"
	"os"
)

//DefaultLogger returns an implementation of logger for MessageBus, used by default if other isn't specified
func DefaultLogger() Logger {
	return &defaultLogger{internalLogger: log.New(os.Stdout, "[messagebus] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)}
}

type defaultLogger struct {
	internalLogger *log.Logger
}

func (l defaultLogger) Log(level Level, v... interface{}) {
	if level == FatalLevel {
		l.internalLogger.Fatal(v...)
		return
	}
	l.internalLogger.Output(3, fmt.Sprint(v...))
}

func (l defaultLogger) Logf(level Level, template string, args... interface{}) {
	if level == FatalLevel {
		l.internalLogger.Fatalf(template, args...)
		return
	}
	l.internalLogger.Output(3, fmt.Sprintf(template, args...))
}

func (l defaultLogger) SetLevel(level Level) {
}
