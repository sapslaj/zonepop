package http

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/pkg/utils"
)

type HTTPProviderConfig struct{}

type CurrentEndpointData struct {
	Forward []byte
	// TODO: maybe?
	// ReverseIPv4 []byte
	// ReverseIPv6 []byte
}

type HTTPProvider struct {
	Config              HTTPProviderConfig
	Logger              *zap.Logger
	ForwardLookupFilter configtypes.EndpointFilterFunc
	Mutex               sync.RWMutex
	CurrentEndpointData CurrentEndpointData
}

func NewHTTPProvider(
	providerConfig HTTPProviderConfig,
	forwardLookupFilter configtypes.EndpointFilterFunc,
) (*HTTPProvider, error) {
	p := &HTTPProvider{
		Config:              providerConfig,
		ForwardLookupFilter: forwardLookupFilter,
		Logger:              log.MustNewLogger().Named("http_provider"),
		CurrentEndpointData: CurrentEndpointData{
			Forward: []byte("[]"),
		},
	}
	http.HandleFunc("/endpoints/forward", p.MakeHandleFunc(func(h *HTTPProvider) []byte {
		h.Mutex.RLock()
		defer h.Mutex.RUnlock()
		return h.CurrentEndpointData.Forward
	}))
	return p, nil
}

func (p *HTTPProvider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	forwardEndpoints := utils.Filter(p.ForwardLookupFilter, endpoints)

	data, err := json.Marshal(forwardEndpoints)
	if err != nil {
		return err
	}

	p.Mutex.Lock()
	p.CurrentEndpointData.Forward = data
	p.Mutex.Unlock()

	return nil
}

func (p *HTTPProvider) MakeHandleFunc(getter func(*HTTPProvider) []byte) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p.Mutex.RLock()
		defer p.Mutex.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(getter(p))
	}
}
