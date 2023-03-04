package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"go.uber.org/zap"

	"github.com/sapslaj/zonepop/endpoint"
	"github.com/sapslaj/zonepop/pkg/log"
	"github.com/sapslaj/zonepop/pkg/utils"
	"github.com/sapslaj/zonepop/provider"
)

type route53Provider struct {
	recordSuffix        string
	forwardZoneID       string
	forwardZoneName     string
	ipv4ReverseZoneID   string
	ipv4ReverseZoneName string
	ipv6ReverseZoneID   string
	ipv6ReverseZoneName string
	client              *route53.Client
	logger              *zap.Logger
}

func getRoute53ZoneName(ctx context.Context, client *route53.Client, zoneID string) (string, error) {
	ghzout, err := client.GetHostedZone(ctx, &route53.GetHostedZoneInput{
		Id: aws.String(zoneID),
	})
	if err != nil {
		return "", fmt.Errorf("could not get hosted zone information for %s: %w", zoneID, err)
	}
	return aws.ToString(ghzout.HostedZone.Name), nil
}

func NewRoute53Provider(recordSuffix, forwardZoneID, ipv4ReverseZoneID, ipv6ReverseZoneID string) (provider.Provider, error) {
	client, err := defaultR53Client()
	if err != nil {
		return nil, fmt.Errorf("could not get default Route53 client: %w", err)
	}
	p := &route53Provider{
		recordSuffix:      recordSuffix,
		forwardZoneID:     forwardZoneID,
		ipv4ReverseZoneID: ipv4ReverseZoneID,
		ipv6ReverseZoneID: ipv6ReverseZoneID,
		client:            client,
		logger:            log.MustNewLogger().Named("aws_route53_provider"),
	}
	return p, nil
}

func (p *route53Provider) UpdateEndpoints(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	err := p.updateForward(ctx, endpoints)
	if err != nil {
		p.logger.Sugar().Errorw("failed to update forward lookup zone", "err", err)
		return err
	}
	err = p.updateIPv4Reverse(ctx, endpoints)
	if err != nil {
		p.logger.Sugar().Errorw("failed to update IPv4 reverse lookup zone", "err", err)
		return err
	}
	err = p.updateIPv6Reverse(ctx, endpoints)
	if err != nil {
		p.logger.Sugar().Errorw("failed to update IPv6 reverse lookup zone", "err", err)
		return err
	}
	return nil
}

func (p *route53Provider) updateForward(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	if p.forwardZoneID == "" {
		p.logger.Warn("Forward lookup zone disabled")
		return nil
	}
	if p.forwardZoneID != "" && p.forwardZoneName == "" {
		forwardZoneName, err := getRoute53ZoneName(ctx, p.client, p.forwardZoneID)
		if err != nil {
			p.logger.Sugar().Errorw("could not get Route53 zone name", "err", err)
			return err
		}
		p.forwardZoneName = forwardZoneName
	}

	if p.recordSuffix == "" {
		p.recordSuffix = "." + p.forwardZoneName
	}

	hostnameEndpoints := make(map[string][]*endpoint.Endpoint)
	for _, endpoint := range endpoints {
		if endpoint.Hostname == "" {
			continue
		}
		hostnameEndpoints[endpoint.Hostname] = append(hostnameEndpoints[endpoint.Hostname], endpoint)
	}
	changes := make([]types.Change, 0)
	for hostname, endpoints := range hostnameEndpoints {
		ipv4 := make([]string, 0)
		ipv6 := make([]string, 0)
		if len(endpoints) == 0 {
			p.logger.Sugar().Warnf("No endpoints for hostname %q", hostname)
			continue
		}
		for _, endpoint := range endpoints {
			ipv4 = append(ipv4, endpoint.IPv4s...)
			ipv6 = append(ipv6, endpoint.IPv6s...)
		}
		if len(ipv4) > 0 {
			changes = append(changes, p.dnsChange(
				utils.DNSSafeName(hostname)+p.recordSuffix,
				ipv4,
				"A",
				endpoints[0].RecordTTL,
			))
		}
		if len(ipv6) > 0 {
			changes = append(changes, p.dnsChange(
				utils.DNSSafeName(hostname)+p.recordSuffix,
				ipv6,
				"AAAA",
				endpoints[0].RecordTTL,
			))
		}
	}
	_, err := p.client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.forwardZoneID),
		ChangeBatch:  &types.ChangeBatch{Changes: changes},
	})

	return err
}

