package file

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"text/template"

	scp "github.com/bramvdbogaerde/go-scp"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/gluamapper"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/pkg/rdns"
	"github.com/sapslaj/zonepop/pkg/sshconnection"
	"github.com/sapslaj/zonepop/pkg/utils"
	"github.com/sapslaj/zonepop/provider"
)

type FileProviderConfigSSH struct {
	Host     string
	Username string
	Password string
}

type FileProviderConfigFile struct {
	Filename     string
	Permissions  string
	RecordSuffix string
	Template     string
	Zone         string
	Generate     *lua.LFunction
}

type FileProviderConfig struct {
	SSH   FileProviderConfigSSH
	Files []FileProviderConfigFile
}

type FileProvider struct {
	Config              FileProviderConfig
	Logger              *zap.Logger
	ForwardLookupFilter configtypes.EndpointFilterFunc
	ReverseLookupFilter configtypes.EndpointFilterFunc
	State               *lua.LState
}

type TemplateResult struct {
	FileProviderConfigFile
	Result string
}

func NewFileProvider(
	state *lua.LState,
	providerConfig FileProviderConfig,
	forwardLookupFilter configtypes.EndpointFilterFunc,
	reverseLookupFilter configtypes.EndpointFilterFunc,
) (provider.Provider, error) {
	p := &FileProvider{
		State:               state,
		Config:              providerConfig,
		Logger:              log.MustNewLogger().Named("file_provider"),
		ForwardLookupFilter: forwardLookupFilter,
		ReverseLookupFilter: reverseLookupFilter,
	}
	return p, nil
}

func (p *FileProvider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	forwardEndpoints := utils.Filter(p.ForwardLookupFilter, endpoints)
	reverseEndpoints := utils.Filter(p.ReverseLookupFilter, endpoints)

	results := []TemplateResult{}

	for _, fileConfig := range p.Config.Files {
		if fileConfig.Permissions == "" {
			fileConfig.Permissions = "0644"
		}
		logger := p.Logger.With(
			zap.String("filename", fileConfig.Filename),
			zap.String("permissions", fileConfig.Permissions),
		)
		if fileConfig.Filename == "" {
			logger.Error("missing filename for file config")
			return fmt.Errorf("missing filename for file config")
		}

		rdnsZone := ""
		if rdns.IsReverseDNSZone(fileConfig.Zone) {
			rdnsZone = fileConfig.Zone
		}

		ptrs, err := rdns.PTRsForEndpoints(reverseEndpoints, rdns.Config{
			Zone:         rdnsZone,
			RecordSuffix: fileConfig.RecordSuffix,
			Logger:       p.Logger,
		})
		if err != nil {
			logger.Error("error generating rDNS records", zap.Error(err))
			return fmt.Errorf("error generating rDNS records: %w", err)
		}

		var result string

		if fileConfig.Generate != nil {
			logger.Sugar().Infof("len(endpoints) = %d", len(endpoints))
			logger.Sugar().Infof("endpoints[0].Hostname = %s", endpoints[0].Hostname)
			logger.Sugar().Infof("endpoints[0].IPv4s = %v", endpoints[0].IPv4s)
			co, _ := p.State.NewThread()
			for {
				st, err, values := p.State.Resume(
					co,
					fileConfig.Generate,
					gluamapper.FromGoValue(co, forwardEndpoints),
					gluamapper.FromGoValue(co, ptrs),
				)

				if err != nil {
					logger.Error("error while running generate function", zap.Error(err))
					return fmt.Errorf("error while running generate function: %w", err)
				}

				if len(values) == 0 {
					logger.Error("nothing returned from generate function")
					return fmt.Errorf("nothing returned from generate function")
				}

				rawResult := gluamapper.ToGoValue(values[0], gluamapper.Option{})
				var ok bool
				result, ok = rawResult.(string)
				if !ok {
					logger.Sugar().Errorf("wrong return value type (expected string; got %T)", rawResult)
					return fmt.Errorf("wrong return value type (expected string; got %T)", rawResult)
				}

				if st == lua.ResumeOK {
					break
				}
			}
		} else {
			tpl, err := template.New("").Funcs(utils.NewSprout().Build()).Parse(fileConfig.Template)
			if err != nil {
				logger.Error("failed to parse template", zap.Error(err))
				return fmt.Errorf("failed to parse template: %w", err)
			}
			var sb strings.Builder
			err = tpl.Execute(&sb, struct {
				Endpoints  []*endpoint.Endpoint
				PTRRecords []rdns.PTRRecord
			}{
				Endpoints:  forwardEndpoints,
				PTRRecords: ptrs,
			})
			if err != nil {
				logger.Error("failed to render template", zap.Error(err))
				return fmt.Errorf("failed to render template: %w", err)
			}
			result = sb.String()
		}

		results = append(results, TemplateResult{
			FileProviderConfigFile: fileConfig,
			Result:                 result,
		})
	}

	if p.Config.SSH.Host == "" {
		errMsg := "failed to save to local file"
		for _, result := range results {
			logger := p.Logger.With(
				zap.String("filename", result.Filename),
				zap.String("permissions", result.Permissions),
			)
			logger.Sugar().Infof("saving hosts file to (local) %s with permissions %s", result.Filename, result.Permissions)
			perm, err := strconv.ParseInt(result.Permissions, 8, 0)
			if err != nil {
				logger.Error(errMsg, zap.Error(err))
				return fmt.Errorf("%s: %w", errMsg, err)
			}
			logger.Sugar().Infof("perm: %s  %d", result.Permissions, fs.FileMode(perm))
			err = os.WriteFile(result.Filename, []byte(result.Result), fs.FileMode(perm))
			if err != nil {
				logger.Error(errMsg, zap.Error(err))
				return fmt.Errorf("%s: %w", errMsg, err)
			}
		}
	} else {
		errMsg := "failed to save to remote SSH file"
		logger := p.Logger.With(
			zap.String("ssh_host", p.Config.SSH.Host),
			zap.String("ssh_username", p.Config.SSH.Username),
		)
		logger.Sugar().Infof("connecting to SSH host %s@%s", p.Config.SSH.Host, p.Config.SSH.Username)
		conn, err := sshconnection.Connect(p.Config.SSH.Host, p.Config.SSH.Username, p.Config.SSH.Password)
		if err != nil {
			logger.Error(errMsg, zap.Error(err))
			return fmt.Errorf("%s: %w", errMsg, err)
		}
		defer conn.Disconnect()
		for _, result := range results {
			client, err := scp.NewClientBySSH(conn.Client)
			if err != nil {
				logger.Error(errMsg, zap.Error(err))
				return fmt.Errorf("%s: %w", errMsg, err)
			}
			defer client.Close()
			rlogger := logger.With(
				zap.String("filename", result.Filename),
				zap.String("permissions", result.Permissions),
			)
			rlogger.Sugar().Infof(
				"saving hosts file to (SSH) %s:%s with permissions %s",
				p.Config.SSH.Host,
				result.Filename,
				result.Permissions,
			)
			reader := strings.NewReader(result.Result)
			err = client.CopyFile(ctx, reader, result.Filename, result.Permissions)
			if err != nil {
				rlogger.Error(errMsg, zap.Error(err))
				return fmt.Errorf("%s: %w", errMsg, err)
			}
		}
	}

	return nil
}
