package custom

import (
	"context"
	"fmt"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/source"
)

type customLuaSource struct {
	state         *lua.LState
	endpointsFunc *lua.LFunction
	logger        *zap.Logger
}

func NewCustomLuaSource(state *lua.LState, endpointsFunc *lua.LFunction) (source.Source, error) {
	s := &customLuaSource{
		state:         state,
		endpointsFunc: endpointsFunc,
		logger:        log.MustNewLogger().Named("custom_lua_source"),
	}
	return s, nil
}

func (s *customLuaSource) Endpoints(ctx context.Context) ([]*endpoint.Endpoint, error) {
	co, _ := s.state.NewThread()
	configLt := s.state.NewTable()
	var endpointsLt *lua.LTable
	for {
		st, err, values := s.state.Resume(co, s.endpointsFunc, configLt)

		if st == lua.ResumeError {
			return nil, fmt.Errorf("lua.ResumeError: %w", err)
		}

		for _, lv := range values {
			if r, ok := lv.(*lua.LTable); ok {
				endpointsLt = r
			}
		}

		if st == lua.ResumeOK {
			break
		}
	}
	if endpointsLt == nil {
		return nil, fmt.Errorf("could not get table from endpoints function")
	}
	endpoints := make([]*endpoint.Endpoint, 0)
	for i := 1; i <= endpointsLt.MaxN(); i++ {
		ltEndpoint, ok := endpointsLt.RawGetInt(i).(*lua.LTable)
		if !ok {
			return nil, fmt.Errorf("could not convert element %d to table", i)
		}
		endpoints = append(endpoints, endpoint.FromLuaTable(s.state, ltEndpoint))
	}
	return endpoints, nil
}
