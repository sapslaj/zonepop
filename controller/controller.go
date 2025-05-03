package controller

import (
	"context"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/provider"
	"github.com/sapslaj/zonepop/source"
)

type Controller struct {
	Sources   []source.NamedSource
	Providers []provider.NamedProvider
	// The interval between individual synchronizations
	Interval time.Duration
	// Logger instance
	Logger *zap.Logger
	// The nextRunAt used for throttling and batching reconciliation
	nextRunAt time.Time
	// The nextRunAtMux is for atomic updating of nextRunAt
	nextRunAtMux sync.Mutex
}

// RunOnce runs a single iteration of a reconciliation loop.
func (c *Controller) RunOnce(ctx context.Context) error {
	var errors error
	start := time.Now()
	// Hack to force the labels to always show up
	MetricRuns.WithLabelValues("success").Add(0)
	MetricRuns.WithLabelValues("errored").Add(0)
	defer func() {
		status := "success"
		if errors != nil {
			status = "errored"
		}
		MetricRuns.WithLabelValues(status).Inc()
		MetricLastRunTimestamp.SetToCurrentTime()
		duration := time.Since(start)
		MetricLastRunDurationSeconds.Set(duration.Seconds())
		MetricRunDurationSeconds.Observe(duration.Seconds())
	}()
	logger := c.Logger.Sugar()
	endpoints := make([]*endpoint.Endpoint, 0)
	for _, s := range c.Sources {
		e, err := s.Source.Endpoints(ctx)
		if err != nil {
			logger.Errorw(
				"error getting endpoints from source",
				"source", s.Name,
				"err", err,
			)
			errors = multierr.Append(errors, err)
			MetricSourceUp.WithLabelValues(s.Name).Set(0)
		} else {
			MetricSourceUp.WithLabelValues(s.Name).Set(1)
		}
		MetricEndpoints.WithLabelValues(s.Name).Set(float64(len(e)))
		for i := range e {
			if e[i].SourceProperties == nil {
				e[i].SourceProperties = map[string]any{}
			}
			e[i].SourceProperties["source"] = s.Name
		}
		endpoints = append(endpoints, e...)
	}
	if errors != nil {
		// Bail early, and set all providers to down
		for _, p := range c.Providers {
			MetricProviderUp.WithLabelValues(p.Name).Set(0)
		}
		return errors
	}
	for _, endpoint := range endpoints {
		logger.Infow(
			"registered new endpoint",
			"hostname", endpoint.Hostname,
			"ipv4", endpoint.IPv4s,
			"ipv6", endpoint.IPv6s,
			"ttl", endpoint.RecordTTL,
			"source_properties", endpoint.SourceProperties,
			"provider_properties", endpoint.ProviderProperties,
		)
	}
	dryRun, ok := ctx.Value(configtypes.DryRunContextKey).(bool)
	if !ok {
		dryRun = false
	}
	if !dryRun {
		for _, p := range c.Providers {
			err := p.Provider.UpdateEndpoints(ctx, endpoints)
			if err != nil {
				logger.Errorw(
					"error updating endpoints with provider",
					"provider", p.Name,
					"err", err,
				)
				errors = multierr.Append(errors, err)
				MetricProviderUp.WithLabelValues(p.Name).Set(0)
			} else {
				MetricProviderUp.WithLabelValues(p.Name).Set(1)
			}
		}
	}
	return errors
}

// ScheduleRunOnce makes sure execution happens at most once per interval.
func (c *Controller) ScheduleRunOnce(now time.Time) {
	c.nextRunAtMux.Lock()
	defer c.nextRunAtMux.Unlock()
}

func (c *Controller) ShouldRunOnce(now time.Time) bool {
	c.nextRunAtMux.Lock()
	defer c.nextRunAtMux.Unlock()
	if now.Before(c.nextRunAt) {
		return false
	}
	c.nextRunAt = now.Add(c.Interval)
	return true
}

// Run runs RunOnce in a loop with a delay until context is canceled.
func (c *Controller) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if c.ShouldRunOnce(time.Now()) {
			if err := c.RunOnce(ctx); err != nil {
				c.Logger.Sugar().Errorf("controller.Run error: %v", err)
			}
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			c.Logger.Info("Terminating main controller loop")
			return
		}
	}
}
