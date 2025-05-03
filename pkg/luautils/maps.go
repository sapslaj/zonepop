package luautils

import (
	lua "github.com/yuin/gopher-lua"

	"github.com/sapslaj/zonepop/pkg/gluamapper"
)

// LuaTableToMap converts a Lua Table to a Go map without making unwanted
// transformations to keys.
func LuaTableToMap[K ~string, V any](lv lua.LValue) map[K]V {
	raw, ok := gluamapper.ToGoValue(lv, gluamapper.Option{}).(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[K]V)
	for k, v := range raw {
		key := K(k)
		value, ok := v.(V)
		if !ok {
			continue
		}
		result[key] = value
	}
	return result
}
