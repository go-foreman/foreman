package saga

import (
	"fmt"
	"testing"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSagaUIDService(t *testing.T) {
	svc := NewSagaUIDService()

	t.Run("add and fetch saga uid", func(t *testing.T) {
		headers := message.Headers{}
		svc.AddSagaId(headers, "xxx")
		extractedUID, err := svc.ExtractSagaUID(headers)
		require.NoError(t, err)
		assert.Equal(t, "xxx", extractedUID)
	})

	t.Run("saga uid not found", func(t *testing.T) {
		headers := message.Headers{}
		extractedUID, err := svc.ExtractSagaUID(headers)
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("saga uid was not found in headers by key %s", sagaUIDKey))
		assert.Empty(t, extractedUID)
	})

	t.Run("saga uid is not string type in headers", func(t *testing.T) {
		headers := message.Headers{}
		headers[sagaUIDKey] = 1
		extractedUID, err := svc.ExtractSagaUID(headers)
		assert.Error(t, err)
		assert.EqualError(t, err, "Saga uid was found, but has wrong type, should be string")
		assert.Empty(t, extractedUID)
	})

	t.Run("set saga id into headers", func(t *testing.T) {
		headers := message.Headers{}
		svc.AddSagaId(headers, "uid")

		assert.Equal(t, headers[sagaUIDKey], "uid")
	})
}
