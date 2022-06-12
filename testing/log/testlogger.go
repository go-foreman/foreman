package log

import (
	"fmt"

	"github.com/go-foreman/foreman/log"
)

//NewNilLogger is used mostly in testing, prints nothing
func NewNilLogger() *testLogger {
	return &testLogger{entriesStore: &entriesStore{}}
}

type entriesStore struct {
	entries []entry
}

type testLogger struct {
	level        log.Level
	fields       log.Fields
	entriesStore *entriesStore
}

type entry struct {
	Msg   string
	Level log.Level
}

func (n *testLogger) Log(level log.Level, v ...interface{}) {
	n.entriesStore.entries = append(n.entriesStore.entries, entry{Msg: fmt.Sprint(v...), Level: level})
}

func (n *testLogger) Logf(level log.Level, template string, args ...interface{}) {
	n.entriesStore.entries = append(n.entriesStore.entries, entry{Msg: fmt.Sprintf(template, args...), Level: level})
}

func (n *testLogger) SetLevel(level log.Level) {
	n.level = level
}

func (n *testLogger) WithFields(fields log.Fields) log.Logger {
	mergedFields := make(log.Fields)

	for k, v := range n.fields {
		mergedFields[k] = v
	}

	for k, v := range fields {
		mergedFields[k] = v
	}

	return &testLogger{
		entriesStore: n.entriesStore,
		level:        n.level,
		fields:       mergedFields,
	}
}

func (n testLogger) Entries() []entry {
	return n.entriesStore.entries
}

func (n testLogger) Messages() []string {
	r := make([]string, len(n.entriesStore.entries))
	for i := range n.entriesStore.entries {
		r[i] = n.entriesStore.entries[i].Msg
	}

	return r
}

func (n testLogger) LastMessage() string {
	if len(n.entriesStore.entries) > 0 {
		return n.entriesStore.entries[len(n.entriesStore.entries)-1].Msg
	}

	return ""
}

func (n *testLogger) Clear() {
	n.entriesStore.entries = make([]entry, 0)
	n.level = log.InfoLevel
	n.fields = nil
}
