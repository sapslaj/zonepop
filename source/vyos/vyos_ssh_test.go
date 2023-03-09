package vyos

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/source"
	"go.uber.org/zap"
)

type mockVyOSSSHConnection struct {
}

func mockVyOSSSHConnectionConnect(host, username, password string) (*mockVyOSSSHConnection, error) {
	return &mockVyOSSSHConnection{}, nil
}

func (c *mockVyOSSSHConnection) Disconnect() error {
	return nil
}

func (c *mockVyOSSSHConnection) Output(cmd string) ([]byte, error) {
	var output string
	switch cmd {
	case "ip -f inet6 neigh show":
		output = `fe80::2f57:5e5a:b3a4:b0c0 dev eth0 lladdr 00:53:3e:03:9a:3b STALE
2001:db8:7357:4:5054:ff:fe6a:99e7 dev eth0 lladdr 00:53:16:b7:7e:4b REACHABLE
fe80::5054:ff:fe6a:99e7 dev eth0 lladdr 00:53:16:b7:7e:4b REACHABLE
`
	case "/usr/libexec/vyos/op_mode/show_dhcp.py --leases --json":
		output = `[
    {
        "start": "2023/03/08 21:57:58",
        "end": "2023/03/09 21:57:58",
        "remaining": "19:16:19",
        "tstp": "",
        "tsfp": "",
        "atsfp": "",
        "cltt": "3 2023/03/08 21:57:58",
        "hardware_address": "00:53:97:50:a0:52",
        "hostname": "host-1",
        "state": "active",
        "ip": "192.0.2.1",
        "pool": "LAN"
    },
    {
        "start": "2023/03/08 15:01:32",
        "end": "2023/03/09 15:01:32",
        "remaining": "12:19:53",
        "tstp": "",
        "tsfp": "",
        "atsfp": "",
        "cltt": "3 2023/03/08 15:01:32",
        "hardware_address": "00:53:3e:03:9a:3b",
        "hostname": "host-2",
        "state": "active",
        "ip": "192.0.2.2",
        "pool": "LAN"
    },
    {
        "start": "2023/03/08 14:57:47",
        "end": "2023/03/09 14:57:47",
        "remaining": "12:16:08",
        "tstp": "",
        "tsfp": "",
        "atsfp": "",
        "cltt": "3 2023/03/08 14:57:47",
        "hardware_address": "00:53:16:b7:7e:4b",
        "hostname": "host-3",
        "state": "active",
        "ip": "192.0.2.3",
        "pool": "LAN"
    }
]
`
	}

	return []byte(output), nil
}

func newMockVyOSSource(sourceConfig VyOSSSHSourceConfig) (source.Source, error) {
	connect := func(host, username, password string) (ConnectionClient, error) {
		return mockVyOSSSHConnectionConnect(host, username, password)
	}
	return &vyosSSHSource{
		config:                 sourceConfig,
		connectionClentConnect: connect,
		logger:                 zap.NewNop(),
	}, nil
}

func TestEndpoints(t *testing.T) {
	for n, withIPv6Neighbors := range map[string]bool{"onlyLeases": false, "withIPV6Neighbors": true} {
		t.Run(n, func(t *testing.T) {
			config := VyOSSSHSourceConfig{
				RecordTTL: 60,
			}
			if withIPv6Neighbors {
				config.CollectIPv6Neighbors = true
			}
			s, err := newMockVyOSSource(config)
			if err != nil {
				t.Fatalf("something went wrong creating mock source: %v", err)
			}
			endpoints, err := s.Endpoints(context.Background())
			if err != nil {
				t.Fatalf("error retrieving endpoints: %v", err)
			}
			endpointsByHostname := make(map[string]*endpoint.Endpoint)
			for _, e := range endpoints {
				_, in := endpointsByHostname[e.Hostname]
				if in {
					t.Errorf("hostname %q is listed multiple times in endpoints", e.Hostname)
				}
				endpointsByHostname[e.Hostname] = e
			}
			expected := map[string]*endpoint.Endpoint{
				"host-1": {
					Hostname:  "host-1",
					IPv4s:     []string{"192.0.2.1"},
					IPv6s:     []string{},
					RecordTTL: 60,
					SourceProperties: map[string]any{
						"dhcp_pool":        "LAN",
						"hardware_address": "00:53:97:50:a0:52",
					},
					ProviderProperties: nil,
				},
				"host-2": {
					Hostname:  "host-2",
					IPv4s:     []string{"192.0.2.2"},
					IPv6s:     []string{},
					RecordTTL: 60,
					SourceProperties: map[string]any{
						"dhcp_pool":        "LAN",
						"hardware_address": "00:53:3e:03:9a:3b",
					},
					ProviderProperties: nil,
				},
				"host-3": {
					Hostname:  "host-3",
					IPv4s:     []string{"192.0.2.3"},
					IPv6s:     []string{"2001:db8:7357:4:5054:ff:fe6a:99e7"},
					RecordTTL: 60,
					SourceProperties: map[string]any{
						"dhcp_pool":        "LAN",
						"hardware_address": "00:53:16:b7:7e:4b",
					},
					ProviderProperties: nil,
				},
			}

			if !withIPv6Neighbors {
				for k := range expected {
					expected[k].IPv6s = nil
				}
			}

			diff := cmp.Diff(endpointsByHostname, expected)
			if diff != "" {
				t.Fatalf("mismatch:\n%s", diff)
			}
		})
	}
}
