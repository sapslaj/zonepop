package custom

import (
	"context"
	"fmt"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/provider"
)

type customLuaProvider struct {
	forwardLookupFilter configtypes.EndpointFilterFunc
	reverseLookupFilter configtypes.EndpointFilterFunc
	state               *lua.LState
	updateEndpointsFunc *lua.LFunction
	logger              *zap.Logger
}

func NewCustomLuaProvider(
	state *lua.LState,
	updateEndpointsFunc *lua.LFunction,
	forwardLookupFilter configtypes.EndpointFilterFunc,
	reverseLookupFilter configtypes.EndpointFilterFunc,
) (provider.Provider, error) {
	p := &customLuaProvider{
		forwardLookupFilter: forwardLookupFilter,
		reverseLookupFilter: reverseLookupFilter,
		state:               state,
		updateEndpointsFunc: updateEndpointsFunc,
		logger:              log.MustNewLogger().Named("custom_lua_provider"),
	}
	return p, nil
}

func (p *customLuaProvider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	co, _ := p.state.NewThread()
	endpointsLt := p.state.NewTable()
	for _, e := range endpoints {
		endpointsLt.Append(e.ToLuaTable(p.state))
	}
	configLt := p.state.NewTable()
	forwardLookupFilterFunc := p.state.NewFunction(p.createEndpointFilterFunction(p.forwardLookupFilter))
	reverseLookupFilterFunc := p.state.NewFunction(p.createEndpointFilterFunction(p.reverseLookupFilter))
	configLt.RawSetString("forward_lookup_filter", forwardLookupFilterFunc)
	configLt.RawSetString("reverse_lookup_filter", reverseLookupFilterFunc)
	for {
		st, err, _ := p.state.Resume(co, p.updateEndpointsFunc, configLt, endpointsLt)

		if st == lua.ResumeError {
			return fmt.Errorf("lua.ResumeError: %w", err)
		}

		if st == lua.ResumeOK {
			break
		}
	}
	return nil
}

func (p *customLuaProvider) createEndpointFilterFunction(f configtypes.EndpointFilterFunc) func(L *lua.LState) int {
	return func(L *lua.LState) int {
		lt := L.CheckTable(1)
		e := endpoint.FromLuaTable(L, lt)
		result := f(e)
		L.Push(lua.LBool(result))
		if result {
			return 1
		}
		return 0
	}
}
