package prometheusmetrics

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
)

func TestPrometheusMetricsProvider(t *testing.T) {
	t.Parallel()

	p, err := NewPrometheusMetricsProvider(
		"prometheus",
		PrometheusMetricsProviderConfig{
			SourceLabels:   []string{"source_prop"},
			ProviderLabels: []string{"provider_prop"},
		},
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoError(t, err)

	err = p.UpdateEndpoints(context.Background(), []*endpoint.Endpoint{
		{
			Hostname:  "test-host1",
			IPv4s:     []string{"192.0.2.1"},
			IPv6s:     []string{"2001:db8::1"},
			RecordTTL: 60,
			SourceProperties: map[string]any{
				"source_prop":        "foo",
				"source_prop_absent": true,
			},
			ProviderProperties: map[string]any{
				"provider_prop":        "bar",
				"provider_prop_absent": true,
			},
		},
		{
			Hostname:  "test-host2",
			IPv4s:     []string{"192.0.2.2", "192.0.2.202"},
			IPv6s:     []string{},
			RecordTTL: 60,
			SourceProperties: map[string]any{
				"source_prop": "bar",
			},
			ProviderProperties: map[string]any{
				"provider_prop": "foo",
			},
		},
		{
			Hostname:           "test-host3",
			IPv4s:              []string{},
			IPv6s:              []string{"2001:db8::3", "2001:db8::2:3"},
			RecordTTL:          60,
			SourceProperties:   map[string]any{},
			ProviderProperties: map[string]any{},
		},
	})
	require.NoError(t, err)

	err = testutil.CollectAndCompare(
		p,
		strings.NewReader(`# HELP zonepop_endpoint Info metric about an endpoint
# TYPE zonepop_endpoint gauge
zonepop_endpoint{hostname="test-host1",ipv4="192.0.2.1",ipv6="2001:db8::1",provider_prop="bar",source_prop="foo",ttl="60"} 1
zonepop_endpoint{hostname="test-host2",ipv4="192.0.2.2",ipv6="",provider_prop="foo",source_prop="bar",ttl="60"} 1
zonepop_endpoint{hostname="test-host2",ipv4="192.0.2.202",ipv6="",provider_prop="foo",source_prop="bar",ttl="60"} 1
zonepop_endpoint{hostname="test-host3",ipv4="",ipv6="2001:db8::3",provider_prop="",source_prop="",ttl="60"} 1
zonepop_endpoint{hostname="test-host3",ipv4="",ipv6="2001:db8::2:3",provider_prop="",source_prop="",ttl="60"} 1
`,
		),
		"zonepop_endpoint",
	)
	require.NoError(t, err)
}
