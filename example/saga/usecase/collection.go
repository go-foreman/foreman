package usecase

import (
	"github.com/go-foreman/foreman/pkg/saga"
)

var DefaultSagasCollection = SagasCollection{}

type SagasCollection struct {
	sagas     []saga.Saga
	contracts []interface{}
}

func (c *SagasCollection) AddSaga(s saga.Saga) {
	c.sagas = append(c.sagas, s)
}

func (c *SagasCollection) Sagas() []saga.Saga {
	return c.sagas
}

func (c *SagasCollection) RegisterContracts(p ...interface{}) {
	if len(p) > 0 {
		c.contracts = append(c.contracts, p...)
	}
}

func (c *SagasCollection) Contracts() []interface{} {
	return c.contracts
}
