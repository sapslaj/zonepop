package source

import (
	"context"

	"github.com/sapslaj/zonepop/endpoint"
)

// Source defines the interface Endpoint sources should implement.
type Source interface {
	Endpoints(ctx context.Context) ([]*endpoint.Endpoint, error)
}
