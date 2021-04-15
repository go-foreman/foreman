package scheme

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

var KnownTypesRegistryInstance = NewKnownTypesRegistry()

type KnownTypesRegistry interface {
	AddKnownTypes(gv Group, types ...Object)
	AddKnownTypeWithName(gk GroupKind, obj Object)
	NewObject(gv GroupKind) (Object, error)
	ObjectKind(object Object) (*GroupKind, error)
}

func NewKnownTypesRegistry() KnownTypesRegistry {
	return &knownTypesRegistry{gvkToType: map[GroupKind]reflect.Type{}, typeToGVK: map[reflect.Type]GroupKind{}}
}

type knownTypesRegistry struct {
	// versionMap allows one to figure out the go type of an object with
	// the given version and name.
	gvkToType map[GroupKind]reflect.Type
	// typeToGroupVersion allows one to find metadata for a given go object.
	// The reflect.Type we index by should *not* be a pointer.
	typeToGVK map[reflect.Type]GroupKind
}

func (r *knownTypesRegistry) AddKnownTypes(g Group, types ...Object) {
	for _, obj := range types {
		structType := getStructType(obj)
		r.addKnownTypeWithName(GroupKind{
			Group:   g,
			Kind:    structType.Name(),
		}, obj, structType)
	}
}

func (r *knownTypesRegistry) AddKnownTypeWithName(gk GroupKind, obj Object) {
	structType := getStructType(obj)
	r.addKnownTypeWithName(gk, obj, structType)
}

func (r *knownTypesRegistry) NewObject(gk GroupKind) (Object, error) {
	t, exists := r.gvkToType[gk]

	if !exists {
		return nil, errors.Errorf("type %s is not registered in KnownTypes", gk.String())
	}

	return reflect.New(t).Interface().(Object), nil
}

func (r *knownTypesRegistry) ObjectKind(obj Object) (*GroupKind, error) {
	structType := getStructType(obj)
	gk, ok := r.typeToGVK[structType]
	if !ok {
		return nil, errors.Errorf("no kind is registered in schema for the type %s", structType.Name())
	}

	if gk.Empty() {
		return nil, errors.Errorf("empty GK returned")
	}

	return &gk, nil
}

func (r *knownTypesRegistry) addKnownTypeWithName(gk GroupKind, obj Object, structType reflect.Type) {
	if len(gk.Group) == 0 {
		panic(fmt.Sprintf("group is required on all types: %s %v", gk, structType))
	}

	if oldT, found := r.gvkToType[gk]; found && oldT != structType {
		panic(fmt.Sprintf("Double registration of different types for %v: old=%v.%v, new=%v.%v", gk, oldT.PkgPath(), oldT.Name(), structType.PkgPath(), structType.Name()))
	}

	r.gvkToType[gk] = structType
	r.typeToGVK[structType] = gk
	obj.SetGroupKind(&gk)
}

func getStructType(obj Object) reflect.Type {
	structType := reflect.TypeOf(obj)

	if structType.Kind() != reflect.Ptr {
		structType = reflect.PtrTo(structType)
	}

	structType = structType.Elem()
	if structType.Kind() != reflect.Struct {
		panic("all types must be pointers to structs")
	}

	if hasDeepEmbeddedGK(structType) {
		panic("struct has embedded another struct on the first level which implement Object interface. need implement explicitly Object interface(embed TypeMeta struct)")
	}

	return structType
}

var objectType = reflect.TypeOf((*Object)(nil)).Elem()

func hasDeepEmbeddedGK(structType reflect.Type) bool {
	for i := 0; i < structType.NumField(); i++ {
		if structType := structType.Field(i).Type; structType.Kind() == reflect.Struct {
			if structType.Implements(objectType) {
				return true
			}
		}
	}

	return false
}
