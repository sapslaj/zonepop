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
	"github.com/sapslaj/zonepop/pkg/rdns"
	"github.com/sapslaj/zonepop/pkg/utils"
)

type HTTPProviderConfig struct{}

type CurrentEndpointData struct {
	Forward     []byte
	ReverseIPv4 []byte
	ReverseIPv6 []byte
}

type HTTPProvider struct {
	Config              HTTPProviderConfig
	Logger              *zap.Logger
	ForwardLookupFilter configtypes.EndpointFilterFunc
	ReverseLookupFilter configtypes.EndpointFilterFunc
	Mutex               sync.RWMutex
	CurrentEndpointData CurrentEndpointData
}

func NewHTTPProvider(
	providerConfig HTTPProviderConfig,
	forwardLookupFilter configtypes.EndpointFilterFunc,
	reverseLookupFilter configtypes.EndpointFilterFunc,
) (*HTTPProvider, error) {
	p := &HTTPProvider{
		Config:              providerConfig,
		ForwardLookupFilter: forwardLookupFilter,
		ReverseLookupFilter: reverseLookupFilter,
		Logger:              log.MustNewLogger().Named("http_provider"),
		CurrentEndpointData: CurrentEndpointData{
			Forward:     []byte("[]"),
			ReverseIPv4: []byte("[]"),
			ReverseIPv6: []byte("[]"),
		},
	}
	http.HandleFunc("/endpoints/forward", p.MakeHandleFunc(func(h *HTTPProvider) []byte {
		h.Mutex.RLock()
		defer h.Mutex.RUnlock()
		return h.CurrentEndpointData.Forward
	}))
	http.HandleFunc("/endpoints/reverse-ipv4", p.MakeHandleFunc(func(h *HTTPProvider) []byte {
		h.Mutex.RLock()
		defer h.Mutex.RUnlock()
		return h.CurrentEndpointData.ReverseIPv4
	}))
	http.HandleFunc("/endpoints/reverse-ipv6", p.MakeHandleFunc(func(h *HTTPProvider) []byte {
		h.Mutex.RLock()
		defer h.Mutex.RUnlock()
		return h.CurrentEndpointData.ReverseIPv6
	}))
	return p, nil
}

func (p *HTTPProvider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	forwardEndpoints := utils.Filter(p.ForwardLookupFilter, endpoints)

	forwardData, err := json.Marshal(forwardEndpoints)
	if err != nil {
		return err
	}

	reverseEndpoints := utils.Filter(p.ReverseLookupFilter, endpoints)

	reverseIPv4PTRs, err := rdns.PTRsForEndpoints(reverseEndpoints, rdns.Config{
		Zone: "in-addr.arpa.",
	})
	if err != nil {
		return err
	}
	reverseIPv4Data, err := json.Marshal(reverseIPv4PTRs)
	if err != nil {
		return err
	}

	reverseIPv6PTRs, err := rdns.PTRsForEndpoints(reverseEndpoints, rdns.Config{
		Zone: "ip6.arpa.",
	})
	if err != nil {
		return err
	}
	reverseIPv6Data, err := json.Marshal(reverseIPv6PTRs)
	if err != nil {
		return err
	}

	p.Mutex.Lock()
	p.CurrentEndpointData.Forward = forwardData
	p.CurrentEndpointData.ReverseIPv4 = reverseIPv4Data
	p.CurrentEndpointData.ReverseIPv6 = reverseIPv6Data
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
