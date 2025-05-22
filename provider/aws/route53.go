package aws

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/config/configtypes"
	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/pkg/rdns"
	"github.com/sapslaj/zonepop/pkg/utils"
	"github.com/sapslaj/zonepop/provider"
)

type Route53ProviderConfig struct {
	RecordSuffix         string
	ForwardZoneID        string
	ForwardZoneName      string
	Ipv4ReverseZoneID    string
	Ipv4ReverseZoneName  string
	Ipv6ReverseZoneID    string
	Ipv6ReverseZoneName  string
	CleanForwardZone     bool
	CleanIPv4ReverseZone bool
	CleanIPv6ReverseZone bool
}

type Route53Client interface {
	// Implementation of [github.com/aws/aws-sdk-go-v2/service/route53.Client.GetHostedZone]
	GetHostedZone(
		ctx context.Context,
		params *route53.GetHostedZoneInput,
		optFns ...func(*route53.Options),
	) (*route53.GetHostedZoneOutput, error)
	// Implementation of [github.com/aws/aws-sdk-go-v2/service/route53.Client.ChangeResourceRecordSets]
	ChangeResourceRecordSets(
		ctx context.Context,
		params *route53.ChangeResourceRecordSetsInput,
		optFns ...func(*route53.Options),
	) (*route53.ChangeResourceRecordSetsOutput, error)
	ListResourceRecordSets(
		ctx context.Context,
		params *route53.ListResourceRecordSetsInput,
		optFns ...func(*route53.Options),
	) (*route53.ListResourceRecordSetsOutput, error)
}

type route53Provider struct {
	config              Route53ProviderConfig
	forwardLookupFilter configtypes.EndpointFilterFunc
	reverseLookupFilter configtypes.EndpointFilterFunc
	client              Route53Client
	logger              *zap.Logger
}

func getRoute53ZoneName(ctx context.Context, client Route53Client, zoneID string) (string, error) {
	ghzout, err := client.GetHostedZone(ctx, &route53.GetHostedZoneInput{
		Id: aws.String(zoneID),
	})
	if err != nil {
		return "", fmt.Errorf("could not get hosted zone information for %s: %w", zoneID, err)
	}
	return aws.ToString(ghzout.HostedZone.Name), nil
}

func NewRoute53Provider(
	providerConfig Route53ProviderConfig,
	forwardLookupFilter configtypes.EndpointFilterFunc,
	reverseLookupFilter configtypes.EndpointFilterFunc,
) (provider.Provider, error) {
	client, err := defaultR53Client()
	if err != nil {
		return nil, fmt.Errorf("could not get default Route53 client: %w", err)
	}
	p := &route53Provider{
		config:              providerConfig,
		forwardLookupFilter: forwardLookupFilter,
		reverseLookupFilter: reverseLookupFilter,
		client:              client,
		logger:              log.MustNewLogger().Named("aws_route53_provider"),
	}
	return p, nil
}

func (p *route53Provider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	forwardEndpoints := utils.Filter(p.forwardLookupFilter, endpoints)
	err := p.updateForward(ctx, forwardEndpoints)
	if err != nil {
		p.logger.Sugar().Errorw("failed to update forward lookup zone", "err", err)
		return err
	}
	reverseEndpoints := utils.Filter(p.reverseLookupFilter, endpoints)
	err = p.updateIPv4Reverse(ctx, reverseEndpoints)
	if err != nil {
		p.logger.Sugar().Errorw("failed to update IPv4 reverse lookup zone", "err", err)
		return err
	}
	err = p.updateIPv6Reverse(ctx, reverseEndpoints)
	if err != nil {
		p.logger.Sugar().Errorw("failed to update IPv6 reverse lookup zone", "err", err)
		return err
	}
	return nil
}

