package log

import (
	"fmt"

	"github.com/go-foreman/foreman/log"
)

//NewNilLogger is used mostly in testing, prints nothing
func NewNilLogger() *nilLogger {
	return &nilLogger{}
}

type nilLogger struct {
	entries []entry
	level   log.Level
}

type entry struct {
	Msg   string
	Level log.Level
}

func (n *nilLogger) Log(level log.Level, v ...interface{}) {
	n.entries = append(n.entries, entry{Msg: fmt.Sprint(v...), Level: level})
}

func (n *nilLogger) Logf(level log.Level, template string, args ...interface{}) {
	n.entries = append(n.entries, entry{Msg: fmt.Sprintf(template, args...), Level: level})
}

func (n *nilLogger) SetLevel(level log.Level) {
	n.level = level
}

func (n nilLogger) Entries() []entry {
	return n.entries
}

func (n nilLogger) Messages() []string {
	r := make([]string, len(n.entries))
	for i := range n.entries {
		r[i] = n.entries[i].Msg
	}

	return r
}

func (n nilLogger) LastMessage() string {
	if len(n.entries) > 0 {
		return n.entries[len(n.entries)-1].Msg
	}

	return ""
}

func (n *nilLogger) Clear() {
	n.entries = make([]entry, 0)
	n.level = log.InfoLevel
}
