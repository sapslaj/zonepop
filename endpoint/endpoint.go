package endpoint

import (
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"

	"github.com/sapslaj/zonepop/pkg/gluamapper"
	"github.com/sapslaj/zonepop/pkg/luautils"
)

// Endpoint is a host configuration emitted from a source that contain all of
// the information for a provider to manage DNS records.
type Endpoint struct {
	// The hostname for the endpoint
	Hostname string `json:"hostname"`
	// List of IPv4 addresses for A record creation
	IPv4s []string `json:"ipv4s,omitempty"`
	// List of IPv6 addresses for AAAA record creation
	IPv6s []string `json:"ipv6s,omitempty"`
	// Preferred TTL for resulting records
	RecordTTL int64 `json:"ttl,omitempty"`
	// Additional key, value pairs from the source
	SourceProperties map[string]any `json:"source_properties,omitempty"`
	// Additional key, value pairs for the provider
	ProviderProperties map[string]any `json:"provider_properties,omitempty"`
}

func FromLuaTable(state *lua.LState, lt *lua.LTable) *Endpoint {
	structMapper := gluamapper.NewMapper(gluamapper.Option{
		NameFunc: gluamapper.ToUpperCamelCase,
	})
	var e *Endpoint
	err := structMapper.Map(lt, &e)
	if err != nil {
		return nil
	}
	// don't want CamelCase keys in properties, so have to jump through a few hoops
	e.SourceProperties = luautils.LuaTableToMap[string, any](lt.RawGetString("source_properties"))
	e.ProviderProperties = luautils.LuaTableToMap[string, any](lt.RawGetString("provider_properties"))
	return e
}

func (e *Endpoint) ToLuaTable(state *lua.LState) *lua.LTable {
	lt := state.NewTable()
	lt.RawSetString("record_ttl", lua.LNumber(e.RecordTTL))
	lt.RawSetString("hostname", lua.LString(e.Hostname))
	ipv4s := state.NewTable()
	for _, ipv4 := range e.IPv4s {
		ipv4s.Append(lua.LString(ipv4))
	}
	lt.RawSetString("ipv4s", ipv4s)
	ipv6s := state.NewTable()
	for _, ipv6 := range e.IPv6s {
		ipv6s.Append(lua.LString(ipv6))
	}
	lt.RawSetString("ipv6s", ipv6s)
	sourceProperties := state.NewTable()
	for k, v := range e.SourceProperties {
		sourceProperties.RawSetString(k, luar.New(state, v))
	}
	lt.RawSetString("source_properties", sourceProperties)
	providerProperties := state.NewTable()
	for k, v := range e.ProviderProperties {
		providerProperties.RawSetString(k, luar.New(state, v))
	}
	lt.RawSetString("provider_properties", providerProperties)
	return lt
}
