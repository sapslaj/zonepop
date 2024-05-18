package config

import (
	"fmt"

	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/config/luazap"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/provider"
	"github.com/sapslaj/zonepop/provider/aws"
	custom_provider "github.com/sapslaj/zonepop/provider/custom"
	hostsfile "github.com/sapslaj/zonepop/provider/hosts_file"
	prometheusmetrics "github.com/sapslaj/zonepop/provider/prometheus_metrics"
	"github.com/sapslaj/zonepop/source"
	custom_source "github.com/sapslaj/zonepop/source/custom"
	"github.com/sapslaj/zonepop/source/vyos"
)

// Config is an interface for configuration providers for source and provider
// configuration and initialization.
type Config interface {
	Parse() error
	Sources() ([]source.NamedSource, error)
	Providers() ([]provider.NamedProvider, error)
}

type luaConfig struct {
	logger               *zap.Logger
	configFileName       string
	state                *lua.LState
	sourceDeclarations   map[string]*lua.LTable
	providerDeclarations map[string]*lua.LTable
}

// NewLuaConfig builds new Lua script configuration provider.
func NewLuaConfig(configFileName string) (Config, error) {
	c := &luaConfig{
		logger:         log.MustNewLogger().Named("lua_config"),
		configFileName: configFileName,
	}
	return c, nil
}

// Parse parses and executes the Lua script passed in and builds source and
// provider declarations.
func (c *luaConfig) Parse() error {
	if c.state != nil && !c.state.IsClosed() {
		c.state.Close()
	}
	c.state = lua.NewState()
	// Disable zap's built-in caller func since luazap provides its own
	logloader := luazap.NewLoader(c.logger.WithOptions(zap.WithCaller(false)))
	c.state.PreloadModule("log", logloader)
	c.state.PreloadModule("zap", logloader)
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

// Sources parses the source declarations into a slice of initialized and
// configured sources.
func (c *luaConfig) Sources() ([]source.NamedSource, error) {
	sources := make([]source.NamedSource, 0)
	for sourceName, sourceDeclaration := range c.sourceDeclarations {
		sourceLogger := c.logger.With(zap.String("source", sourceName)).Sugar()
		sourceLogger.Infof("config: processing source %s", sourceName)

		var sourceInstance source.Source
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
		case "custom":
			endpointFunc, ok := sourceConfig.RawGetString("endpoints").(*lua.LFunction)
			if ok {
				sourceInstance, err = custom_source.NewCustomLuaSource(c.state, endpointFunc)
			}
		case "vyos_ssh":
			var vyosConfig vyos.VyOSSSHSourceConfig
			err = gluamapper.Map(sourceConfig, &vyosConfig)
			if err != nil {
				sourceLogger.Errorw("error configuring source", "err", err)
				return sources, err
			}
			sourceInstance, err = vyos.NewVyOSSSHSource(vyosConfig)
		}

		if err != nil {
			sourceLogger.Errorw("error configuring source", "err", err)
			return sources, err
		}
		if sourceInstance != nil {
			sources = append(sources, source.NamedSource{
				Name:   sourceName,
				Source: sourceInstance,
			})
			sourceLogger.Info("config: Finished configuration")
		}
	}
	return sources, nil
}

// Providers parses the provider declarations into a slice of initialized and
// configured providers.
func (c *luaConfig) Providers() ([]provider.NamedProvider, error) {
	providers := make([]provider.NamedProvider, 0)
	for providerName, providerDeclaration := range c.providerDeclarations {
		providerLogger := c.logger.With(zap.String("provider", providerName)).Sugar()
		providerLogger.Infof("config: processing provider %s", providerName)

		var providerInstance provider.Provider
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
		forwardFilterFunc := c.createEndpointFilterFunction(providerConfig, "forward_lookup_filter")
		reverseFilterFunc := c.createEndpointFilterFunction(providerConfig, "reverse_lookup_filter")
		switch kind {
		case "aws_route53":
			var r53Config aws.Route53ProviderConfig
			err = gluamapper.Map(providerConfig, &r53Config)
			if err != nil {
				providerLogger.Errorw("error configuring provider", "err", err)
				return providers, err
			}

			providerInstance, err = aws.NewRoute53Provider(
				r53Config,
				forwardFilterFunc,
				reverseFilterFunc,
			)
		case "custom":
			updateEndpointsFunc, ok := providerConfig.RawGetString("update_endpoints").(*lua.LFunction)
			if ok {
				providerInstance, err = custom_provider.NewCustomLuaProvider(
					c.state,
					updateEndpointsFunc,
					forwardFilterFunc,
					reverseFilterFunc,
				)
			}
		case "hosts_file":
			var hfConfig hostsfile.HostsFileProviderConfig
			err = gluamapper.Map(providerConfig, &hfConfig)
			if err != nil {
				providerLogger.Errorw("error configuring provider", "err", err)
				return providers, err
			}
			providerInstance, err = hostsfile.NewHostsFileProvider(
				hfConfig,
				forwardFilterFunc,
			)
		case "prometheus_metrics":
			var pmConfig prometheusmetrics.PrometheusMetricsProviderConfig
			err = gluamapper.Map(providerConfig, &pmConfig)
			if err != nil {
				providerLogger.Errorw("error configuring provider", "err", err)
				return providers, err
			}
			providerInstance, err = prometheusmetrics.NewPrometheusMetricsProvider(
				providerName,
				pmConfig,
				forwardFilterFunc,
			)
		}

		if err != nil {
			providerLogger.Errorw("error configuring provider", "err", err)
			return providers, err
		}
		if providerInstance != nil {
			providers = append(providers, provider.NamedProvider{
				Name:     providerName,
				Provider: providerInstance,
			})
			providerLogger.Info("config: Finished configuration")
		}
	}
	return providers, nil
}

func (c *luaConfig) createEndpointFilterFunction(table *lua.LTable, key string) configtypes.EndpointFilterFunc {
	luaFunc, ok := table.RawGetString(key).(*lua.LFunction)
	if !ok {
		c.logger.Sugar().Infof("no %s endpoint filter function defined", key)
		return configtypes.DefaultEndpointFilterFunc
	}
	return func(e *endpoint.Endpoint) bool {
		co, _ := c.state.NewThread()
		result := true
		for {
			st, err, values := c.state.Resume(co, luaFunc, e.ToLuaTable(c.state))

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
