package endpoint

import (
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
)

type Router interface {
	RegisterEndpointWithKey(endpoint Endpoint, msgKeyChoices ...scheme.KeyChoice)
	RegisterEndpoint(endpoint Endpoint, entities ...interface{})
	Route(keyChoice scheme.KeyChoice) []Endpoint
}

func NewRouter() Router {
	return &router{routes: make(map[string][]Endpoint)}
}

type router struct {
	routes map[string][]Endpoint
}

func (r *router) RegisterEndpointWithKey(endpoint Endpoint, msgKeyChoices ...scheme.KeyChoice) {

	for _, msgKeyChoice := range msgKeyChoices {
		key := msgKeyChoice()
		r.routes[key] = append(r.routes[key], endpoint)
	}
}

func (r *router) RegisterEndpoint(endpoint Endpoint, entities ...interface{}) {
	for _, e := range entities {
		r.RegisterEndpointWithKey(endpoint, scheme.WithStruct(e))
	}
}

func (r router) Route(keyChoice scheme.KeyChoice) []Endpoint {
	key := keyChoice()
	if route, ok := r.routes[key]; ok {
		return route
	}

	return []Endpoint{}
}
