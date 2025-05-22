package config

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/provider"
	"github.com/sapslaj/zonepop/source"
)

func newTestLuaConfig(t *testing.T, configFileName string) Config {
	t.Helper()
	config, err := NewLuaConfig(configFileName)
	if err != nil {
		t.Fatalf("NewLuaConfig returned error: %v", err)
	}
	err = config.Parse()
	if err != nil {
		t.Fatalf("luaConfig.Parse returned error: %v", err)
	}
	return config
}

func configProviders(t *testing.T, config Config) []provider.NamedProvider {
	t.Helper()
	p, err := config.Providers()
	if err != nil {
		t.Fatalf("config.Providers returned error: %v", err)
	}
	return p
}

func configSources(t *testing.T, config Config) []source.NamedSource {
	t.Helper()
	s, err := config.Sources()
	if err != nil {
		t.Fatalf("config.Sources returned error: %v", err)
	}
	return s
}

func assertType(t *testing.T, v any, want string) {
	t.Helper()
	got := reflect.TypeOf(v).String()
	if got != want {
		t.Errorf("incorrect type; got: %s, want: %s", got, want)
	}
}

func TestLuaConfig_Basic(t *testing.T) {
	configFlavors := map[string]string{
		"basic":               "test_lua/lua_config_basic_basic.lua",
		"with global vars":    "test_lua/lua_config_basic_with_global_vars.lua",
		"with local vars":     "test_lua/lua_config_basic_with_local_vars.lua",
		"with function calls": "test_lua/lua_config_basic_with_function_calls.lua",
	}
	for flavorName, configFileName := range configFlavors {
		t.Run(flavorName, func(t *testing.T) {
			config := newTestLuaConfig(t, configFileName)
			sources := configSources(t, config)
			assert.Len(t, sources, 1)
			assert.Equal(t, sources[0].Name, "vyos")
			assertType(t, sources[0].Source, "*vyos.vyosSSHSource")
			providers := configProviders(t, config)
			assert.Len(t, providers, 1)
			assert.Equal(t, providers[0].Name, "route53")
			assertType(t, providers[0].Provider, "*aws.route53Provider")
		})
	}
}

func TestLuaConfig_Providers(t *testing.T) {
	luaConfig := map[string]struct {
		providerType   string
		providerName   string
		configFileName string
	}{
		"aws_route53": {
			providerType:   "*aws.route53Provider",
			providerName:   "route53",
			configFileName: "test_lua/lua_config_providers_aws_route53.lua",
		},
		"custom": {
			providerType:   "*custom.customLuaProvider",
			providerName:   "custom",
			configFileName: "test_lua/lua_config_providers_custom.lua",
		},
		"file": {
			providerType:   "*file.FileProvider",
			providerName:   "file",
			configFileName: "test_lua/lua_config_providers_file.lua",
		},
		"hosts_file": {
			providerType:   "*hostsfile.hostsFileProvider",
			providerName:   "hostsfile",
			configFileName: "test_lua/lua_config_providers_hosts_file.lua",
		},
		"http": {
			providerType:   "*http.HTTPProvider",
			providerName:   "http",
			configFileName: "test_lua/lua_config_providers_http.lua",
		},
		"prometheus_metrics": {
			providerType:   "*prometheusmetrics.prometheusMetricsProvider",
			providerName:   "prom",
			configFileName: "test_lua/lua_config_providers_prometheus_metrics.lua",
		},
	}
	for n, tc := range luaConfig {
		t.Run(n, func(t *testing.T) {
			config := newTestLuaConfig(t, tc.configFileName)
			providers := configProviders(t, config)
			assert.Len(t, providers, 1)
			assert.Equal(t, providers[0].Name, tc.providerName)
			assertType(t, providers[0].Provider, tc.providerType)
		})
	}
}

func TestLuaConfig_Sources(t *testing.T) {
	luaConfig := map[string]struct {
		sourceType     string
		sourceName     string
		configFileName string
	}{
		"custom": {
			sourceType:     "*custom.customLuaSource",
			sourceName:     "custom",
			configFileName: "test_lua/lua_config_sources_custom.lua",
		},
		"vyos_ssh": {
			sourceType:     "*vyos.vyosSSHSource",
			sourceName:     "vyos",
			configFileName: "test_lua/lua_config_sources_vyos_ssh.lua",
		},
	}
	for n, tc := range luaConfig {
		t.Run(n, func(t *testing.T) {
			config := newTestLuaConfig(t, tc.configFileName)
			sources := configSources(t, config)
			assert.Len(t, sources, 1)
			assert.Equal(t, sources[0].Name, tc.sourceName)
			assertType(t, sources[0].Source, tc.sourceType)
		})
	}
}

func TestLuaConfig_LookupFilter(t *testing.T) {
	luaConfig := map[string]struct {
		configFileName string
		endpoints      []*endpoint.Endpoint
	}{
		"basic forward": {
			configFileName: "test_lua/lua_config_lookup_filter_basic_forward.lua",
			endpoints: []*endpoint.Endpoint{
				{
					Hostname: "host-1",
				},
				{
					Hostname: "host-2",
				},
			},
		},
		"basic reverse": {
			configFileName: "test_lua/lua_config_lookup_filter_basic_reverse.lua",
			endpoints: []*endpoint.Endpoint{
				{
					Hostname: "host-1",
				},
				{
					Hostname: "host-2",
				},
			},
		},
	}
	for n, tc := range luaConfig {
		t.Run(n, func(t *testing.T) {
			config := newTestLuaConfig(t, tc.configFileName)
			providers := configProviders(t, config)
			ctx := context.Background()
			err := providers[0].Provider.UpdateEndpoints(ctx, tc.endpoints)
			if err != nil {
				t.Fatalf("UpdateEndpoints failed: %v", err)
			}
		})
	}
}
