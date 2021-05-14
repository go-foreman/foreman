package saga

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/saga"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type pgStoreTest struct {
	intSuite.PgSuite
}

func TestPGSuite(t *testing.T) {
	pgStoreTest := &pgStoreTest{}
	suite.Run(t, pgStoreTest)
}

func (p *pgStoreTest) TestPGStore() {
	t := p.T()

	schemeRegistry := scheme.NewKnownTypesRegistry()
	schemeRegistry.AddKnownTypes(testGroup, &WorkflowSaga{})
	schemeRegistry.AddKnownTypes(testGroup, &FilterSaga{})
	marshaller := message.NewJsonMarshaller(schemeRegistry)
	pgStore, err := saga.NewSQLSagaStore(p.Connection(), saga.PGDriver, marshaller)

	require.NoError(t, err)
	require.NotNil(t, pgStore)

	testUseCases(t, pgStore, schemeRegistry, p.Connection())
}
