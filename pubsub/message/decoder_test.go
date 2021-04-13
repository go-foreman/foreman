package message

import (
	"encoding/json"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	group scheme.Group = "test"
)

type SomeTestType struct {
	ObjectMeta
	A int
}

func TestJsonDecoder(t *testing.T) {
	knownRegistry := scheme.NewKnownTypesRegistry()
	decoder := NewJsonDecoder(knownRegistry)
	knownRegistry.AddKnownTypes(group, &SomeTestType{})

	t.Run("decode valid payload", func(t *testing.T) {
		instance := &SomeTestType{
			ObjectMeta: ObjectMeta{
				UID: "some-id",
				TypeMeta: scheme.TypeMeta{
					Kind:  "SomeTestType",
					Group: group.String(),
				},
			},
			A:          1,
		}

		marshaled, err := json.Marshal(instance)
		require.NoError(t, err)

		decodedObj, err := decoder.Decode(marshaled)
		require.NoError(t, err)
		require.NotNil(t, decodedObj)
		assert.IsType(t, &SomeTestType{}, decodedObj)
		assert.NotEmpty(t, decodedObj.GetUID())
		assert.EqualValues(t, instance, decodedObj)
	})
	
	t.Run("decode invalid payload with empty GK", func(t *testing.T) {
		instance := &SomeTestType{
			ObjectMeta: ObjectMeta{
				UID: "some-id",
				TypeMeta: scheme.TypeMeta{
					Kind:  "",
					Group: group.String(),
				},
			},
			A:          1,
		}

		marshaled, err := json.Marshal(instance)
		require.NoError(t, err)

		decodedObj, err := decoder.Decode(marshaled)
		require.Error(t, err)
		require.Nil(t, decodedObj)
		assert.Equal(t, "creating instance of object for test.: type test. is not registered in KnownTypes", err.Error())
	})
}
