package log

import (
	"fmt"
	"log"
	"os"
)

//DefaultLogger returns an implementation of logger for MessageBus, used by default if other isn't specified
func DefaultLogger() Logger {
	l := &defaultLogger{}
	l.internalLogger = l.createInternalLogger()
	return l
}

type defaultLogger struct {
	internalLogger *log.Logger
	level          Level
}

func (l defaultLogger) createInternalLogger() *log.Logger {
	return log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
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

	if level <= l.level {
		if err := l.internalLogger.Output(3, fmt.Sprint(v...)); err != nil {
			l.internalLogger.Printf("err logging an entry: %s. %s\n", err, v)
		}
	}
}

func (l *defaultLogger) WithFields(fields Fields) Logger {
	newLogger := &defaultLogger{}
	newLogger.createInternalLogger()

	prefix := ""

	for k, v := range fields {
		prefix += fmt.Sprintf("[%s=%s] ", k, v)
	}

	newLogger.internalLogger.SetPrefix(prefix)

	return newLogger
}

func (l defaultLogger) Logf(level Level, template string, args ...interface{}) {
	l.Log(level, fmt.Sprintf(template, args...))
}

func (l *defaultLogger) SetLevel(level Level) {
	l.level = level

	l.internalLogger.SetPrefix(fmt.Sprintf("[messagebus] %s ", levelNames[level]))
}

var levelNames = map[Level]string{
	PanicLevel: "panic",
	FatalLevel: "fatal",
	ErrorLevel: "error",
	WarnLevel:  "warn",
	InfoLevel:  "info",
	DebugLevel: "debug",
	TraceLevel: "trace",
}
