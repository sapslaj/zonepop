package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

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

func configProviders(t *testing.T, config Config) []provider.Provider {
	t.Helper()
	p, err := config.Providers()
	if err != nil {
		t.Fatalf("config.Providers returned error: %v", err)
	}
	return p
}

func configSources(t *testing.T, config Config) []source.Source {
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
			if len(sources) != 1 {
				t.Errorf("len(sources) != 1 (got %d)", len(sources))
			}
			assertType(t, sources[0], "*vyos.vyosSSHSource")
			providers := configProviders(t, config)
			assert.Len(t, providers, 1)
			assertType(t, providers[0], "*aws.route53Provider")
		})
	}
}

func TestLuaConfig_Providers(t *testing.T) {
	luaConfig := map[string]struct {
		providerType   string
		configFileName string
	}{
		"custom": {
			providerType:   "*custom.customLuaProvider",
			configFileName: "test_lua/lua_config_providers_custom.lua",
		},
		"aws_route53": {
			providerType:   "*aws.route53Provider",
			configFileName: "test_lua/lua_config_providers_aws_route53.lua",
		},
	}
	for n, tc := range luaConfig {
		t.Run(n, func(t *testing.T) {
			config := newTestLuaConfig(t, tc.configFileName)
			providers := configProviders(t, config)
			assert.Len(t, providers, 1)
			assertType(t, providers[0], tc.providerType)
		})
	}
}

func TestLuaConfig_Sources(t *testing.T) {
	luaConfig := map[string]struct {
		sourceType     string
		configFileName string
	}{
		"custom": {
			sourceType:     "*custom.customLuaSource",
			configFileName: "test_lua/lua_config_sources_custom.lua",
		},
		"vyos_ssh": {
			sourceType:     "*vyos.vyosSSHSource",
			configFileName: "test_lua/lua_config_sources_vyos_ssh.lua",
		},
	}
	for n, tc := range luaConfig {
		t.Run(n, func(t *testing.T) {
			config := newTestLuaConfig(t, tc.configFileName)
			sources := configSources(t, config)
			assert.Len(t, sources, 1)
			assertType(t, sources[0], tc.sourceType)
		})
	}
}
