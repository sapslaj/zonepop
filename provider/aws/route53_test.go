package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/utils"
)

type mockRoute53Client struct {
	GetHostedZoneCalls []struct {
		Input  *route53.GetHostedZoneInput
		OptFns []func(*route53.Options)
	}
	GetHostedZoneOutput *route53.GetHostedZoneOutput
	GetHostedZoneError  error

	ChangeResourceRecordSetsCalls []struct {
		Input  *route53.ChangeResourceRecordSetsInput
		OptFns []func(*route53.Options)
	}
	ChangeResourceRecordSetsOutput *route53.ChangeResourceRecordSetsOutput
	ChangeResourceRecordSetsError  error
}

func (m *mockRoute53Client) GetHostedZone(
	ctx context.Context,
	params *route53.GetHostedZoneInput,
	optFns ...func(*route53.Options),
) (*route53.GetHostedZoneOutput, error) {
	if m.GetHostedZoneCalls == nil {
		m.GetHostedZoneCalls = make([]struct {
			Input  *route53.GetHostedZoneInput
			OptFns []func(*route53.Options)
		}, 0)
	}
	m.GetHostedZoneCalls = append(m.GetHostedZoneCalls, struct {
		Input  *route53.GetHostedZoneInput
		OptFns []func(*route53.Options)
	}{
		Input:  params,
		OptFns: optFns,
	})
	err := m.GetHostedZoneError
	out := m.GetHostedZoneOutput
	if out == nil {
		switch aws.ToString(params.Id) {
		case "ex-ipv4-reverse":
			out = &route53.GetHostedZoneOutput{
				HostedZone: &types.HostedZone{
					CallerReference:        aws.String(""),
					Id:                     aws.String("ex-ipv4-reverse"),
					Name:                   aws.String("0.192.in-addr.arpa."),
					ResourceRecordSetCount: aws.Int64(69),
					Config: &types.HostedZoneConfig{
						Comment:     aws.String(""),
						PrivateZone: false,
					},
				},
				DelegationSet: &types.DelegationSet{
					NameServers: []string{
						"ns-01.awsdns-01.com",
						"ns-02.awsdns-02.com",
						"ns-03.awsdns-03.com",
						"ns-04.awsdns-04.com",
					},
				},
			}
		case "ex-ipv6-reverse":
			out = &route53.GetHostedZoneOutput{
				HostedZone: &types.HostedZone{
					CallerReference:        aws.String(""),
					Id:                     aws.String("ex-ipv4-reverse"),
					Name:                   aws.String("0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa."),
					ResourceRecordSetCount: aws.Int64(69),
					Config: &types.HostedZoneConfig{
						Comment:     aws.String(""),
						PrivateZone: false,
					},
				},
				DelegationSet: &types.DelegationSet{
					NameServers: []string{
						"ns-01.awsdns-01.com",
						"ns-02.awsdns-02.com",
						"ns-03.awsdns-03.com",
						"ns-04.awsdns-04.com",
					},
				},
			}
		default:
			out = &route53.GetHostedZoneOutput{
				HostedZone: &types.HostedZone{
					CallerReference:        aws.String(""),
					Id:                     aws.String("Z2FDTNDATAQYW2"),
					Name:                   aws.String("example.com."),
					ResourceRecordSetCount: aws.Int64(69),
					Config: &types.HostedZoneConfig{
						Comment:     aws.String(""),
						PrivateZone: false,
					},
				},
				DelegationSet: &types.DelegationSet{
					NameServers: []string{
						"ns-01.awsdns-01.com",
						"ns-02.awsdns-02.com",
						"ns-03.awsdns-03.com",
						"ns-04.awsdns-04.com",
					},
				},
			}
		}
	}
	return out, err
}

