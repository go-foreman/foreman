package log

import (
	"fmt"
	"log"
	"os"
	"sync"
)

//DefaultLogger returns an implementation of logger for MessageBus, used by default if other isn't specified
func DefaultLogger() Logger {
	l := &defaultLogger{mutex: &sync.Mutex{}}
	l.internalLogger = l.createInternalLogger()
	return l
}

type defaultLogger struct {
	mutex          *sync.Mutex
	internalLogger *log.Logger
	level          Level
	fields         []Field
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

func (l *defaultLogger) WithFields(fields []Field) Logger {
	newLogger := &defaultLogger{fields: append(l.fields, fields...), mutex: &sync.Mutex{}, level: l.level}
	newLogger.internalLogger = newLogger.createInternalLogger()

	newLogger.internalLogger.SetPrefix(newLogger.generatePrefix())

	return newLogger
}

func (l defaultLogger) Logf(level Level, format string, args ...interface{}) {
	l.Log(level, fmt.Sprintf(format, args...))
}

func (l *defaultLogger) SetLevel(level Level) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.level = level
	l.internalLogger.SetPrefix(l.generatePrefix())
}

func (l *defaultLogger) generatePrefix() string {
	prefix := levelNames[l.level] + " "

	for _, f := range l.fields {
		prefix += fmt.Sprintf("[%s=%s] ", f.Name, f.Val)
	}

	return prefix
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
