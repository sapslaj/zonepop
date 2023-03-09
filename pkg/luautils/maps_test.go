package luautils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func makeTable(state *lua.LState, m map[any]any) *lua.LTable {
	lt := state.NewTable()
	for k, v := range m {
		lt.RawSet(luar.New(state, k), luar.New(state, v))
	}
	return lt
}

func TestLuaTableToMap(t *testing.T) {
	state := lua.NewState()
	tests := map[string]struct {
		input    *lua.LTable
		expected map[string]any
	}{
		"simple": {
			input: makeTable(state, map[any]any{
				"test": "foobar",
			}),
			expected: map[string]any{
				"test": "foobar",
			},
		},
	}

	for desc, tc := range tests {
		got := LuaTableToMap[string, any](tc.input)
		if diff := cmp.Diff(tc.expected, got); diff != "" {
			t.Errorf("%s: diff:\n%s", desc, diff)
		}
	}
}
