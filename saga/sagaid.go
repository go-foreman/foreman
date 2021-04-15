package saga

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/pkg/errors"
)

const SagaIdKey = "sagaId"

type IdExtractor interface {
	ExtractSagaId(headers message.Headers) (string, error)
}

func NewSagaIdExtractor() IdExtractor {
	return &idHeaderExtractor{}
}

type idHeaderExtractor struct {
}

func (i idHeaderExtractor) ExtractSagaId(headers message.Headers) (string, error) {
	if val, ok := headers[SagaIdKey]; ok {
		sagaId, converted := val.(string)

		if !converted {
			return "", errors.Errorf("Saga uid was found, but has wrong type, should be string")
		}

		return sagaId, nil
	}

	return "", errors.New("saga uid was not found in headers")
}
