package status

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/go-foreman/foreman/log"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestResponseWriter(t *testing.T) {
	t.Run("write error message", func(t *testing.T) {
		rw := NewResponseWriterFromErrMsg("lol kek", 403)

		writer := httptest.NewRecorder()
		logger := log.DefaultLogger(io.Discard)

		rw.write(writer, logger)

		assert.Equal(t, `lol kek`, writer.Body.String())
		assert.Equal(t, 403, writer.Code)
	})

	t.Run("write plain error", func(t *testing.T) {
		rw := NewResponseWriterFromError(errors.New("foo bar"))

		writer := httptest.NewRecorder()
		logger := log.DefaultLogger(io.Discard)

		rw.write(writer, logger)

		assert.Equal(t, `foo bar`, writer.Body.String())
		assert.Equal(t, 500, writer.Code)
	})

	t.Run("write response error", func(t *testing.T) {
		rw := NewResponseWriterFromError(NewResponseError(401, errors.New("no access")))

		writer := httptest.NewRecorder()
		logger := log.DefaultLogger(io.Discard)

		rw.write(writer, logger)

		assert.Equal(t, `no access`, writer.Body.String())
		assert.Equal(t, 401, writer.Code)
	})

	t.Run("write object", func(t *testing.T) {
		rw := NewResponseWriter(map[string]string{
			"created": "OK",
		}, 201)

		writer := httptest.NewRecorder()
		logger := log.DefaultLogger(io.Discard)

		rw.write(writer, logger)

		assert.Equal(t, `{"created":"OK"}`, writer.Body.String())
		assert.Equal(t, 201, writer.Code)
	})

	t.Run("write string", func(t *testing.T) {
		rw := NewResponseWriter("my str", 201)

		writer := httptest.NewRecorder()
		logger := log.DefaultLogger(io.Discard)

		rw.write(writer, logger)

		assert.Equal(t, `"my str"`, writer.Body.String())
		assert.Equal(t, 201, writer.Code)
	})

	t.Run("write non-writable", func(t *testing.T) {
		rw := NewResponseWriter(func() {}, 201)

		writer := httptest.NewRecorder()
		logger := log.DefaultLogger(io.Discard)

		rw.write(writer, logger)

		assert.Equal(t, ``, writer.Body.String())
		assert.Equal(t, 500, writer.Code)
	})

}
