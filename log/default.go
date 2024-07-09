package log

import (
	"fmt"
	"io"
	"log"
	"sync"
)

// DefaultLogger returns an implementation of logger for MessageBus, used by default if other isn't specified
func DefaultLogger(out io.Writer) Logger {
	l := &defaultLogger{mutex: &sync.Mutex{}, out: out, level: InfoLevel}
	l.internalLogger = l.createInternalLogger(out)
	return l
}

type defaultLogger struct {
	out               io.Writer
	mutex             *sync.Mutex
	internalLogger    *log.Logger
	level             Level
	fields            []Field
	compiledFieldsStr string
}

func (l defaultLogger) createInternalLogger(out io.Writer) *log.Logger {
	return log.New(out, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
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
		if err := l.internalLogger.Output(3, fmt.Sprint(levelNames[level], l.compiledFieldsStr, " ", v)); err != nil {
			l.internalLogger.Printf("err logging an entry: %s. %s\n", err, v)
		}
	}
}

func (l *defaultLogger) WithFields(fields []Field) Logger {
	newLogger := &defaultLogger{fields: append(l.fields, fields...), mutex: &sync.Mutex{}, level: l.level, out: l.out}
	newLogger.internalLogger = newLogger.createInternalLogger(l.out)

	newLogger.compiledFieldsStr = newLogger.generatePrefix()

	return newLogger
}

func (l defaultLogger) Logf(level Level, format string, args ...interface{}) {
	l.Log(level, fmt.Sprintf(format, args...))
}

func (l *defaultLogger) SetLevel(level Level) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.level = level
}

func (l *defaultLogger) generatePrefix() string {
	prefix := " "

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
