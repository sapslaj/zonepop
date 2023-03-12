package endpoint

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	lua "github.com/yuin/gopher-lua"
)

func TestLuaTableConversions(t *testing.T) {
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