func (p *route53Provider) updateForward(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	if p.config.ForwardZoneID == "" {
		p.logger.Warn("Forward lookup zone disabled")
		return nil
	}
	if p.config.ForwardZoneID != "" && p.config.ForwardZoneName == "" {
		forwardZoneName, err := getRoute53ZoneName(ctx, p.client, p.config.ForwardZoneID)
		if err != nil {
			p.logger.Sugar().Errorw("could not get Route53 zone name", "err", err)
			return err
		}
		p.config.ForwardZoneName = forwardZoneName
	}

	if p.config.RecordSuffix == "" {
		p.config.RecordSuffix = "." + p.config.ForwardZoneName
	}

	hostnameEndpoints := make(map[string][]*endpoint.Endpoint)
	for _, endpoint := range endpoints {
		if endpoint.Hostname == "" {
			continue
		}
		hostnameEndpoints[endpoint.Hostname] = append(hostnameEndpoints[endpoint.Hostname], endpoint)
	}

	if p.config.CleanForwardZone {
		p.logger.Info("cleanup: cleaning forward lookup zone")
		p.cleanupZone(ctx, p.config.ForwardZoneID, []types.RRType{types.RRTypeA, types.RRTypeAaaa}, func(name string) bool {
			for hostname := range hostnameEndpoints {
				if hostname == name {
					return true
				}
			}
			return false
		})
	}

	changes := make([]types.Change, 0)
	for hostname, endpoints := range hostnameEndpoints {
		fullHostname := utils.DNSSafeName(hostname) + p.config.RecordSuffix
		hostnameLogger := p.logger.Sugar().With(
			"hostname", hostname,
			"full_hostname", fullHostname,
		)
		ipv4 := make([]string, 0)
		ipv6 := make([]string, 0)
		if len(endpoints) == 0 {
			hostnameLogger.Warnf("No endpoints for hostname %q", hostname)
			continue
		}
		for _, endpoint := range endpoints {
			// make sure its deduped otherwise Route53 will get angry with us
			for _, newIpv4 := range endpoint.IPv4s {
				if !slices.Contains(ipv4, newIpv4) {
					ipv4 = append(ipv4, newIpv4)
				}
			}
			for _, newIpv6 := range endpoint.IPv6s {
				if !slices.Contains(ipv6, newIpv6) {
					ipv6 = append(ipv6, newIpv6)
				}
			}
		}
		ttl := endpoints[0].RecordTTL
		hostnameLogger = hostnameLogger.With("ttl", ttl)
		if len(ipv4) > 0 {
			hostnameLogger.With(
				"ipv4", ipv4,
				"record_type", "A",
			).Infof("adding IPv4 (A) record %q for hostname %q", ipv4, hostname)
			changes = append(changes, p.dnsChange(
				fullHostname,
				ipv4,
				"A",
				endpoints[0].RecordTTL,
			))
		}
		if len(ipv6) > 0 {
			hostnameLogger.With(
				"ipv6", ipv6,
				"record_type", "AAAA",
			).Infof("adding IPv6 (AAAA) record %q for hostname %q", ipv6, hostname)
			changes = append(changes, p.dnsChange(
				fullHostname,
				ipv6,
				"AAAA",
				endpoints[0].RecordTTL,
			))
		}
	}
	if len(changes) == 0 {
		p.logger.Info("No forward lookup changes.")
		return nil
	}

	_, err := p.client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.config.ForwardZoneID),
		ChangeBatch:  &types.ChangeBatch{Changes: changes},
	})

	return err
}