func (m *mockRoute53Client) ChangeResourceRecordSets(
	ctx context.Context,
	params *route53.ChangeResourceRecordSetsInput,
	optFns ...func(*route53.Options),
) (*route53.ChangeResourceRecordSetsOutput, error) {
	if m.ChangeResourceRecordSetsCalls == nil {
		m.ChangeResourceRecordSetsCalls = make([]struct {
			Input  *route53.ChangeResourceRecordSetsInput
			OptFns []func(*route53.Options)
		}, 0)
	}
	m.ChangeResourceRecordSetsCalls = append(m.ChangeResourceRecordSetsCalls, struct {
		Input  *route53.ChangeResourceRecordSetsInput
		OptFns []func(*route53.Options)
	}{
		Input:  params,
		OptFns: optFns,
	})
	err := m.ChangeResourceRecordSetsError
	out := m.ChangeResourceRecordSetsOutput
	if out == nil {
		out = &route53.ChangeResourceRecordSetsOutput{
			ChangeInfo: &types.ChangeInfo{
				Id:          aws.String(""),
				Status:      "PENDING",
				SubmittedAt: aws.Time(time.Now()),
			},
		}
	}
	return out, err
}

func (m *mockRoute53Client) ListResourceRecordSets(
	ctx context.Context,
	params *route53.ListResourceRecordSetsInput,
	optFns ...func(*route53.Options),
) (*route53.ListResourceRecordSetsOutput, error) {
	return &route53.ListResourceRecordSetsOutput{
		IsTruncated: false,
		MaxItems: aws.Int32(0),
		ResourceRecordSets: []types.ResourceRecordSet{},
	}, nil
}

func newMockNewRoute53Provider(
	client Route53Client,
	logger *zap.Logger,
	providerConfig Route53ProviderConfig,
	forwardLookupFilter configtypes.EndpointFilterFunc,
	reverseLookupFilter configtypes.EndpointFilterFunc,
) (*route53Provider, error) {
	return &route53Provider{
		config:              providerConfig,
		forwardLookupFilter: forwardLookupFilter,
		reverseLookupFilter: reverseLookupFilter,
		client:              client,
		logger:              logger,
	}, nil
}

func TestUpdateEndpoints_Minimal(t *testing.T) {
	mockClient := &mockRoute53Client{}
	config := Route53ProviderConfig{
		RecordSuffix:  ".example.com",
		ForwardZoneID: "Z2FDTNDATAQYW2",
	}
	p, err := newMockNewRoute53Provider(
		mockClient,
		zap.NewExample(),
		config,
		configtypes.DefaultEndpointFilterFunc,
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoErrorf(t, err, "something went wrong creating mock provider: %v", err)

	endpoints := []*endpoint.Endpoint{
		{
			Hostname:           "test-host",
			IPv4s:              []string{"192.0.2.1"},
			IPv6s:              []string{},
			RecordTTL:          69,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}
	err = p.UpdateEndpoints(context.Background(), endpoints)
	require.NoErrorf(t, err, "error updating endpoints: %v", err)

	require.Len(
		t,
		mockClient.ChangeResourceRecordSetsCalls,
		1,
		"mockRoute53Client.ChangeResourceRecordSets was never called",
	)
	changes := mockClient.ChangeResourceRecordSetsCalls[0].Input.ChangeBatch.Changes

	assert.Len(t, changes, 1)
	for _, change := range changes {
		assert.Equal(t, types.ChangeActionUpsert, change.Action)
		assert.Equal(t, int64(69), aws.ToInt64(change.ResourceRecordSet.TTL))
		assert.Equal(t, "test-host.example.com", aws.ToString(change.ResourceRecordSet.Name))
		assert.Equal(t, types.RRTypeA, change.ResourceRecordSet.Type)
		assert.Len(t, change.ResourceRecordSet.ResourceRecords, 1)
		assert.Equal(t, "192.0.2.1", aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value))
	}
}

func TestUpdateEndpoints_ForwardAndReverse(t *testing.T) {
	mockClient := &mockRoute53Client{}
	config := Route53ProviderConfig{
		RecordSuffix:      ".example.com",
		ForwardZoneID:     "ex-forward",
		Ipv4ReverseZoneID: "ex-ipv4-reverse",
		Ipv6ReverseZoneID: "ex-ipv6-reverse",
	}
	p, err := newMockNewRoute53Provider(
		mockClient,
		zap.NewExample(),
		config,
		configtypes.DefaultEndpointFilterFunc,
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoErrorf(t, err, "something went wrong creating mock provider: %v", err)

	endpoints := []*endpoint.Endpoint{
		{
			Hostname:           "test-host",
			IPv4s:              []string{"192.0.2.1"},
			IPv6s:              []string{"2001:db8::1"},
			RecordTTL:          69,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}
	err = p.UpdateEndpoints(context.Background(), endpoints)
	require.NoErrorf(t, err, "error updating endpoints: %v", err)

	require.Len(
		t,
		mockClient.ChangeResourceRecordSetsCalls,
		3,
		"mockRoute53Client.ChangeResourceRecordSets was called an incorrect number of times",
	)
	for _, call := range mockClient.ChangeResourceRecordSetsCalls {
		changes := call.Input.ChangeBatch.Changes
		for _, change := range changes {
			assert.Equal(t, types.ChangeActionUpsert, change.Action)
			assert.Equal(t, int64(69), aws.ToInt64(change.ResourceRecordSet.TTL))
			assert.Len(t, change.ResourceRecordSet.ResourceRecords, 1)
			switch change.ResourceRecordSet.Type {
			case types.RRTypeA:
				assert.Len(t, changes, 2)
				assert.Equal(t, "test-host.example.com", aws.ToString(change.ResourceRecordSet.Name))
				assert.Equal(t, "192.0.2.1", aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value))
			case types.RRTypeAaaa:
				assert.Len(t, changes, 2)
				assert.Equal(t, "test-host.example.com", aws.ToString(change.ResourceRecordSet.Name))
				assert.Equal(t, "2001:db8::1", aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value))
			case types.RRTypePtr:
				assert.Len(t, changes, 1)
				assert.Equal(t, "test-host.example.com", aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value))
				record := aws.ToString(change.ResourceRecordSet.Name)
				if utils.All([]bool{
					record != "1.2.0.192.in-addr.arpa.",
					record != "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.",
				}) {
					t.Errorf("Unexpected ResourceRecordSet Name for PTR: %q", record)
				}
			default:
				t.Errorf("Unexpected ResourceRecordSet Type: expected A or AAAA, got %v", change.ResourceRecordSet.Type)
			}
		}
	}
}

