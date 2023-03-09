package custom

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	lua "github.com/yuin/gopher-lua"

	"github.com/sapslaj/zonepop/endpoint"
)

func TestEndpoints(t *testing.T) {
	state := lua.NewState()
	defer state.Close()
	err := state.DoString(`
		return function()
			return {
				{
					hostname = "test-host",
					ipv4s = {"192.0.2.1"},
					ipv6s = {},
					record_ttl = 60,
					source_properties = nil,
					provider_properties = nil,
				},
			}
		end
	`)
	if err != nil {
		t.Fatalf("failed to execute Lua: %v", err)
	}
	endpointsFunc := state.Get(-1).(*lua.LFunction)

	s, err := NewCustomLuaSource(state, endpointsFunc)
	if err != nil {
		t.Fatalf("error creating new custom Lua source: %v", err)
	}

	ctx := context.Background()

	endpoints, err := s.Endpoints(ctx)
	if err != nil {
		t.Fatalf("error getting endpoints: %v", err)
	}

	expected := []*endpoint.Endpoint{
		{
			Hostname:           "test-host",
			IPv4s:              []string{"192.0.2.1"},
			IPv6s:              []string{},
			RecordTTL:          60,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}

	diff := cmp.Diff(endpoints, expected)
	if diff != "" {
		t.Fatalf("mismatch:\n%s", diff)
	}
}
