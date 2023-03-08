package config

import (
	"fmt"

	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	luar "layeh.com/gopher-luar"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/provider"
	"github.com/sapslaj/zonepop/provider/aws"
	"github.com/sapslaj/zonepop/source"
	"github.com/sapslaj/zonepop/source/vyos"
)

var defaultEndpointFilterFunc configtypes.EndpointFilterFunc = func(_ *endpoint.Endpoint) bool { return true }

type Config interface {
	Parse() error
	Sources() ([]source.Source, error)
	Providers() ([]provider.Provider, error)
}

type luaConfig struct {
	logger               *zap.Logger
	configFileName       string
	state                *lua.LState
	sourceDeclarations   map[string]*lua.LTable
	providerDeclarations map[string]*lua.LTable
}

func NewLuaConfig(configFileName string) (Config, error) {
	c := &luaConfig{
		logger:         log.MustNewLogger().Named("lua_config"),
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
		newErr := fmt.Errorf("config: failed to execute configuration file %s: %w", c.configFileName, err)
		c.logger.Error(newErr.Error())
		return newErr
	}
	lv := c.state.Get(-1)
	t, ok := lv.(*lua.LTable)
	if !ok {
		err = fmt.Errorf("config: config file %q does not return a table", c.configFileName)
		c.logger.Error(err.Error())
		return err
	}
	sourceDeclarations := make(map[string]*lua.LTable)
	providerDeclarations := make(map[string]*lua.LTable)
	t.ForEach(func(key, value lua.LValue) {
		if key.String() == "sources" {
			st, ok := value.(*lua.LTable)
			st.ForEach(func(sourceName, sourceDeclaration lua.LValue) {
				sd, ok := sourceDeclaration.(*lua.LTable)
				if !ok {
					c.logger.Sugar().Panicf("config: could not convert %#v to LTable", sourceDeclaration)
				}
				sourceDeclarations[sourceName.String()] = sd
			})
			if !ok {
				c.logger.Sugar().Panicf("config: could not convert %#v to LTable", value)
			}
		}
		if key.String() == "providers" {
			pt, ok := value.(*lua.LTable)
			pt.ForEach(func(providerName, providerDeclaration lua.LValue) {
				pd, ok := providerDeclaration.(*lua.LTable)
				if !ok {
					c.logger.Sugar().Panicf("config: could not convert %#v to LTable", providerDeclaration)
				}
				providerDeclarations[providerName.String()] = pd
			})
			if !ok {
				c.logger.Sugar().Panicf("config: could not convert %#v to LTable", value)
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
		sourceLogger := c.logger.With(zap.String("source", sourceName)).Sugar()
		sourceLogger.Infof("config: processing source %s", sourceName)

		var source source.Source
		var err error

		kind := sourceDeclaration.RawGetInt(1).String()
		sourceConfigRaw := sourceDeclaration.RawGetString("config")
		sourceConfig, ok := sourceConfigRaw.(*lua.LTable)
		if !ok {
			err = fmt.Errorf("config: config for %s could not convert value %#v to LTable", sourceName, sourceConfigRaw)
			sourceLogger.Error(err)
			return sources, err
		}

		sourceLogger = sourceLogger.With("kind", kind)
		sourceLogger.Infof("config: source %s is kind %s", sourceName, kind)
		switch kind {
		case "vyos_ssh":
			var vyosConfig vyos.VyOSSSHSourceConfig
			err = gluamapper.Map(sourceConfig, &vyosConfig)
			if err != nil {
				sourceLogger.Errorw("error configuring source", "err", err)
				return sources, err
			}
			source, err = vyos.NewVyOSSSHSource(vyosConfig)
		}

		if err != nil {
			sourceLogger.Errorw("error configuring source", "err", err)
			return sources, err
		}
		if source != nil {
			sources = append(sources, source)
			sourceLogger.Info("config: Finished configuration")
		}
	}
	return sources, nil
}

func (c *luaConfig) Providers() ([]provider.Provider, error) {
	providers := make([]provider.Provider, 0)
	for providerName, providerDeclaration := range c.providerDeclarations {
		providerLogger := c.logger.With(zap.String("provider", providerName)).Sugar()
		providerLogger.Infof("config: processing provider %s", providerName)

		var provider provider.Provider
		var err error

		kind := providerDeclaration.RawGetInt(1).String()
		providerConfigRaw := providerDeclaration.RawGetString("config")
		providerConfig, ok := providerConfigRaw.(*lua.LTable)
		if !ok {
			err = fmt.Errorf("config: config for %s could not convert value %#v to LTable", providerName, providerConfigRaw)
			providerLogger.Error(err)
			return providers, err
		}

		providerLogger = providerLogger.With("kind", kind)
		providerLogger.Infof("config: provider %s is kind %s", providerName, kind)
		switch kind {
		case "aws_route53":
			forwardFilterFunc := c.createEndpointFilterFunction(providerConfig, "forward_lookup_filter")
			reverseFilterFunc := c.createEndpointFilterFunction(providerConfig, "reverse_lookup_filter")
			var r53Config aws.Route53ProviderConfig
			err = gluamapper.Map(providerConfig, &r53Config)
			if err != nil {
				providerLogger.Errorw("error configuring provider", "err", err)
				return providers, err
			}

			provider, err = aws.NewRoute53Provider(
				r53Config,
				forwardFilterFunc,
				reverseFilterFunc,
			)
		}

		if err != nil {
			providerLogger.Errorw("error configuring provider", "err", err)
			return providers, err
		}
		if provider != nil {
			providers = append(providers, provider)
			providerLogger.Info("config: Finished configuration")
		}
	}
	return providers, nil
}

func (c *luaConfig) endpointToLTable(e *endpoint.Endpoint) *lua.LTable {
	lt := c.state.NewTable()
	lt.RawSetString("hostname", luar.New(c.state, e.Hostname))
	lt.RawSetString("ipv4s", luar.New(c.state, e.IPv4s))
	lt.RawSetString("ipv6s", luar.New(c.state, e.IPv6s))
	lt.RawSetString("record_ttl", luar.New(c.state, e.RecordTTL))
	lt.RawSetString("source_properties", luar.New(c.state, e.SourceProperties))
	lt.RawSetString("provider_properties", luar.New(c.state, e.ProviderProperties))
	return lt
}

func (c *luaConfig) createEndpointFilterFunction(table *lua.LTable, key string) configtypes.EndpointFilterFunc {
	luaFunc, ok := table.RawGetString(key).(*lua.LFunction)
	if !ok {
		c.logger.Sugar().Infof("no %s endpoint filter function defined", key)
		return defaultEndpointFilterFunc
	}
	return func(e *endpoint.Endpoint) bool {
		co, _ := c.state.NewThread()
		result := true
		for {
			st, err, values := c.state.Resume(co, luaFunc, c.endpointToLTable(e))

			if st == lua.ResumeError {
				c.logger.Sugar().Panicf("endpoint filter lua.ResumeError: %v", err)
				break
			}

			for _, lv := range values {
				if r, ok := lv.(lua.LBool); ok {
					result = bool(r)
				}
			}

			if st == lua.ResumeOK {
				c.logger.Sugar().Debugf("endpoint filter call success")
				break
			}

		}
		return result
	}
}
