package log

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-foreman/foreman/log"
)

//NewNilLogger is used mostly in testing, prints nothing
func NewNilLogger() *testLogger {
	return &testLogger{entriesStore: &entriesStore{}, mutex: &sync.Mutex{}}
}

type entriesStore struct {
	entries []entry
}

type testLogger struct {
	mutex        *sync.Mutex
	level        log.Level
	fields       []log.Field
	entriesStore *entriesStore
}

type entry struct {
	Msg   string
	Level log.Level
}

func (n *testLogger) Log(level log.Level, v ...interface{}) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.entriesStore.entries = append(n.entriesStore.entries, entry{Msg: fmt.Sprint(v...), Level: level})
}

func (n *testLogger) Logf(level log.Level, template string, args ...interface{}) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.entriesStore.entries = append(n.entriesStore.entries, entry{Msg: fmt.Sprintf(template, args...), Level: level})
}

func (n *testLogger) SetLevel(level log.Level) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.level = level
}

func (n *testLogger) WithFields(fields []log.Field) log.Logger {
	return &testLogger{
		entriesStore: n.entriesStore,
		level:        n.level,
		fields:       append(n.fields, fields...),
		mutex:        &sync.Mutex{},
	}
}

func (n testLogger) Entries() []entry {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	return n.entriesStore.entries
}

func (n testLogger) Messages() []string {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	r := make([]string, len(n.entriesStore.entries))
	for i := range n.entriesStore.entries {
		r[i] = n.entriesStore.entries[i].Msg
	}

	return r
}

func (n testLogger) Fields() []log.Field {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	return n.fields
}

func (n testLogger) LastMessage() string {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if len(n.entriesStore.entries) > 0 {
		return n.entriesStore.entries[len(n.entriesStore.entries)-1].Msg
	}

	return ""
}

func (n *testLogger) Clear() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.entriesStore.entries = make([]entry, 0)
	n.level = log.InfoLevel
	n.fields = nil
}

func (n *testLogger) AssertContainsSubstr(t *testing.T, substr string) {
	present := false
	for _, l := range n.Messages() {
		if strings.Contains(l, substr) {
			present = true
			break
		}
	}

	assert.Truef(t, present, "asserting that '%s' was logged", substr)
}
