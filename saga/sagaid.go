package saga

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/pkg/errors"
)

const sagaUIDKey = "sagaUID"

// SagaUIDService manipulates with sagaId in headers
type SagaUIDService interface {
	ExtractSagaUID(headers message.Headers) (string, error)
	AddSagaId(headers message.Headers, sagaUID string)
}

// NewSagaUIDService constructs default implementation of SagaUIDService
func NewSagaUIDService() SagaUIDService {
	return &sagaUIDService{}
}

type sagaUIDService struct {
}

// ExtractSagaUID extracts sagaUID key from headers
func (i sagaUIDService) ExtractSagaUID(headers message.Headers) (string, error) {
	if val, ok := headers[sagaUIDKey]; ok {
		sagaId, converted := val.(string)

		if !converted {
			return "", errors.Errorf("Saga uid was found, but has wrong type, should be string")
		}

		return sagaId, nil
	}

	return "", errors.Errorf("saga uid was not found in headers by key %s", sagaUIDKey)
}

// AddSagaId adds sagaUID to headers
func (i sagaUIDService) AddSagaId(headers message.Headers, sagaUID string) {
	headers[sagaUIDKey] = sagaUID
}
