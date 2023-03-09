package custom

import (
	"context"
	"fmt"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/provider"
)

type customLuaProvider struct {
	state               *lua.LState
	updateEndpointsFunc *lua.LFunction
	logger              *zap.Logger
}

func NewCustomLuaProvider(state *lua.LState, updateEndpointsFunc *lua.LFunction) (provider.Provider, error) {
	p := &customLuaProvider{
		state:               state,
		updateEndpointsFunc: updateEndpointsFunc,
		logger:              log.MustNewLogger().Named("custom_lua_provider"),
	}
	return p, nil
}

func (p *customLuaProvider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	co, _ := p.state.NewThread()
	lt := p.state.NewTable()
	for _, e := range endpoints {
		lt.Append(e.ToLuaTable(p.state))
	}
	for {
		st, err, _ := p.state.Resume(co, p.updateEndpointsFunc, lt)

		if st == lua.ResumeError {
			return fmt.Errorf("lua.ResumeError: %w", err)
		}

		if st == lua.ResumeOK {
			break
		}
	}
	return nil
}
