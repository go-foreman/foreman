package scheme

import (
	"github.com/stretchr/testify/assert"
	"reflect"

	"testing"
)

const someTypeTypePkgPath = "github.com/go-foreman/foreman/pkg/runtime/scheme.sometesttype"

type SomeTestType struct {
	A int
}

type SomeAnotherTestType struct {
	B int
}

func TestRegisterTypes(t *testing.T) {
	testCases := []struct {
		registry        KnownTypesRegistry
		registerType    interface{}
		registerWithKey bool
		key             string
		isType          interface{}
		loadBy          []KeyChoice
	}{
		//registered ptr, loaded by struct with a ptr, as value and by key which is pkg path
		{
			registry:     NewKnownTypesRegistry(),
			registerType: &SomeTestType{},
			isType:       &SomeTestType{},
			loadBy:       []KeyChoice{WithStruct(&SomeTestType{}), WithStruct(SomeTestType{}), WithKey(someTypeTypePkgPath)},
		},
		//registered by value, loaded by struct with a ptr, as value and by key. Loaded type is ptr.
		{
			registry:     NewKnownTypesRegistry(),
			registerType: SomeTestType{},
			isType:       &SomeTestType{},
			loadBy:       []KeyChoice{WithStruct(&SomeTestType{}), WithStruct(SomeTestType{}), WithKey(someTypeTypePkgPath)},
		},
		{
			registry:        NewKnownTypesRegistry(),
			registerType:    &SomeTestType{},
			registerWithKey: true,
			key:             "someKey",
			isType:          &SomeTestType{},
			loadBy:          []KeyChoice{WithKey("someKey")},
		},
		{
			registry:        NewKnownTypesRegistry(),
			registerType:    SomeTestType{},
			registerWithKey: true,
			key:             "someKey",
			isType:          &SomeTestType{},
			loadBy:          []KeyChoice{WithKey("someKey")},
		},
	}

	for _, testCase := range testCases {
		if testCase.registerWithKey {
			testCase.registry.RegisterTypeWithKey(WithKey(testCase.key), testCase.registerType)
		} else {
			testCase.registry.RegisterTypes(testCase.registerType)
		}

		for _, loadBy := range testCase.loadBy {
			loadedType, err := testCase.registry.LoadType(loadBy)
			assert.Nil(t, err)
			assert.NotNil(t, loadedType)
			assert.IsType(t, loadedType, testCase.isType)
		}
	}

	registry := NewKnownTypesRegistry()
	registry.RegisterTypes(&SomeTestType{})
	loadedType, err := registry.LoadType(WithStruct(&SomeAnotherTestType{}))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "is not registered in KnownTypes")
	assert.Nil(t, loadedType)

}

func TestGetType(t *testing.T) {
	registry := NewKnownTypesRegistry()
	registry.RegisterTypes(&SomeTestType{})

	gotType := registry.GetType(WithStruct(&SomeAnotherTestType{}))
	assert.Nil(t, gotType)

	gotType = registry.GetType(WithStruct(&SomeTestType{}))
	assert.NotNil(t, gotType)
	assert.True(t, gotType.String() == reflect.TypeOf(&SomeTestType{}).String())

}

func TestGetStructTypeKey(t *testing.T) {
	assert.Equal(t, GetStructTypeKey(&SomeTestType{}), someTypeTypePkgPath)
	assert.Equal(t, GetStructTypeKey(SomeTestType{}), someTypeTypePkgPath)
}
