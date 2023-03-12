package configtypes

import "github.com/sapslaj/zonepop/endpoint"

type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "provider context value " + k.name }

// DryRun is a context key. It is used to tell components to not make any changes.
var DryRunContextKey = &contextKey{"dry-run"}

type EndpointFilterFunc func(*endpoint.Endpoint) bool

var DefaultEndpointFilterFunc EndpointFilterFunc = func(_ *endpoint.Endpoint) bool { return true }
