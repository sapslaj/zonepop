package provider

import (
	"context"

	"github.com/sapslaj/zonepop/endpoint"
)

// Providers defines the interface DNS providers should implement.
type Provider interface {
	UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error
}

// NamedProvider is a struct that pairs a Provider instance with a logical
// name.
type NamedProvider struct {
	Name     string
	Provider Provider
}
