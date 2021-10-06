package message

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	group scheme.Group = "test"
)

type WrapperType struct {
	ObjectMeta
	Nested Object
	Value  int
}

type SomeTypeWithNestedType struct {
	ObjectMeta
	Nested Object
	Value  int
}

type SomeTestType struct {
	ObjectMeta
	Value int
	Child ChildType
}

type ChildType struct {
	Value int
}

type WithAnon struct {
	ObjectMeta
	SomeVal int
	ChildType
}

type WithTime struct {
	ObjectMeta
	CreatedAt time.Time
}

type WithAnonAndJsonTag struct {
	ObjectMeta
	SomeVal   int
	ChildType `json:"child"`
}

type WithAnonPointer struct {
	ObjectMeta
	SomeVal int
	*ChildType
}

type WithKindChild struct {
	ObjectMeta
	SomeVal        int
	KindGroupChild KindChild `json:"kind_child"`
}

type KindChild struct {
	Kind string `json:"kind"`
}

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
			Value: 1,
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
			Value: 1,
		}
		marshaled, err := decoder.Marshal(instance)
		require.NoError(t, err)

		decodedObj, err := decoder.Unmarshal(marshaled)
		require.NoError(t, err)
		require.NotNil(t, decodedObj)
		assert.IsType(t, &SomeTestType{}, decodedObj)
		assert.Equal(t, instance.Value, instance.Value)
	})

	t.Run("decode invalid payload with empty GK", func(t *testing.T) {
		instance := &SomeTestType{
			ObjectMeta: ObjectMeta{
				TypeMeta: scheme.TypeMeta{
					Kind:  "",
					Group: group.String(),
				},
			},
			Value: 1,
		}

		marshaled, err := json.Marshal(instance)
		require.NoError(t, err)

		decodedObj, err := decoder.Unmarshal(marshaled)
		require.Error(t, err)
		require.Nil(t, decodedObj)
		assert.True(t, strings.Contains(err.Error(), " is not registered in KnownTypes"))
	})

	t.Run("decode nil payload", func(t *testing.T) {
		decodedObj, err := decoder.Unmarshal(nil)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "unexpected end of JSON input"))
		require.Nil(t, decodedObj)
	})

	t.Run("encode and decode type with another nested object", func(t *testing.T) {
		knownRegistry.AddKnownTypes(group, &SomeTestType{})
		knownRegistry.AddKnownTypes(group, &SomeTypeWithNestedType{})
		knownRegistry.AddKnownTypes(group, &WrapperType{})
		instance := &WrapperType{
			Nested: &SomeTypeWithNestedType{
				Nested: &SomeTestType{
					Value: 1,
					Child: ChildType{
						Value: -1,
					},
				},
				Value: 2,
			},
			Value: 3,
		}

		marshaled, err := decoder.Marshal(instance)
		require.NoError(t, err)
		assert.NotEmpty(t, marshaled)

		decodedObj, err := decoder.Unmarshal(marshaled)
		require.NoError(t, err)
		assert.EqualValues(t, instance, decodedObj)
	})

	t.Run("squashing of an anonymous struct", func(t *testing.T) {
		knownRegistry.AddKnownTypes(group, &WithAnon{})
		instance := &WithAnon{SomeVal: 1, ChildType: ChildType{Value: 2}}

		marshaled, err := decoder.Marshal(instance)
		require.NoError(t, err)
		assert.NotEmpty(t, marshaled)

		decodedObj, err := decoder.Unmarshal(marshaled)
		require.NoError(t, err)
		assert.EqualValues(t, instance, decodedObj)
	})

	t.Run("encode and decode a struct with time", func(t *testing.T) {
		knownRegistry.AddKnownTypes(group, &WithTime{})
		instance := &WithTime{CreatedAt: time.Now()}

		marshaled, err := decoder.Marshal(instance)
		require.NoError(t, err)
		assert.NotEmpty(t, marshaled)

		decodedObj, err := decoder.Unmarshal(marshaled)
		require.NoError(t, err)

		decoded, _ := decodedObj.(*WithTime)

		assert.True(t, instance.CreatedAt.Equal(decoded.CreatedAt))
	})

	// todo FAILING TEST, fix it
	//t.Run("encode and decode a struct with child type that contains a field with name 'kind'", func(t *testing.T) {
	//	knownRegistry.AddKnownTypes(group, &WithKindChild{})
	//	instance := &WithKindChild{SomeVal: 1, KindGroupChild: KindChild{Kind: "some"}}
	//
	//	marshaled, err := decoder.Marshal(instance)
	//	require.NoError(t, err)
	//	assert.NotEmpty(t, marshaled)
	//
	//	decodedObj, err := decoder.Unmarshal(marshaled)
	//	require.NoError(t, err)
	//	assert.EqualValues(t, instance, decodedObj)
	//})

	//@todo these 2 cases should work too
	//t.Run("squashing of an anonymous struct with json tag", func(t *testing.T) {
	//	knownRegistry.AddKnownTypes(group, &WithAnonAndJsonTag{})
	//	instance := &WithAnonAndJsonTag{SomeVal: 1, ChildType: ChildType{Value: 2}}
	//
	//	marshaled, err := decoder.Marshal(instance)
	//	require.NoError(t, err)
	//	assert.NotEmpty(t, marshaled)
	//
	//	decodedObj, err := decoder.Unmarshal(marshaled)
	//	require.NoError(t, err)
	//	assert.EqualValues(t, instance, decodedObj)
	//})

	//t.Run("squashing of an anonymous struct as pointer", func(t *testing.T) {
	//	knownRegistry.AddKnownTypes(group, &WithAnonPointer{})
	//	instance := &WithAnonPointer{SomeVal: 1, ChildType: &ChildType{Value: 2}}
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