func (p *route53Provider) updateIPv4Reverse(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	if p.config.Ipv4ReverseZoneID == "" {
		p.logger.Warn("IPv4 reverse lookup zone disabled")
		return nil
	}
	if p.config.Ipv4ReverseZoneName == "" {
		ipv4ReverseZoneName, err := getRoute53ZoneName(ctx, p.client, p.config.Ipv4ReverseZoneID)
		if err != nil {
			p.logger.Sugar().Errorw("could not get Route53 zone name", "err", err)
			return err
		}
		p.config.Ipv4ReverseZoneName = ipv4ReverseZoneName
	}

	if p.config.CleanIPv4ReverseZone {
		p.logger.Info("cleanup: cleaning IPv4 reverse lookup zone")
		p.cleanupZone(ctx, p.config.Ipv4ReverseZoneID, []types.RRType{types.RRTypePtr}, func(name string) bool {
			for _, endpoint := range endpoints {
				for _, ipv4 := range endpoint.IPv4s {
					ptr, err := rdns.ReverseAddr(ipv4)
					if err != nil {
						p.logger.Sugar().Errorw("cleanup: could not determine PTR record", "err", err)
						break
					}
					if ptr == name {
						return true
					}
				}
			}
			return false
		})
	}

	ptrHostnames := map[string]string{}
	changes := make([]types.Change, 0)
	for _, endpoint := range endpoints {
		hostname := endpoint.Hostname
		for _, ipv4 := range endpoint.IPv4s {
			addrLogger := p.logger.Sugar().With(
				"addr", ipv4,
				"zone", p.config.Ipv4ReverseZoneName,
				"ttl", endpoint.RecordTTL,
			)
			if hostname == "" {
				hostname = "ip-" + strings.ReplaceAll(ipv4, ".", "-")
				addrLogger.Infof("No hostname defined for endpoint, using generated hostname of %s", hostname)
			}
			fullHostname := utils.DNSSafeName(hostname) + p.config.RecordSuffix
			addrLogger = addrLogger.With(
				"hostname", hostname,
				"full_hostname", fullHostname,
			)
			fits, err := rdns.FitsInReverseZone(ipv4, p.config.Ipv4ReverseZoneName)
			if err != nil {
				addrLogger.Errorw(
					"could not determine if address fits in reverse zone",
					"err", err,
				)
				return err
			}
			if !fits {
				addrLogger.Warnf("IPv4 %q does not fit in zone %q", ipv4, p.config.Ipv4ReverseZoneName)
				continue
			}
			ptr, err := rdns.ReverseAddr(ipv4)
			if err != nil {
				addrLogger.Errorw("could not determine PTR record", "err", err)
				return err
			}
			addrLogger = addrLogger.With("ptr", ptr)
			if existingHostname, seen := ptrHostnames[ptr]; seen {
				addrLogger.Warnf("already registered PTR %q with hostname %q", ptr, existingHostname)
				continue
			}
			ptrHostnames[ptr] = fullHostname
			changes = append(changes, p.dnsChange(
				ptr,
				[]string{fullHostname},
				"PTR",
				endpoint.RecordTTL,
			))
			addrLogger.Infof("adding IPv4 PTR record %q for hostname %q", ptr, hostname)
		}
	}
	if len(changes) == 0 {
		p.logger.Info("No reverse IPv4 changes.")
		return nil
	}
	_, err := p.client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.config.Ipv4ReverseZoneID),
		ChangeBatch:  &types.ChangeBatch{Changes: changes},
	})

	return err
}

func (p *route53Provider) updateIPv6Reverse(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	if p.config.Ipv6ReverseZoneID == "" {
		p.logger.Warn("IPv6 reverse lookup zone disabled")
		return nil
	}
	if p.config.Ipv6ReverseZoneName == "" {
		ipv6ReverseZoneName, err := getRoute53ZoneName(ctx, p.client, p.config.Ipv6ReverseZoneID)
		if err != nil {
			p.logger.Sugar().Errorw("could not get Route53 zone name", "err", err)
			return err
		}
		p.config.Ipv6ReverseZoneName = ipv6ReverseZoneName
	}

	if p.config.CleanIPv6ReverseZone {
		p.logger.Info("cleanup: cleaning IPv6 reverse lookup zone")
		p.cleanupZone(ctx, p.config.Ipv6ReverseZoneID, []types.RRType{types.RRTypePtr}, func(name string) bool {
			for _, endpoint := range endpoints {
				for _, ipv6 := range endpoint.IPv6s {
					ptr, err := rdns.ReverseAddr(ipv6)
					if err != nil {
						p.logger.Sugar().Errorw("cleanup: could not determine PTR record", "err", err)
						break
					}
					if ptr == name {
						return true
					}
				}
			}
			return false
		})
	}

	ptrHostnames := map[string]string{}
	changes := make([]types.Change, 0)
	for _, endpoint := range endpoints {
		hostname := endpoint.Hostname
		if hostname == "" {
			if len(endpoint.IPv4s) == 0 {
				p.logger.Warn("Cannot generate hostname for endpoint due to missing IPv4 address.")
				return nil
			}
			hostname = "ip-" + strings.ReplaceAll(endpoint.IPv4s[0], ".", "-")
			p.logger.Sugar().Infof("No hostname defined for endpoint, using generated hostname of %s", hostname)
		}
		fullHostname := utils.DNSSafeName(hostname) + p.config.RecordSuffix
		for _, ipv6 := range endpoint.IPv6s {
			addrLogger := p.logger.Sugar().With(
				"addr", ipv6,
				"zone", p.config.Ipv6ReverseZoneName,
				"ttl", endpoint.RecordTTL,
				"full_hostname", fullHostname,
				"hostname", hostname,
			)
			fits, err := rdns.FitsInReverseZone(ipv6, p.config.Ipv6ReverseZoneName)
			if err != nil {
				addrLogger.Errorw(
					"could not determine if address fits in reverse zone",
					"err", err,
				)
				return err
			}
			if !fits {
				addrLogger.Warnf("IPv6 %q does not fit in zone %q", ipv6, p.config.Ipv6ReverseZoneName)
				continue
			}
			ptr, err := rdns.ReverseAddr(ipv6)
			if err != nil {
				addrLogger.Errorw("could not determine PTR record", "err", err)
				return err
			}
			addrLogger = addrLogger.With("ptr", ptr)
			if existingHostname, seen := ptrHostnames[ptr]; seen {
				addrLogger.Warnf("already registered PTR %q with hostname %q", ptr, existingHostname)
				continue
			}
			ptrHostnames[ptr] = fullHostname
			changes = append(changes, p.dnsChange(
				ptr,
				[]string{fullHostname},
				"PTR",
				endpoint.RecordTTL,
			))
			addrLogger.Infof("adding IPv6 PTR record %q for hostname %q", ptr, hostname)
		}
	}
	if len(changes) == 0 {
		p.logger.Info("No reverse IPv6 changes.")
		return nil
	}
	_, err := p.client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.config.Ipv6ReverseZoneID),
		ChangeBatch:  &types.ChangeBatch{Changes: changes},
	})

	return err
}

