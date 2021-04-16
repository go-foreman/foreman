package saga

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/pkg/errors"
)

const sagaUIDKey = "sagaUID"

type SagaUIDService interface {
	ExtractSagaUID(headers message.Headers) (string, error)
	AddSagaId(headers message.Headers, sagaUID string)
}

func NewSagaUIDService() SagaUIDService {
	return &sagaUIDService{}
}

type sagaUIDService struct {
}

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

func (i sagaUIDService) AddSagaId(headers message.Headers, sagaUID string) {
	headers[sagaUIDKey] = sagaUID
}
