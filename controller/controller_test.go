package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	endpoints     []*endpoint.Endpoint
	endpointsFunc func(ctx context.Context) ([]*endpoint.Endpoint, error)
}

func (s *mockSource) Endpoints(ctx context.Context) ([]*endpoint.Endpoint, error) {
	if s.endpointsFunc != nil {
		return s.endpointsFunc(ctx)
	}
	return s.endpoints, nil
}

type mockProvider struct {
	endpoints           []*endpoint.Endpoint
	updateEndpointsFunc func(ctx context.Context, endpoints []*endpoint.Endpoint) error
}

func (p *mockProvider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	if p.updateEndpointsFunc != nil {
		return p.updateEndpointsFunc(ctx, endpoints)
	}
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
				Sources:   []source.NamedSource{
					{Name: "mock_source", Source: s},
				},
				Providers: []provider.NamedProvider{
					{Name: "mock_provider", Provider: p},
				},
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

func TestMultierr(t *testing.T) {
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
	sourceCalled := false
	sourceOk := &mockSource{
		endpoints: endpoints,
		endpointsFunc: func(ctx context.Context) ([]*endpoint.Endpoint, error) {
			sourceCalled = true
			return endpoints, nil
		},
	}
	sourceErrored := &mockSource{
		endpoints: endpoints,
		endpointsFunc: func(ctx context.Context) ([]*endpoint.Endpoint, error) {
			return nil, errors.New("source error")
		},
	}
	providerCalled := false
	providerOk := &mockProvider{
		updateEndpointsFunc: func(ctx context.Context, endpoints []*endpoint.Endpoint) error {
			providerCalled = true
			return nil
		},
	}
	providerErrored := &mockProvider{
		updateEndpointsFunc: func(ctx context.Context, endpoints []*endpoint.Endpoint) error {
			return errors.New("provider error")
		},
	}
	ctrl := &Controller{
		// always load errored first so it runs first
		Sources:   []source.NamedSource{
			{Name: "source_errored", Source: sourceErrored},
			{Name: "source_ok", Source: sourceOk},
		},
		Providers: []provider.NamedProvider{
			{Name: "provider_errored", Provider: providerErrored},
			{Name: "provider_ok", Provider: providerOk},
		},
		Interval:  1 * time.Minute,
		Logger:    zap.NewNop(),
	}

	// First run should exit early on a source encountering an error and not call
	// providers
	ctx := context.Background()
	err := ctrl.RunOnce(ctx)
	assert.ErrorContainsf(t, err, "source error", "source returned error")
	assert.True(t, sourceCalled)
	assert.False(t, providerCalled)

	// Reset
	sourceCalled = false
	providerCalled = false

	// Let sources work this time. Providers should be called and it should
	// return an error from the errored provider.
	ctrl.Sources = []source.NamedSource{{Name: "source_ok", Source: sourceOk}}
	err = ctrl.RunOnce(ctx)
	assert.ErrorContainsf(t, err, "provider error", "provider returned error")
	assert.True(t, sourceCalled)
	assert.True(t, providerCalled)
}
