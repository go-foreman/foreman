package log

//NewNilLogger is used mostly in testing, prints nothing
func NewNilLogger() Logger {
	return &nilLogger{}
}

type nilLogger struct {
}

func (n nilLogger) Log(level Level, v ...interface{}) {
}

func (n nilLogger) Logf(level Level, template string, args ...interface{}) {
}

func (n nilLogger) SetLevel(level Level) {
}
