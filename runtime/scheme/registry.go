package scheme

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// KnownTypesRegistryInstance is a default global types registry
var KnownTypesRegistryInstance = NewKnownTypesRegistry()

type KnownTypesRegistry interface {
	// AddKnownTypes registers list of types of objects to a Group. Kind of each type will be set as struct name using reflection
	AddKnownTypes(gv Group, types ...Object)
	// AddKnownTypeWithName registers a type an object to a Group and custom defined Kind
	AddKnownTypeWithName(gk GroupKind, obj Object)
	// NewObject instantiates new object instance of a type registered behind GroupKind
	NewObject(gk GroupKind) (Object, error)
	// ObjectKind returns GroupKind of an already registered type
	ObjectKind(object Object) (*GroupKind, error)
}

// NewKnownTypesRegistry returns new empty registry of types
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

// AddKnownTypes registers list of types of objects to a Group. Kind of each type will be set as struct name using reflection
func (r *knownTypesRegistry) AddKnownTypes(g Group, types ...Object) {
	for _, obj := range types {
		structType := GetStructType(obj)
		r.addKnownTypeWithName(GroupKind{
			Group: g,
			Kind:  structType.Name(),
		}, obj, structType)
	}
}

// AddKnownTypeWithName registers a type an object to a Group and custom defined Kind
func (r *knownTypesRegistry) AddKnownTypeWithName(gk GroupKind, obj Object) {
	structType := GetStructType(obj)
	r.addKnownTypeWithName(gk, obj, structType)
}

// NewObject instantiates new object instance of a type registered behind GroupKind
func (r *knownTypesRegistry) NewObject(gk GroupKind) (Object, error) {
	t, exists := r.gvkToType[gk]

	if !exists {
		return nil, errors.Errorf("type %s is not registered in KnownTypes", gk.String())
	}

	obj := reflect.New(t).Interface().(Object)
	obj.SetGroupKind(&gk)

	return obj, nil
}

// ObjectKind returns GroupKind of an already registered type
func (r *knownTypesRegistry) ObjectKind(obj Object) (*GroupKind, error) {
	structType := GetStructType(obj)
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

// GetStructType returns reflect.Type of the passed object
func GetStructType(obj Object) reflect.Type {
	structType := reflect.TypeOf(obj)

	if structType.Kind() != reflect.Ptr {
		structType = reflect.PtrTo(structType)
	}

	structType = structType.Elem()
	if structType.Kind() != reflect.Struct {
		panic("all types must be pointers to structs")
	}

	//if hasDeepEmbeddedGK(structType) {
	//	panic("struct has embedded another struct on the first level which implement Object interface. need implement explicitly Object interface(embed TypeMeta struct)")
	//}

	return structType
}

//var objectType = reflect.TypeOf((*Object)(nil)).Elem()

//func hasDeepEmbeddedGK(structType reflect.Type) bool {
//	for i := 0; i < structType.NumField(); i++ {
//		if structType := structType.Field(i).Type; structType.Kind() == reflect.Struct {
//			panic(structType.Name())
//			_, ok := reflect.New(structType).Interface().(Object)
//			if ok {
//				return true
//			}
//		}
//	}
//
//	return false
//}
