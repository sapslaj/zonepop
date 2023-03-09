package endpoint

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	lua "github.com/yuin/gopher-lua"
)

func TestLuaTableConversions(t *testing.T) {
	expected := &Endpoint{
		Hostname:  "test-host",
		IPv4s:     []string{"192.0.2.1"},
		IPv6s:     []string{},
		RecordTTL: 60,
		SourceProperties: map[string]any{
			"dummy": "prop",
		},
		ProviderProperties: map[string]any{},
	}

	state := lua.NewState()
	lt := expected.ToLuaTable(state)

	converted := FromLuaTable(state, lt)

	diff := cmp.Diff(expected, converted)
	if diff != "" {
		t.Fatalf("diff:\n%s", diff)
	}
}
