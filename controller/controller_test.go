package controller

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/provider"
	"github.com/sapslaj/zonepop/source"
)

func TestShouldRunOnce(t *testing.T) {
	ctrl := &Controller{
		Interval: 10 * time.Minute,
	}

	now := time.Now()

	if !ctrl.ShouldRunOnce(now) {
		t.Errorf("controller.ShouldRunOnce(now) should be true on first run")
	}
	if ctrl.ShouldRunOnce(now) {
		t.Errorf("controller.ShouldRunOnce(now) should be false on second run")
	}

	now = now.Add(10 * time.Second)
	if ctrl.ShouldRunOnce(now) {
		t.Fatalf("controller.ShouldRunOnce(now) should be false after only a short time after first schedule")
	}

	now = now.Add(10 * time.Minute)
	if !ctrl.ShouldRunOnce(now) {
		t.Fatalf("controller.ShouldRunOnce(now) should be true after the full interval is elapsed")
	}
}

type mockSource struct {
	endpoints []*endpoint.Endpoint
}

func (s *mockSource) Endpoints(ctx context.Context) ([]*endpoint.Endpoint, error) {
	return s.endpoints, nil
}

type mockProvider struct {
	endpoints []*endpoint.Endpoint
}

func (p *mockProvider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	p.endpoints = endpoints
	return nil
}

func TestRunOnce(t *testing.T) {
	for n, dryRun := range map[string]bool{"realRun": false, "dryRun": true} {
		t.Run(n, func(t *testing.T) {
			endpoints := []*endpoint.Endpoint{
				{
					Hostname:           "test-host",
					IPv4s:              []string{"192.0.2.0"},
					IPv6s:              nil,
					RecordTTL:          60,
					SourceProperties:   nil,
					ProviderProperties: nil,
				},
			}
			s := &mockSource{
				endpoints: endpoints,
			}
			p := &mockProvider{}
			ctrl := &Controller{
				Sources:   []source.Source{s},
				Providers: []provider.Provider{p},
				Interval:  1 * time.Minute,
				Logger:    zap.NewNop(),
			}

			ctx := context.Background()
			if dryRun {
				ctx = context.WithValue(ctx, configtypes.DryRunContextKey, true)
			}
			err := ctrl.RunOnce(ctx)
			if err != nil {
				t.Fatalf("controller.RunOnce returned error: %v", err)
			}

			if dryRun && len(p.endpoints) != 0 {
				t.Fatalf("dry run should have not updated endpoints (len(p.endpoints) == %d but expected 0)", len(p.endpoints))
			}
			if !dryRun && len(p.endpoints) != 1 {
				t.Fatalf("endpoints weren't updated correctly in provider (len(p.endpoints) == %d but expected 1)", len(p.endpoints))
			}
		})
	}
}
