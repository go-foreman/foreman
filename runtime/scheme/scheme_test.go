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

type WithEmbeddedStruct struct {
	SomeTestType
	Kek int
}

func TestKnownTypesRegistry_AddKnownTypeWithName(t *testing.T) {
	knownRegistry := NewKnownTypesRegistry()

	t.Run("add known type by pointer", func(t *testing.T) {
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
	})

	t.Run("add known type by value", func(t *testing.T) {
		knownRegistry.AddKnownTypeWithName(GroupKind{
			Group: group,
			Kind: "SomeKind",
		}, &SomeTestType{})

		someKindInstance, err := knownRegistry.NewObject(GroupKind{
			Group:   group,
			Kind:    "CustomKind",
		})
		require.NoError(t, err)
		assert.NotNil(t, someKindInstance)
		assert.IsType(t, &SomeTestType{}, someKindInstance)
	})

	t.Run("group is empty", func(t *testing.T) {
		expected := "group is required on all types: CustomKind scheme.SomeTestType"
		require.PanicsWithValue(t, expected, func() {
			knownRegistry.AddKnownTypeWithName(GroupKind{
				Group:   "",
				Kind:    "CustomKind",
			}, &SomeTestType{})
		})
	})

	t.Run("double registration", func(t *testing.T) {
		expected := "Double registration of different types for test.CustomKind: old=github.com/go-foreman/foreman/runtime/scheme.SomeTestType, new=github.com/go-foreman/foreman/runtime/scheme.SomeAnotherTestType"
		require.PanicsWithValue(t, expected, func() {
			knownRegistry.AddKnownTypeWithName(GroupKind{
				Group:   group,
				Kind:    "CustomKind",
			}, &SomeAnotherTestType{})
		})
	})

	t.Run("object is not struct type", func(t *testing.T) {
		wrongType := notStructType("xxx")
		assert.PanicsWithValue(t, "all types must be pointers to structs", func() {
			knownRegistry.AddKnownTypeWithName(GroupKind{
				Group:   group,
				Kind:    "WrongKind",
			}, &wrongType)
		})
	})

	t.Run("schema returns no kind for not registered object with embedded schema.obj", func(t *testing.T) {
		knownRegistry.AddKnownTypes(group, &SomeTestType{})
		embeddedInstance, err := knownRegistry.ObjectKind(&WithEmbeddedStruct{})
		assert.Error(t, err)
		assert.EqualError(t, err, "no kind is registered in schema for the type WithEmbeddedStruct")
		assert.Nil(t, embeddedInstance)
	})

	t.Run("schema returns kind for registered object with embedded schema.obj", func(t *testing.T) {
		knownRegistry.AddKnownTypes(group, &WithEmbeddedStruct{})
		embeddedInstance, err := knownRegistry.ObjectKind(&WithEmbeddedStruct{})
		require.NoError(t, err)
		require.NotNil(t, embeddedInstance)
		assert.EqualValues(t, &GroupKind{Group: group, Kind: "WithEmbeddedStruct"}, embeddedInstance)
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

type notStructType string

func (n notStructType) GroupKind() GroupKind {
	panic("implement me")
}

func (n notStructType) SetGroupKind(gk *GroupKind) {

}

