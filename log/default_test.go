package log

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestDefaultLogger(t *testing.T) {
	output := bytes.NewBuffer(nil)
	l := DefaultLogger(output)
	logger, ok := l.(*defaultLogger)
	require.True(t, ok)

	t.Run("panic", func(t *testing.T) {
		defer output.Reset()

		assert.PanicsWithValue(t, "paaaaaanic", func() {
			logger.Log(PanicLevel, "paaaaaanic")
		})
	})

	t.Run("default info level", func(t *testing.T) {
		defer output.Reset()

		logger.Log(DebugLevel, "debug")
		logger.Log(TraceLevel, "trace")
		assert.Empty(t, output.String())
	})

	t.Run("set level", func(t *testing.T) {
		defer output.Reset()

		l.SetLevel(DebugLevel)
		logger.Log(DebugLevel, "debug")

		assert.Contains(t, output.String(), "debug")
	})

	t.Run("logf", func(t *testing.T) {
		defer output.Reset()

		logger.Logf(WarnLevel, "%s", "someinfo")
		assert.Contains(t, output.String(), "warn [someinfo]")
	})

	t.Run("with fields", func(t *testing.T) {
		defer output.Reset()

		l.SetLevel(DebugLevel)

		fieldsLogger := logger.WithFields([]Field{
			{
				Name: "key",
				Val:  "val",
			},
		})

		fieldsLogger.Log(DebugLevel, "some debug")
		assert.Contains(t, output.String(), "debug [key=val]  [some debug]")
	})

}
