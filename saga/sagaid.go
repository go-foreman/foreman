package saga

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/pkg/errors"
)

const SagaIdKey = "sagaId"

type IdExtractor interface {
	ExtractSagaId(msg *message.Message) (string, error)
}

func NewSagaIdExtractor() IdExtractor {
	return &idHeaderExtractor{}
}

type idHeaderExtractor struct {
}

func (i idHeaderExtractor) ExtractSagaId(msg *message.Message) (string, error) {
	if val, ok := msg.Headers[SagaIdKey]; ok {
		sagaId, converted := val.(string)

		if !converted {
			return "", errors.Errorf("Saga id was found, but has wrong type, should be string")
		}

		return sagaId, nil
	}

	return "", errors.Errorf("Saga id was not found in message's %s headers of type %s", msg.ID, msg.Type)
}