func TestUpdateEndpoints_Filtering(t *testing.T) {
	mockClient := &mockRoute53Client{}
	config := Route53ProviderConfig{
		RecordSuffix:      ".example.com",
		ForwardZoneID:     "ex-forward",
		Ipv4ReverseZoneID: "ex-ipv4-reverse",
		Ipv6ReverseZoneID: "ex-ipv6-reverse",
	}
	forwardLookupFilterFunc := func(e *endpoint.Endpoint) bool {
		return e.Hostname == "only-forward"
	}
	reverseLookupFilterFunc := func(e *endpoint.Endpoint) bool {
		return e.Hostname == "only-reverse"
	}
	p, err := newMockNewRoute53Provider(
		mockClient,
		zap.NewExample(),
		config,
		forwardLookupFilterFunc,
		reverseLookupFilterFunc,
	)
	require.NoErrorf(t, err, "something went wrong creating mock provider: %v", err)

	endpoints := []*endpoint.Endpoint{
		{
			Hostname:           "only-forward",
			IPv4s:              []string{"192.0.2.1"},
			IPv6s:              []string{"2001:db8::1"},
			RecordTTL:          69,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
		{
			Hostname:           "only-reverse",
			IPv4s:              []string{"192.0.2.2"},
			IPv6s:              []string{"2001:db8::2"},
			RecordTTL:          69,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}
	err = p.UpdateEndpoints(context.Background(), endpoints)
	require.NoErrorf(t, err, "error updating endpoints: %v", err)

	require.Len(
		t,
		mockClient.ChangeResourceRecordSetsCalls,
		3,
		"mockRoute53Client.ChangeResourceRecordSets was called an incorrect number of times",
	)
	for _, call := range mockClient.ChangeResourceRecordSetsCalls {
		changes := call.Input.ChangeBatch.Changes
		for _, change := range changes {
			assert.Equal(t, types.ChangeActionUpsert, change.Action)
			assert.Equal(t, int64(69), aws.ToInt64(change.ResourceRecordSet.TTL))
			assert.Len(t, change.ResourceRecordSet.ResourceRecords, 1)
			switch change.ResourceRecordSet.Type {
			case types.RRTypeA:
				fallthrough
			case types.RRTypeAaaa:
				assert.Len(t, changes, 2)
				assert.Equal(t, "only-forward.example.com", aws.ToString(change.ResourceRecordSet.Name))
			case types.RRTypePtr:
				assert.Len(t, changes, 1)
				assert.Equal(t, "only-reverse.example.com", aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value))
			default:
				t.Errorf("Unexpected ResourceRecordSet Type: expected A or AAAA, got %v", change.ResourceRecordSet.Type)
			}
		}
	}
}

func TestUpdateEndpoints_NoChanges(t *testing.T) {
	mockClient := &mockRoute53Client{}
	config := Route53ProviderConfig{
		RecordSuffix:      ".example.com",
		ForwardZoneID:     "ex-forward",
		Ipv4ReverseZoneID: "ex-ipv4-reverse",
		Ipv6ReverseZoneID: "ex-ipv6-reverse",
	}
	p, err := newMockNewRoute53Provider(
		mockClient,
		zap.NewExample(),
		config,
		configtypes.DefaultEndpointFilterFunc,
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoErrorf(t, err, "something went wrong creating mock provider: %v", err)

	endpoints := []*endpoint.Endpoint{
		{
			Hostname:           "test-host",
			IPv4s:              []string{},
			IPv6s:              []string{},
			RecordTTL:          69,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}
	err = p.UpdateEndpoints(context.Background(), endpoints)
	require.NoErrorf(t, err, "error updating endpoints: %v", err)

	require.Len(
		t,
		mockClient.ChangeResourceRecordSetsCalls,
		0,
		"mockRoute53Client.ChangeResourceRecordSets was called when it should not have been",
	)
}

func TestUpdateEndpoints_ErrorUpdatingZone(t *testing.T) {
	expectedErr := errors.New("injected error")
	mockClient := &mockRoute53Client{
		GetHostedZoneError: expectedErr,
	}
	config := Route53ProviderConfig{
		RecordSuffix:      ".example.com",
		ForwardZoneID:     "ex-forward",
		Ipv4ReverseZoneID: "ex-ipv4-reverse",
		Ipv6ReverseZoneID: "ex-ipv6-reverse",
	}
	p, err := newMockNewRoute53Provider(
		mockClient,
		zap.NewExample(),
		config,
		configtypes.DefaultEndpointFilterFunc,
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoErrorf(t, err, "something went wrong creating mock provider: %v", err)

	endpoints := []*endpoint.Endpoint{
		{
			Hostname:           "test-host",
			IPv4s:              []string{"192.0.2.1"},
			IPv6s:              []string{"2001:db8::1"},
			RecordTTL:          69,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}
	err = p.UpdateEndpoints(context.Background(), endpoints)
	assert.ErrorIs(t, err, expectedErr)

	assert.Len(
		t,
		mockClient.ChangeResourceRecordSetsCalls,
		0,
		"mockRoute53Client.ChangeResourceRecordSets was called when it should not have been",
	)
}

func TestUpdateEndpoints_NoHostname(t *testing.T) {
	mockClient := &mockRoute53Client{}
	config := Route53ProviderConfig{
		RecordSuffix:      ".example.com",
		ForwardZoneID:     "ex-forward",
		Ipv4ReverseZoneID: "ex-ipv4-reverse",
		Ipv6ReverseZoneID: "ex-ipv6-reverse",
	}
	p, err := newMockNewRoute53Provider(
		mockClient,
		zap.NewExample(),
		config,
		configtypes.DefaultEndpointFilterFunc,
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoErrorf(t, err, "something went wrong creating mock provider: %v", err)

	endpoints := []*endpoint.Endpoint{
		{
			Hostname:           "",
			IPv4s:              []string{"192.0.2.1"},
			IPv6s:              []string{"2001:db8::1"},
			RecordTTL:          69,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}
	err = p.UpdateEndpoints(context.Background(), endpoints)
	require.NoErrorf(t, err, "error updating endpoints: %v", err)

	require.Len(
		t,
		mockClient.ChangeResourceRecordSetsCalls,
		2,
		"mockRoute53Client.ChangeResourceRecordSets was called an incorrect number of times",
	)

	for _, call := range mockClient.ChangeResourceRecordSetsCalls {
		changes := call.Input.ChangeBatch.Changes
		for _, change := range changes {
			assert.Equal(t, types.ChangeActionUpsert, change.Action)
			assert.Equal(t, int64(69), aws.ToInt64(change.ResourceRecordSet.TTL))
			assert.Len(t, change.ResourceRecordSet.ResourceRecords, 1)
			switch change.ResourceRecordSet.Type {
			case types.RRTypeA:
				assert.Len(t, changes, 2)
				assert.Equal(t, "ip-192-0-2-1.example.com", aws.ToString(change.ResourceRecordSet.Name))
				assert.Equal(t, "192.0.2.1", aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value))
			case types.RRTypeAaaa:
				assert.Len(t, changes, 2)
				assert.Equal(t, "ip-192-0-2-1.example.com", aws.ToString(change.ResourceRecordSet.Name))
				assert.Equal(t, "2001:db8::1", aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value))
			case types.RRTypePtr:
				assert.Len(t, changes, 1)
				assert.Equal(t, "ip-192-0-2-1.example.com", aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value))
				record := aws.ToString(change.ResourceRecordSet.Name)
				if utils.All([]bool{
					record != "1.2.0.192.in-addr.arpa.",
					record != "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.",
				}) {
					t.Errorf("Unexpected ResourceRecordSet Name for PTR: %q", record)
				}
			default:
				t.Errorf("Unexpected ResourceRecordSet Type: expected A or AAAA, got %v", change.ResourceRecordSet.Type)
			}
		}
	}
}

func TestUpdateEndpoints_NoHostnameNoIpv4(t *testing.T) {
	mockClient := &mockRoute53Client{}
	config := Route53ProviderConfig{
		RecordSuffix:      ".example.com",
		ForwardZoneID:     "ex-forward",
		Ipv4ReverseZoneID: "ex-ipv4-reverse",
		Ipv6ReverseZoneID: "ex-ipv6-reverse",
	}
	p, err := newMockNewRoute53Provider(
		mockClient,
		zap.NewExample(),
		config,
		configtypes.DefaultEndpointFilterFunc,
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoErrorf(t, err, "something went wrong creating mock provider: %v", err)

	endpoints := []*endpoint.Endpoint{
		{
			Hostname:           "",
			IPv4s:              []string{},
			IPv6s:              []string{},
			RecordTTL:          69,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}
	err = p.UpdateEndpoints(context.Background(), endpoints)
	require.NoErrorf(t, err, "error updating endpoints: %v", err)

	require.Len(
		t,
		mockClient.ChangeResourceRecordSetsCalls,
		0,
		"mockRoute53Client.ChangeResourceRecordSets was called an incorrect number of times",
	)
}

func TestUpdateEndpoints_NonFittingReverseZones(t *testing.T) {
	mockClient := &mockRoute53Client{}
	config := Route53ProviderConfig{
		RecordSuffix:      ".example.com",
		ForwardZoneID:     "ex-forward",
		Ipv4ReverseZoneID: "ex-ipv4-reverse",
		Ipv6ReverseZoneID: "ex-ipv6-reverse",
	}
	p, err := newMockNewRoute53Provider(
		mockClient,
		zap.NewExample(),
		config,
		configtypes.DefaultEndpointFilterFunc,
		configtypes.DefaultEndpointFilterFunc,
	)
	require.NoErrorf(t, err, "something went wrong creating mock provider: %v", err)

	endpoints := []*endpoint.Endpoint{
		{
			Hostname:           "test-host",
			IPv4s:              []string{"198.51.100.1"},
			IPv6s:              []string{"2001:db8:ffff:ffff::1"},
			RecordTTL:          69,
			SourceProperties:   nil,
			ProviderProperties: nil,
		},
	}
	err = p.UpdateEndpoints(context.Background(), endpoints)
	require.NoErrorf(t, err, "error updating endpoints: %v", err)

	require.Len(
		t,
		mockClient.ChangeResourceRecordSetsCalls,
		1,
		"mockRoute53Client.ChangeResourceRecordSets was called an incorrect number of times",
	)
}
