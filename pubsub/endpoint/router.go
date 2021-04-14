package endpoint

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"reflect"
)

type Router interface {
	RegisterEndpoint(endpoint Endpoint, objects... message.Object)
	Route(obj message.Object) []Endpoint
}

func NewRouter() Router {
	return &router{
		routes: make(map[reflect.Type][]Endpoint),
	}
}

type router struct {
	routes map[reflect.Type][]Endpoint
}

func (r *router) RegisterEndpoint(endpoint Endpoint, objects... message.Object) {
	for _, obj := range objects {
		structType := getStructType(obj)
		r.routes[structType] = append(r.routes[structType], endpoint)
	}
}

func (r router) Route(obj message.Object) []Endpoint {
	structType := getStructType(obj)
	if routes, ok := r.routes[structType]; ok {
		return routes
	}

	return []Endpoint{}
}

func getStructType(obj message.Object) reflect.Type {
	structType := reflect.TypeOf(obj)

	if structType.Kind() != reflect.Ptr {
		structType = reflect.PtrTo(structType)
	}

	structType = structType.Elem()
	if structType.Kind() != reflect.Struct {
		panic("All types must be pointers to structs.")
	}

	return structType
}
