package scheme

import (
	"github.com/pkg/errors"
	"reflect"
)

var KnownTypesRegistryInstance = NewKnownTypesRegistry()

type KnownTypesRegistry interface {
	RegisterTypeWithKey(keyChoice KeyChoice, someType interface{})
	RegisterTypes(someType ...interface{})
	GetType(keyChoice KeyChoice) reflect.Type
	LoadType(keyChoice KeyChoice) (interface{}, error)
}

func NewKnownTypesRegistry() KnownTypesRegistry {
	return &reflectSagaRegistry{types: make(map[string]reflect.Type)}
}

type reflectSagaRegistry struct {
	types map[string]reflect.Type
}

func (r *reflectSagaRegistry) RegisterTypeWithKey(keyChoice KeyChoice, someStruct interface{}) {
	structType := reflect.TypeOf(someStruct)

	if structType.Kind() != reflect.Ptr {
		structType = reflect.PtrTo(structType)
	}

	r.types[keyChoice()] = structType
}

func (r *reflectSagaRegistry) RegisterTypes(someStructs ...interface{}) {
	if len(someStructs) > 0 {
		for _, v := range someStructs {
			r.RegisterTypeWithKey(WithStruct(v), v)
		}
	}
}

func (r reflectSagaRegistry) GetType(keyChoice KeyChoice) reflect.Type {
	if sagaType, wasRegistered := r.types[keyChoice()]; wasRegistered {
		return sagaType
	}

	return nil
}

func (r reflectSagaRegistry) LoadType(keyChoice KeyChoice) (interface{}, error) {
	structType := r.GetType(keyChoice)

	if structType == nil {
		return nil, errors.Errorf("Type with key %s is not registered in KnownTypes", keyChoice())
	}

	return reflect.New(structType.Elem()).Interface(), nil
}
