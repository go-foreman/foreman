package log

import (
	"log"
	"os"
)

func DefaultLogger() Logger {
	return &defaultLogger{logger: log.New(os.Stdout, "[messagebus] ", log.Ldate | log.Ltime | log.Lmicroseconds)}
}

type defaultLogger struct {
	logger *log.Logger
}

func (l defaultLogger) Log(level Level, v interface{}) {
	l.logger.Printf("%s", v)
}

func (l defaultLogger) Logf(level Level, template string, args ...interface{}) {
	l.logger.Printf(template, args...)
}
