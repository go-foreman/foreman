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

func (l defaultLogger) Log(level Level, v ...interface{}) {
	if level == FatalLevel {
		l.internalLogger.Fatal(v...)
		return
	}

	if level == PanicLevel {
		l.internalLogger.Panic(v...)
		return
	}

	if err := l.internalLogger.Output(3, fmt.Sprint(v...)); err != nil {
		l.internalLogger.Println(fmt.Sprintf("err logging an entry: %s. %s", err, v))
	}
}

func (l defaultLogger) Logf(level Level, template string, args ...interface{}) {
	l.Log(level, fmt.Sprintf(template, args...))
}

func (l defaultLogger) SetLevel(level Level) {
}
