package config

import (
	"fmt"
	"log"

	"github.com/sapslaj/zonepop/provider"
	"github.com/sapslaj/zonepop/provider/aws"
	"github.com/sapslaj/zonepop/source"
	"github.com/sapslaj/zonepop/source/vyos"
	lua "github.com/yuin/gopher-lua"
)

type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "provider context value " + k.name }

// DryRun is a context key. It is used to tell components to not make any changes.
var DryRunContextKey = &contextKey{"dry-run"}

type Config interface {
	Parse() error
	Sources() ([]source.Source, error)
	Providers() ([]provider.Provider, error)
}

type luaConfig struct {
	configFileName       string
	state                *lua.LState
	sourceDeclarations   map[string]*lua.LTable
	providerDeclarations map[string]*lua.LTable
}

func NewLuaConfig(configFileName string) (Config, error) {
	c := &luaConfig{
		configFileName: configFileName,
	}
	return c, nil
}

func (c *luaConfig) Parse() error {
	if c.state != nil && !c.state.IsClosed() {
		c.state.Close()
	}
	c.state = lua.NewState()
	err := c.state.DoFile(c.configFileName)
	if err != nil {
		return err
	}
	lv := c.state.Get(-1)
	t, ok := lv.(*lua.LTable)
	if !ok {
		return fmt.Errorf("config file %q does not return a table", c.configFileName)
	}
	sourceDeclarations := make(map[string]*lua.LTable)
	providerDeclarations := make(map[string]*lua.LTable)
	t.ForEach(func(key, value lua.LValue) {
		if key.String() == "sources" {
			st, ok := value.(*lua.LTable)
			st.ForEach(func(sourceName, sourceDeclaration lua.LValue) {
				sd, ok := sourceDeclaration.(*lua.LTable)
				if !ok {
					panic(fmt.Errorf("could not convert %#v to LTable", sourceDeclaration))
				}
				sourceDeclarations[sourceName.String()] = sd
			})
			if !ok {
				panic(fmt.Errorf("could not convert %#v to LTable", value))
			}
		}
		if key.String() == "providers" {
			pt, ok := value.(*lua.LTable)
			pt.ForEach(func(providerName, providerDeclaration lua.LValue) {
				pd, ok := providerDeclaration.(*lua.LTable)
				if !ok {
					panic(fmt.Errorf("could not convert %#v to LTable", providerDeclaration))
				}
				providerDeclarations[providerName.String()] = pd
			})
			if !ok {
				panic(fmt.Errorf("could not convert %#v to LTable", value))
			}
		}
	})
	c.sourceDeclarations = sourceDeclarations
	c.providerDeclarations = providerDeclarations
	return nil
}

func (c *luaConfig) Sources() ([]source.Source, error) {
	sources := make([]source.Source, 0)
	for sourceName, sourceDeclaration := range c.sourceDeclarations {
		log.Printf("processing source %s", sourceName)
		var source source.Source
		var err error
		kind := sourceDeclaration.RawGetInt(1).String()
		sourceConfigRaw := sourceDeclaration.RawGetString("config")
		sourceConfig, ok := sourceConfigRaw.(*lua.LTable)
		if !ok {
			return sources, fmt.Errorf("config for %s could not convert value %#v to LTable", sourceName, sourceConfigRaw)
		}
		log.Printf("source %s is kind %s", sourceName, kind)
		switch kind {
		case "vyos_ssh":
			neighbors, ok := sourceConfig.RawGetString("neighbors").(lua.LBool)
			if !ok {
				neighbors = true
			}
			source, err = vyos.NewVyOSSSHSource(
				sourceConfig.RawGetString("host").String(),
				sourceConfig.RawGetString("username").String(),
				sourceConfig.RawGetString("password").String(),
				bool(neighbors),
			)
		}
		if err != nil {
			return sources, err
		}
		if source != nil {
			sources = append(sources, source)
		}
	}
	return sources, nil
}

func (c *luaConfig) Providers() ([]provider.Provider, error) {
	providers := make([]provider.Provider, 0)
	for providerName, providerDeclaration := range c.providerDeclarations {
		log.Printf("processing provider %s", providerName)
		var provider provider.Provider
		var err error
		kind := providerDeclaration.RawGetInt(1).String()
		providerConfigRaw := providerDeclaration.RawGetString("config")
		providerConfig, ok := providerConfigRaw.(*lua.LTable)
		if !ok {
			return providers, fmt.Errorf("config for %s could not convert value %#v to LTable", providerName, providerConfigRaw)
		}
		log.Printf("provider %s is kind %s", providerName, kind)
		switch kind {
		case "aws_route53":
			provider, err = aws.NewRoute53Provder(
				providerConfig.RawGetString("record_suffix").String(),
				providerConfig.RawGetString("forward_zone_id").String(),
				providerConfig.RawGetString("ipv4_reverse_zone_id").String(),
				providerConfig.RawGetString("ipv6_reverse_zone_id").String(),
			)
		}
		if err != nil {
			return providers, err
		}
		if provider != nil {
			providers = append(providers, provider)
		}
	}
	return providers, nil
}
