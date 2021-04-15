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

//type SomeTypeWithNestedType struct {
//	ObjectMeta
//	Nested Object
//	Value int
//}
//
//type WrapperType struct {
//	ObjectMeta
//	Nested Object
//	Value int
//}

func TestJsonDecoder(t *testing.T) {
	knownRegistry := scheme.NewKnownTypesRegistry()
	decoder := NewJsonMarshaller(knownRegistry)
	knownRegistry.AddKnownTypes(group, &SomeTestType{})

	t.Run("encode and decode obj with specified GK", func(t *testing.T) {
		instance := &SomeTestType{
			ObjectMeta: ObjectMeta{
				TypeMeta: scheme.TypeMeta{
					Kind:  "SomeTestType",
					Group: group.String(),
				},
			},
			A:          1,
		}

		marshaled, err := decoder.Marshal(instance)
		require.NoError(t, err)

		decodedObj, err := decoder.Unmarshal(marshaled)
		require.NoError(t, err)
		require.NotNil(t, decodedObj)
		assert.IsType(t, &SomeTestType{}, decodedObj)
		assert.EqualValues(t, instance, decodedObj)
	})

	t.Run("verify that GK is set from schema before encoding", func(t *testing.T) {
		knownRegistry.AddKnownTypes(group, &SomeTestType{})
		instance := &SomeTestType{
			A:          1,
		}
		marshaled, err := decoder.Marshal(instance)
		require.NoError(t, err)

		decodedObj, err := decoder.Unmarshal(marshaled)
		require.NoError(t, err)
		require.NotNil(t, decodedObj)
		assert.IsType(t, &SomeTestType{}, decodedObj)
		assert.Equal(t, instance.A, instance.A)
	})
	
	t.Run("decode invalid payload with empty GK", func(t *testing.T) {
		instance := &SomeTestType{
			ObjectMeta: ObjectMeta{
				TypeMeta: scheme.TypeMeta{
					Kind:  "",
					Group: group.String(),
				},
			},
			A:          1,
		}

		marshaled, err := json.Marshal(instance)
		require.NoError(t, err)

		decodedObj, err := decoder.Unmarshal(marshaled)
		require.Error(t, err)
		require.Nil(t, decodedObj)
		assert.Equal(t, "creating instance of object for test.: type test. is not registered in KnownTypes", err.Error())
	})

	t.Run("decode nil payload", func(t *testing.T) {
		decodedObj, err := decoder.Unmarshal(nil)
		require.EqualError(t, err, "unexpected end of JSON input")
		require.Nil(t, decodedObj)
	})

	//t.Run("encode and decode type with another nested object", func(t *testing.T) {
	//	knownRegistry.AddKnownTypes(group, &SomeTestType{})
	//	knownRegistry.AddKnownTypes(group, &SomeTypeWithNestedType{})
	//	knownRegistry.AddKnownTypes(group, &WrapperType{})
	//	instance := &WrapperType{
	//		Nested: &SomeTypeWithNestedType{
	//			Nested:     &SomeTestType{
	//				A: 1,
	//			},
	//			Value: 2,
	//		},
	//		Value: 3,
	//	}
	//
	//	marshaled, err := decoder.Marshal(instance)
	//	require.NoError(t, err)
	//	assert.NotEmpty(t, marshaled)
	//
	//	decodedObj, err := decoder.Unmarshal(marshaled)
	//	require.NoError(t, err)
	//	assert.EqualValues(t, instance, decodedObj)
	//})
}

