package custom

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/sapslaj/zonepop/config/luazap"
	"github.com/sapslaj/zonepop/endpoint"
)

func TestUpdateEndpoints(t *testing.T) {
	state := lua.NewState()
	defer state.Close()
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	state.PreloadModule("zap", luazap.NewLoader(logger, luazap.WithCaller(false)))
	err := state.DoString(`
		local zap = require("zap")
		return function(endpoints)
			for _, endpoint in ipairs(endpoints) do
				zap.info("new endpoint", endpoint)
			end
		end
	`)
	if err != nil {
		t.Fatalf("failed to execute Lua: %v", err)
	}
	updateEndpointsFunc := state.Get(-1).(*lua.LFunction)

	p, err := NewCustomLuaProvider(state, updateEndpointsFunc)
	if err != nil {
		t.Fatalf("error creating new custom Lua source: %v", err)
	}

	ctx := context.Background()
	endpoints := []*endpoint.Endpoint{
		{
			Hostname:           "test-host",
			IPv4s:              []string{"192.0.2.1"},
			IPv6s:              []string{},
			RecordTTL:          60,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}

	err = p.UpdateEndpoints(ctx, endpoints)
	if err != nil {
		t.Fatalf("error updating endpoints: %v", err)
	}

	if logs.Len() != 1 {
		t.Fatalf("logs.Len() != 1 (got %d)", logs.Len())
	}

	logEntry := logs.All()[0]
	if logEntry.Message != "new endpoint" {
		t.Fatalf("log entry message did not match expected (got %q, want %q)", logEntry.Message, "new endpoint")
	}

	expected := map[string]any{
		"hostname":            "test-host",
		"ipv4s":               []any{string("192.0.2.1")},
		"ipv6s":               map[any]any{}, // If a Table has no MaxN then it is converted to a map
		"record_ttl":          float64(60),
		"provider_properties": map[any]any{},
		"source_properties":   map[any]any{},
	}
	diff := cmp.Diff(logEntry.ContextMap(), expected)
	if diff != "" {
		t.Fatalf("mismatch:\n%s", diff)
	}
}
