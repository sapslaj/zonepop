package controller

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/sapslaj/zonepop/config"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/provider"
	"github.com/sapslaj/zonepop/source"
)

type Controller struct {
	Sources   []source.Source
	Providers []provider.Provider
	// The interval between individual synchronizations
	Interval time.Duration
	// The nextRunAt used for throttling and batching reconciliation
	nextRunAt time.Time
	// The nextRunAtMux is for atomic updating of nextRunAt
	nextRunAtMux sync.Mutex
}

// RunOnce runs a single iteration of a reconciliation loop.
func (c *Controller) RunOnce(ctx context.Context) error {
	endpoints := make([]*endpoint.Endpoint, 0)
	for _, s := range c.Sources {
		e, err := s.Endpoints(ctx)
		if err != nil {
			return err
		}
		endpoints = append(endpoints, e...)
	}
	for _, endpoint := range endpoints {
		log.Printf(
			"hostname=%v ipv4=%v ipv6=%v ttl=%v source_properties=%v provider_properties=%v",
			endpoint.Hostname,
			endpoint.IPv4s,
			endpoint.IPv6s,
			endpoint.RecordTTL,
			endpoint.SourceProperties,
			endpoint.ProviderProperties,
		)
	}
	dryRun, ok := ctx.Value(config.DryRunContextKey).(bool)
	if !ok {
		dryRun = false
	}
	if !dryRun {
		for _, p := range c.Providers {
			err := p.UpdateEndpoints(ctx, endpoints)
			if err != nil {
				return err
			}
		}
	}
	return nil
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

// Run runs RunOnce in a loop with a delay until context is canceled
func (c *Controller) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if c.ShouldRunOnce(time.Now()) {
			if err := c.RunOnce(ctx); err != nil {
				log.Printf("controller.Run error: %v", err)
			}
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			log.Print("Terminating main controller loop")
			return
		}
	}
}
