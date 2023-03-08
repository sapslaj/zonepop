package vyos

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/pkg/ssh_connection"
	"github.com/sapslaj/zonepop/source"
)

type VyOSSSHSourceConfig struct {
	Host                 string
	Username             string
	Password             string
	CollectIPv6Neighbors bool
	RecordTTL            int64
}

type vyosSSHSource struct {
	config VyOSSSHSourceConfig
	logger *zap.Logger
}

func NewVyOSSSHSource(sourceConfig VyOSSSHSourceConfig) (source.Source, error) {
	return &vyosSSHSource{
		config: sourceConfig,
		logger: log.MustNewLogger().Named("vyos_ssh_source").With(
			zap.String("host", sourceConfig.Host),
			zap.String("username", sourceConfig.Username),
		),
	}, nil
}

func (s *vyosSSHSource) Endpoints(ctx context.Context) ([]*endpoint.Endpoint, error) {
	connection, err := ssh_connection.Connect(s.config.Host, s.config.Username, s.config.Password)

	if err != nil {
		newErr := fmt.Errorf("could not connect to host %s: %w", s.config.Host, err)
		s.logger.Error(newErr.Error())
		return nil, newErr
	}
	defer connection.Disconnect()

	leases, err := s.getLeases(connection, s.config.CollectIPv6Neighbors)
	if err != nil {
		newErr := fmt.Errorf("could not get leases: %w", err)
		s.logger.Error(newErr.Error())
		return nil, newErr
	}

	endpoints := s.leasesToEndpoints(leases)

	return endpoints, nil
}

func (s *vyosSSHSource) leasesToEndpoints(leases []*Lease) []*endpoint.Endpoint {
	endpoints := make([]*endpoint.Endpoint, len(leases))
	for i, lease := range leases {
		endpoints[i] = s.leaseToEndpoint(lease)
	}
	return endpoints
}

func (s *vyosSSHSource) leaseToEndpoint(lease *Lease) *endpoint.Endpoint {
	return &endpoint.Endpoint{
		Hostname:  lease.Hostname,
		IPv4s:     []string{lease.IP},
		IPv6s:     lease.IPv6s,
		RecordTTL: 300,
		SourceProperties: map[string]any{
			"dhcp_pool":        lease.Pool,
			"hardware_address": lease.HardwareAddress,
		},
	}
}

func (s *vyosSSHSource) getNeighbors(connection *ssh_connection.SSHConnection) ([]*Neighbor, error) {
	s.logger.Info("Getting IPv6 neighbors")
	out, err := connection.Output("ip -f inet6 neigh show")
	if err != nil {
		newErr := fmt.Errorf("error getting neighbor output: %w", err)
		s.logger.Error(newErr.Error())
		return nil, newErr
	}
	return ParseNeighborLines(string(out))
}

func (s *vyosSSHSource) getLeases(connection *ssh_connection.SSHConnection, neighbors bool) ([]*Lease, error) {
	s.logger.Info("Getting leases")
	out, err := connection.Output("/usr/libexec/vyos/op_mode/show_dhcp.py --leases --json")
	if err != nil {
		newErr := fmt.Errorf("error getting lease output: %w", err)
		s.logger.Error(newErr.Error())
		return nil, newErr
	}
	leases, err := LeasesFromJSON(out)
	if err != nil {
		newErr := fmt.Errorf("error parsing lease output: %w", err)
		s.logger.Error(newErr.Error())
		return nil, newErr
	}

	if neighbors {
		s.logger.Info("Associating IPv6 neighbors")
		neighbors, err := s.getNeighbors(connection)
		if err != nil {
			newErr := fmt.Errorf("error getting neighbors: %w", err)
			s.logger.Error(newErr.Error())
			return leases, newErr
		}
		for _, lease := range leases {
			lease.AssociatePotentialIPv6s(neighbors)
		}
	}

	return leases, nil
}