func (p *route53Provider) updateIPv4Reverse(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	if p.ipv4ReverseZoneID == "" {
		p.logger.Warn("IPv4 reverse lookup zone disabled")
		return nil
	}
	if p.ipv4ReverseZoneName == "" {
		ipv4ReverseZoneName, err := getRoute53ZoneName(ctx, p.client, p.ipv4ReverseZoneID)
		if err != nil {
			p.logger.Sugar().Errorw("could not get Route53 zone name", "err", err)
			return err
		}
		p.ipv4ReverseZoneName = ipv4ReverseZoneName
	}

	changes := make([]types.Change, 0)
	for _, endpoint := range endpoints {
		hostname := endpoint.Hostname
		for _, ipv4 := range endpoint.IPv4s {
			addrLogger := p.logger.Sugar().With(
				"addr", ipv4,
				"zone", p.ipv4ReverseZoneName,
			)
			if hostname == "" {
				hostname = "ip-" + strings.ReplaceAll(ipv4, ".", "-")
				addrLogger.Infof("No hostname defined for endpoint, using generated hostname of %s", hostname)
			}
			addrLogger = addrLogger.With("hostname", hostname)
			fits, err := utils.FitsInReverseZone(ipv4, p.ipv4ReverseZoneName)
			if err != nil {
				addrLogger.Errorw(
					"could not determine if address fits in reverse zone",
					"err", err,
				)
				return err
			}
			if !fits {
				addrLogger.Warnf("IPv4 %q does not fit in zone %q", ipv4, p.ipv4ReverseZoneName)
				continue
			}
			ptr, err := utils.ReverseAddr(ipv4)
			if err != nil {
				addrLogger.Errorw("could not determine PTR record", "err", err)
				return err
			}
			changes = append(changes, p.dnsChange(
				ptr,
				[]string{utils.DNSSafeName(hostname) + p.recordSuffix},
				"PTR",
				endpoint.RecordTTL,
			))
		}
	}
	_, err := p.client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.ipv4ReverseZoneID),
		ChangeBatch:  &types.ChangeBatch{Changes: changes},
	})

	return err
}

func (p *route53Provider) updateIPv6Reverse(ctx context.Context, endpoints []*endpoint.Endpoint) error {
	if p.ipv6ReverseZoneID == "" {
		p.logger.Warn("IPv6 reverse lookup zone disabled")
		return nil
	}
	if p.ipv6ReverseZoneName == "" {
		ipv6ReverseZoneName, err := getRoute53ZoneName(ctx, p.client, p.ipv6ReverseZoneID)
		if err != nil {
			p.logger.Sugar().Errorw("could not get Route53 zone name", "err", err)
			return err
		}
		p.ipv6ReverseZoneName = ipv6ReverseZoneName
	}

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
		for _, ipv6 := range endpoint.IPv6s {
			addrLogger := p.logger.Sugar().With(
				"addr", ipv6,
				"zone", p.ipv6ReverseZoneName,
			)
			fits, err := utils.FitsInReverseZone(ipv6, p.ipv6ReverseZoneName)
			if err != nil {
				addrLogger.Errorw(
					"could not determine if address fits in reverse zone",
					"err", err,
				)
				return err
			}
			if !fits {
				addrLogger.Warnf("IPv6 %q does not fit in zone %q", ipv6, p.ipv6ReverseZoneName)
				continue
			}
			ptr, err := utils.ReverseAddr(ipv6)
			if err != nil {
				addrLogger.Errorw("could not determine PTR record", "err", err)
				return err
			}
			changes = append(changes, p.dnsChange(
				ptr,
				[]string{utils.DNSSafeName(hostname) + p.recordSuffix},
				"PTR",
				endpoint.RecordTTL,
			))
		}
	}
	_, err := p.client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(p.ipv6ReverseZoneID),
		ChangeBatch:  &types.ChangeBatch{Changes: changes},
	})

	return err
}

func (u *route53Provider) dnsChange(name string, answers []string, recordType string, ttl int64) types.Change {
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
