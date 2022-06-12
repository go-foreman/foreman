package endpoint

import (
	"reflect"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
)

//go:generate mockgen --build_flags=--mod=mod -destination ../../testing/mocks/pubsub/endpoint/router.go -package endpoint . Router

// Router is a registry of Endpoints and types. Each type can have multiple endpoints assigned.
type Router interface {
	// RegisterEndpoint assigns types of objects to an endpoint
	RegisterEndpoint(endpoint Endpoint, objects ...message.Object)
	// Route returns a list of endpoints that were assigned to a type of object
	Route(obj message.Object) []Endpoint
}

// NewRouter creates new instance of Router with default implementation
func NewRouter() Router {
	return &router{
		routes: make(map[reflect.Type][]Endpoint),
	}
}

type router struct {
	routes map[reflect.Type][]Endpoint
}

func (r *router) RegisterEndpoint(endpoint Endpoint, objects ...message.Object) {
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
