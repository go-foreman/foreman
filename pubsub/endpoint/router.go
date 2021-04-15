package endpoint

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
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
		structType := scheme.GetStructType(obj)
		r.routes[structType] = append(r.routes[structType], endpoint)
	}
}

func (r router) Route(obj message.Object) []Endpoint {
	structType := scheme.GetStructType(obj)
	if routes, ok := r.routes[structType]; ok {
		return routes
	}

	return []Endpoint{}
}
