package controller

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sapslaj/zonepop/pkg/metrics"
)

const MetricSubsystem = "controller"

var (
	// No Subsystem.
	MetricSourceUp = metrics.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "source_up",
		},
		[]string{"source"},
	)
	MetricProviderUp = metrics.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "provider_up",
		},
		[]string{"provider"},
	)
	MetricEndpoints = metrics.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "endpoints",
		},
		[]string{"source"},
	)
	// Controller Subsystem.
	MetricRuns = metrics.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: MetricSubsystem,
			Name:      "runs",
		},
		[]string{"status"},
	)
	MetricLastRunTimestamp = metrics.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: MetricSubsystem,
			Name:      "last_run_timestamp",
		},
	)
	MetricLastRunDurationSeconds = metrics.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: MetricSubsystem,
			Name:      "last_run_duration_seconds",
		},
	)
	MetricRunDurationSeconds = metrics.NewHistogram(
		prometheus.HistogramOpts{
			Subsystem: MetricSubsystem,
			Name:      "run_duration_seconds",
		},
	)
)
