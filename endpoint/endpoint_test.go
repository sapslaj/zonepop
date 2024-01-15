package endpoint

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestToLuaTable(t *testing.T) {
	state := lua.NewState()
	defer state.Close()
	err := state.DoFile("test_lua/test_to_lua_table.lua")
	require.NoError(t, err)
	testFunc := state.Get(-1).(*lua.LFunction)

	endpoint := &Endpoint{
		Hostname:  "test-host",
		IPv4s:     []string{"192.0.2.1"},
		IPv6s:     []string{"2001:db8::1"},
		RecordTTL: 60,
		SourceProperties: map[string]any{
			"source_prop": "prop",
		},
		ProviderProperties: map[string]any{
			"provider_prop": "prop",
		},
	}
	ltEndpoint := endpoint.ToLuaTable(state)
	co, _ := state.NewThread()
	st, err, _ := state.Resume(co, testFunc, ltEndpoint)
	require.NoError(t, err)
	require.Equal(t, st, lua.ResumeOK)
}

func TestFromLuaTable(t *testing.T) {
	state := lua.NewState()
	defer state.Close()
	err := state.DoFile("test_lua/test_from_lua_table.lua")
	require.NoError(t, err)
	testFunc := state.Get(-1).(*lua.LFunction)
	co, _ := state.NewThread()
	st, err, values := state.Resume(co, testFunc)
	require.NoError(t, err)
	require.Equal(t, st, lua.ResumeOK)
	ltEndpoint := values[0].(*lua.LTable)
	endpoint := FromLuaTable(state, ltEndpoint)
	assert.Equal(t, "test-host", endpoint.Hostname)
	assert.Equal(t, []string{"192.0.2.1"}, endpoint.IPv4s)
	assert.Equal(t, []string{"2001:db8::1"}, endpoint.IPv6s)
	assert.Equal(t, int64(60), endpoint.RecordTTL)
	assert.Equal(t, map[string]any{"source_prop": "prop"}, endpoint.SourceProperties)
	assert.Equal(t, map[string]any{"provider_prop": "prop"}, endpoint.ProviderProperties)
}

func TestLuaTableDiff(t *testing.T) {
	tests := map[string]struct {
		endpoint *Endpoint
	}{
		"only ipv4 and source props": {
			endpoint: &Endpoint{
				Hostname:  "test-host",
				IPv4s:     []string{"192.0.2.1"},
				IPv6s:     []string{},
				RecordTTL: 60,
				SourceProperties: map[string]any{
					"dummy": "prop",
				},
				ProviderProperties: map[string]any{},
			},
		},
		"all ips and props": {
			endpoint: &Endpoint{
				Hostname:  "test-host",
				IPv4s:     []string{"192.0.2.1"},
				IPv6s:     []string{"2001:db8::1"},
				RecordTTL: 60,
				SourceProperties: map[string]any{
					"source": "prop",
				},
				ProviderProperties: map[string]any{
					"provider": "prop",
				},
			},
		},
	}

	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			state := lua.NewState()
			lt := tc.endpoint.ToLuaTable(state)

			converted := FromLuaTable(state, lt)

			diff := cmp.Diff(tc.endpoint, converted)
			if diff != "" {
				t.Fatalf("diff:\n%s", diff)
			}
		})
	}
}
