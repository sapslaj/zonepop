package config

import (
	"os"
	"reflect"
	"testing"

	"github.com/sapslaj/zonepop/provider"
	"github.com/sapslaj/zonepop/source"
)

func testConfigFile(t *testing.T, contents string) (configFileName string) {
	file, err := os.CreateTemp("", "zonepop.*.config.lua")
	if err != nil {
		t.Fatalf("testConfigFile encountered error when creating temp file: %v", err)
	}
	file.Write([]byte(contents))
	err = file.Close()
	if err != nil {
		t.Fatalf("testConfigFile encountered error when closing temp file: %v", err)
	}
	return file.Name()
}

func withTestConfigFile(t *testing.T, contents string, test func(t *testing.T, configFileName string)) {
	f := testConfigFile(t, contents)
	test(t, f)
	if err := os.Remove(f); err != nil {
		t.Fatalf("withTestConfigFile encountered error when removing temp file: %v", err)
	}
}

func newTestLuaConfig(t *testing.T, configFileName string) Config {
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
	p, err := config.Providers()
	if err != nil {
		t.Fatalf("config.Providers returned error: %v", err)
	}
	return p
}

func configSources(t *testing.T, config Config) []source.Source {
	s, err := config.Sources()
	if err != nil {
		t.Fatalf("config.Sources returned error: %v", err)
	}
	return s
}

func assertType(t *testing.T, v any, want string) {
	got := reflect.TypeOf(v).String()
	if got != want {
		t.Errorf("incorrect type; got: %s, want: %s", got, want)
	}
}

func TestLuaConfig_Basic(t *testing.T) {
	configFlavors := map[string]string{
		"basic": `
			return {
				sources = {
					vyos = {
						"vyos_ssh",
						config = {
							host = os.getenv("VYOS_HOST"),
							username = os.getenv("VYOS_USERNAME"),
							password = os.getenv("VYOS_PASSWORD"),
						},
					},
				},
				providers = {
					route53 = {
						"aws_route53",
						config = {
							record_suffix = ".example.com",
							forward_zone_id = "Z2FDTNDATAQYW2",
						},
					},
				},
			}
		`,
		"with global vars": `
			vyos_host = "router.example.com"
			return {
				sources = {
					vyos = {
						"vyos_ssh",
						config = {
							host = vyos_host,
						},
					},
				},
				providers = {
					route53 = {
						"aws_route53",
						config = {
							record_suffix = ".example.com",
							forward_zone_id = "Z2FDTNDATAQYW2",
						},
					},
				},
			}
		`,
		"with local vars": `
			local vyos_host = "router.example.com"
			return {
				sources = {
					vyos = {
						"vyos_ssh",
						config = {
							host = vyos_host,
						},
					},
				},
				providers = {
					route53 = {
						"aws_route53",
						config = {
							record_suffix = ".example.com",
							forward_zone_id = "Z2FDTNDATAQYW2",
						},
					},
				},
			}
		`,
		"with function calls": `
			function get_vyos_host ()
				return os.getenv("VYOS_HOST")
			end
			return {
				sources = {
					vyos = {
						"vyos_ssh",
						config = {
							host = get_vyos_host(),
						},
					},
				},
				providers = {
					route53 = {
						"aws_route53",
						config = {
							record_suffix = ".example.com",
							forward_zone_id = "Z2FDTNDATAQYW2",
						},
					},
				},
			}
		`,
	}
	for flavorName, configContents := range configFlavors {
		t.Run(flavorName, func(t *testing.T) {
			withTestConfigFile(t, configContents, func(t *testing.T, configFileName string) {
				config := newTestLuaConfig(t, configFileName)
				sources := configSources(t, config)
				if len(sources) != 1 {
					t.Errorf("len(sources) != 1 (got %d)", len(sources))
				}
				assertType(t, sources[0], "*vyos.vyosSSHSource")
				providers := configProviders(t, config)
				if len(providers) != 1 {
					t.Errorf("len(providers) != 1 (got %d)", len(providers))
				}
				assertType(t, providers[0], "*aws.route53Provider")
			})
		})
	}
}
