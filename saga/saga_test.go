package saga

import (
	"testing"

	"github.com/go-foreman/foreman/runtime/scheme"

	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
)

func TestBaseSaga(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	schema := scheme.NewKnownTypesRegistry()

	exp := &sagaExample{}
	assert.Empty(t, exp.EventHandlers())

	assert.PanicsWithError(t, "schema wasn't set", func() {
		exp.Init()
	})

	exp.SetSchema(schema)

	assert.PanicsWithError(t, "ev *saga.DataContract is not registered in schema", func() {
		exp.Init()
	})

	g := scheme.Group("someGroup")

	schema.AddKnownTypes(g, &DataContract{})

	exp.Init()

	assert.Len(t, exp.EventHandlers(), 1)
	singleGK := scheme.GroupKind{}
	for k := range exp.EventHandlers() {
		singleGK = k
	}

	assert.Equal(t, scheme.GroupKind{Group: g, Kind: "DataContract"}, singleGK)
}
