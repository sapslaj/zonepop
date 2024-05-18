package metrics

import "github.com/prometheus/client_golang/prometheus"

const Namespace = "zonepop"

func NewCounter(opts prometheus.CounterOpts) prometheus.Counter {
	if opts.Namespace == "" {
		opts.Namespace = Namespace
	}
	metric := prometheus.NewCounter(opts)
	prometheus.MustRegister(metric)
	return metric
}

func NewCounterVec(opts prometheus.CounterOpts, labelNames []string) *prometheus.CounterVec {
	if opts.Namespace == "" {
		opts.Namespace = Namespace
	}
	metric := prometheus.NewCounterVec(opts, labelNames)
	prometheus.MustRegister(metric)
	return metric
}

func NewGauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	if opts.Namespace == "" {
		opts.Namespace = Namespace
	}
	metric := prometheus.NewGauge(opts)
	prometheus.MustRegister(metric)
	return metric
}

func NewGaugeVec(opts prometheus.GaugeOpts, labelNames []string) *prometheus.GaugeVec {
	if opts.Namespace == "" {
		opts.Namespace = Namespace
	}
	metric := prometheus.NewGaugeVec(opts, labelNames)
	prometheus.MustRegister(metric)
	return metric
}

func NewHistogram(opts prometheus.HistogramOpts) prometheus.Histogram {
	if opts.Namespace == "" {
		opts.Namespace = Namespace
	}
	metric := prometheus.NewHistogram(opts)
	prometheus.MustRegister(metric)
	return metric
}

func NewHistogramVec(opts prometheus.HistogramOpts, labelNames []string) *prometheus.HistogramVec {
	if opts.Namespace == "" {
		opts.Namespace = Namespace
	}
	metric := prometheus.NewHistogramVec(opts, labelNames)
	prometheus.MustRegister(metric)
	return metric
}
