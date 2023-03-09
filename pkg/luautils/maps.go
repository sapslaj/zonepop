package luautils

import (
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

// LuaTableToMap converts a Lua Table to a Go map without making unwanted
// transformations to keys.
func LuaTableToMap[K comparable, V any](lv lua.LValue) map[K]V {
	raw, ok := gluamapper.ToGoValue(lv, gluamapper.Option{
		NameFunc: gluamapper.Id,
	}).(map[any]any)
	if !ok {
		return nil
	}
	result := make(map[K]V)
	for k, v := range raw {
		key, ok := k.(K)
		if !ok {
			continue
		}
		value, ok := v.(V)
		if !ok {
			continue
		}
		result[key] = value
	}
	return result
}
