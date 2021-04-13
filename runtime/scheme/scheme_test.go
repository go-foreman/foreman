package scheme

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	group Group = "test"
)

type SomeTestType struct {
	TypeMeta
	A int
}

type SomeAnotherTestType struct {
	TypeMeta
	B int
}

func TestKnownTypesRegistry_AddKnownTypeWithName(t *testing.T) {
	knownRegistry := NewKnownTypesRegistry()
	knownRegistry.AddKnownTypeWithName(GroupKind{
		Group:   group,
		Kind:    "CustomKind",
	}, &SomeTestType{})

	someTestTypeInstance, err := knownRegistry.NewObject(GroupKind{
		Group:   group,
		Kind:    "CustomKind",
	})
	require.NoError(t, err)
	assert.NotNil(t, someTestTypeInstance)
	assert.IsType(t, &SomeTestType{}, someTestTypeInstance)

	expected := "group is required on all types: CustomKind *scheme.SomeTestType"
	require.PanicsWithValue(t, expected, func() {
		knownRegistry.AddKnownTypeWithName(GroupKind{
			Group:   "",
			Kind:    "CustomKind",
		}, &SomeTestType{})
	})

	expected = "Double registration of different types for test.CustomKind: old=github.com/go-foreman/foreman/runtime/scheme.SomeTestType, new=github.com/go-foreman/foreman/runtime/scheme.SomeAnotherTestType"
	require.PanicsWithValue(t, expected, func() {
		knownRegistry.AddKnownTypeWithName(GroupKind{
			Group:   group,
			Kind:    "CustomKind",
		}, &SomeAnotherTestType{})
	})
}

func TestKnownTypesRegistry_AddKnownTypes(t *testing.T) {
	knownRegistry := NewKnownTypesRegistry()
	t.Run("no types passed", func(t *testing.T) {
		//no error, nothing is registered
		knownRegistry.AddKnownTypes(group)
	})

	t.Run("added two types", func(t *testing.T) {
		knownRegistry.AddKnownTypes(group, &SomeTestType{}, &SomeAnotherTestType{})
		someTestType, err := knownRegistry.NewObject(GroupKind{
			Group:   group,
			Kind:    "SomeTestType",
		})
		require.NoError(t, err)
		assert.NotNil(t, someTestType)
		assert.IsType(t, &SomeTestType{}, someTestType)

		someAnotherTestType, err := knownRegistry.NewObject(GroupKind{
			Group:   group,
			Kind:    "SomeAnotherTestType",
		})
		require.NoError(t, err)
		assert.NotNil(t, someAnotherTestType)
		assert.IsType(t, &SomeAnotherTestType{}, someAnotherTestType)
	})

	t.Run("type is not registered", func(t *testing.T) {
		loadedType, err := knownRegistry.NewObject(GroupKind{
			Group:   group,
			Kind:    "XXXSomeTestType",
		})
		assert.Nil(t, loadedType)
		assert.Error(t, err)
		assert.EqualError(t, err, "type test.XXXSomeTestType is not registered in KnownTypes")
	})
}
