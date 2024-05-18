package prometheusmetrics

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/pkg/metrics"
	"github.com/sapslaj/zonepop/pkg/utils"
	"github.com/sapslaj/zonepop/provider"
)

type PrometheusMetricsProviderConfig struct {
	MetricNamespace string
	MetricSubsystem string
	SourceLabels    []string
	ProviderLabels  []string
}

type prometheusMetricsProvider struct {
	name                 string
	config               PrometheusMetricsProviderConfig
	logger               *zap.Logger
	forwardLookupFilter  configtypes.EndpointFilterFunc
	endpointMetricDesc   *prometheus.Desc
	endpointMetricLabels []string
	endpointMetrics      []prometheus.Metric
	mutex                sync.Mutex
}

func NewPrometheusMetricsProvider(
	name string,
	providerConfig PrometheusMetricsProviderConfig,
	forwardLookupFilter configtypes.EndpointFilterFunc,
) (provider.Provider, error) {
	if providerConfig.MetricNamespace == "" {
		providerConfig.MetricNamespace = metrics.Namespace
	}
	endpointMetricLabels := []string{
		"hostname",
		"ipv4",
		"ipv6",
		"ttl",
	}
	endpointMetricLabels = append(endpointMetricLabels, providerConfig.SourceLabels...)
	endpointMetricLabels = append(endpointMetricLabels, providerConfig.ProviderLabels...)
	p := &prometheusMetricsProvider{
		name:                name,
		config:              providerConfig,
		logger:              log.MustNewLogger().Named("prometheus_metrics_provider:" + name),
		forwardLookupFilter: forwardLookupFilter,
		endpointMetricDesc: prometheus.NewDesc(
			prometheus.BuildFQName(
				providerConfig.MetricNamespace,
				providerConfig.MetricSubsystem,
				"endpoint",
			),
			"Info metric about an endpoint",
			endpointMetricLabels,
			nil,
		),
		endpointMetricLabels: endpointMetricLabels,
		endpointMetrics:      []prometheus.Metric{},
	}
	err := prometheus.Register(p)
	return p, err
}

func (p *prometheusMetricsProvider) push(labels map[string]string) error {
	labelValues := []string{}
	for _, key := range p.endpointMetricLabels {
		labelValues = append(labelValues, labels[key])
	}
	metric, err := prometheus.NewConstMetric(
		p.endpointMetricDesc,
		prometheus.GaugeValue,
		1.0,
		labelValues...,
	)
	if err != nil {
		return err
	}
	p.endpointMetrics = append(p.endpointMetrics, metric)
	return nil
}

func (p *prometheusMetricsProvider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	forwardEndpoints := utils.Filter(p.forwardLookupFilter, endpoints)
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.endpointMetrics = []prometheus.Metric{}
	for _, endpoint := range forwardEndpoints {
		labels := map[string]string{
			"hostname": endpoint.Hostname,
			"ttl":      strconv.FormatInt(endpoint.RecordTTL, 10),
		}
		for key, value := range endpoint.SourceProperties {
			if slices.Contains(p.config.SourceLabels, key) {
				labels[key] = fmt.Sprint(value)
			}
		}
		for key, value := range endpoint.ProviderProperties {
			if slices.Contains(p.config.ProviderLabels, key) {
				labels[key] = fmt.Sprint(value)
			}
		}
		if len(endpoint.IPv4s) == 0 && len(endpoint.IPv6s) == 0 {
			err := p.push(labels)
			if err != nil {
				return err
			}
			continue
		}
		if len(endpoint.IPv4s) == 0 {
			for _, ipv6 := range endpoint.IPv6s {
				labels["ipv6"] = ipv6
				err := p.push(labels)
				if err != nil {
					return err
				}
			}
			continue
		}
		if len(endpoint.IPv6s) == 0 {
			for _, ipv4 := range endpoint.IPv4s {
				labels["ipv4"] = ipv4
				err := p.push(labels)
				if err != nil {
					return err
				}
			}
			continue
		}
		for _, ipv4 := range endpoint.IPv4s {
			labels["ipv4"] = ipv4
			for _, ipv6 := range endpoint.IPv6s {
				labels["ipv6"] = ipv6
				err := p.push(labels)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (p *prometheusMetricsProvider) Describe(ch chan<- *prometheus.Desc) {
	ch <- p.endpointMetricDesc
}

func (p *prometheusMetricsProvider) Collect(ch chan<- prometheus.Metric) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for _, metric := range p.endpointMetrics {
		ch <- metric
	}
}