func (p *route53Provider) dnsChange(name string, answers []string, recordType string, ttl int64) types.Change {
	resourceRecords := make([]types.ResourceRecord, 0)
	for _, address := range answers {
		resourceRecords = append(resourceRecords, types.ResourceRecord{Value: aws.String(address)})
	}
	return types.Change{
		Action: types.ChangeActionUpsert,
		ResourceRecordSet: &types.ResourceRecordSet{
			Name:            aws.String(name),
			Type:            types.RRType(recordType),
			TTL:             aws.Int64(ttl),
			ResourceRecords: resourceRecords,
		},
	}
}

func (p *route53Provider) cleanupZone(ctx context.Context, zoneID string, cleanTypes []types.RRType, foundFunc func(string) bool) error {
	isTruncated := true
	nextRecordIdentifier := ""
	var nextRecordType types.RRType

	for isTruncated {
		cleanChanges := make([]types.Change, 0)
		input := &route53.ListResourceRecordSetsInput{
			HostedZoneId: aws.String(zoneID),
		}
		if nextRecordIdentifier != "" {
			input.StartRecordIdentifier = aws.String(nextRecordIdentifier)
			input.StartRecordType = nextRecordType
		}
		listOutput, err := p.client.ListResourceRecordSets(ctx, input)
		if err != nil {
			p.logger.Sugar().Errorw("cleanup: failed to list resource records for hosted zone", "zone", zoneID, "err", err)
			return err
		}

		isTruncated = listOutput.IsTruncated
		if listOutput.NextRecordIdentifier != nil {
			nextRecordIdentifier = *listOutput.NextRecordIdentifier
			nextRecordType = listOutput.NextRecordType
		}

		for i, rr := range listOutput.ResourceRecordSets {
			if !slices.Contains(cleanTypes, rr.Type) {
				continue
			}
			if !foundFunc(*rr.Name) {
				p.logger.Sugar().Infof("cleanup: removing record %s", *rr.Name)
				cleanChanges = append(cleanChanges, types.Change{
					Action:            types.ChangeActionDelete,
					ResourceRecordSet: &listOutput.ResourceRecordSets[i],
				})
			}
		}

		if len(cleanChanges) == 0 {
			p.logger.Info("cleanup: no changes needed")
			return nil
		}

		_, err = p.client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: aws.String(zoneID),
			ChangeBatch:  &types.ChangeBatch{Changes: cleanChanges},
		})
		if err != nil {
			p.logger.Sugar().Errorw("cleanup: failed to delete resource records for hosted zone", "err", err)
			return err
		}
	}
	return nil
}
