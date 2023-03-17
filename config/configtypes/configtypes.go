package configtypes

import "github.com/sapslaj/zonepop/endpoint"

type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "provider context value " + k.name }

// DryRunContextKey is a context key. It is used to tell components to not make
// any changes.
var DryRunContextKey = &contextKey{"dry-run"}

// EndpointFilterFunc is the interface for a function that can filter endpoints
// during provider updates.
type EndpointFilterFunc func(*endpoint.Endpoint) bool

// DefaultEndpointFilterFunc is an EndpointFilterFunc that returns `true` for
// any endpoint given to it; serving as the bare minimum endpoint filter
// function.
var DefaultEndpointFilterFunc EndpointFilterFunc = func(_ *endpoint.Endpoint) bool { return true }
