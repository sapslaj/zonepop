package rdns

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/sapslaj/zonepop/pkg/utils"
)

type AddressKind string

const (
	AddressKindIPv4 AddressKind = "ipv4"
	AddressKindIPv6 AddressKind = "ipv6"
)

func DetermineAddressKind(addr string) (AddressKind, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return "", fmt.Errorf("failed to parse address %q", addr)
	}
	if ip.To4() != nil {
		return AddressKindIPv4, nil
	}
	return AddressKindIPv6, nil
}

func IsReverseDNSZone(zone string) bool {
	if strings.HasSuffix(zone, ".in-addr.arpa.") {
		return true
	}
	if strings.HasSuffix(zone, ".ip6.arpa.") {
		return true
	}
	return false
}

var ErrInvalidZone = errors.New("invalid zone")

func DetermineReverseZoneKind(zone string) (AddressKind, error) {
	if !strings.HasSuffix(zone, ".") {
		zone += "."
	}
	if strings.HasSuffix(zone, ".in-addr.arpa.") {
		return AddressKindIPv4, nil
	}
	if strings.HasSuffix(zone, ".ip6.arpa.") {
		return AddressKindIPv6, nil
	}
	return "", fmt.Errorf("%w: %q is not a valid rDNS zone name", ErrInvalidZone, zone)
}

const hexDigit = "0123456789abcdef"

// ReverseAddr calculates the rDNS record of an IPv4 or IPv6 address.
func ReverseAddr(addr string) (string, error) {
	// Adapted from src/net/dnsclient.go
	ip := net.ParseIP(addr)
	if ip == nil {
		return "", fmt.Errorf("failed to parse address %q", addr)
	}
	if ip.To4() != nil {
		r := strconv.Itoa(int(ip[15])) + "." + strconv.Itoa(int(ip[14])) + "." + strconv.Itoa(int(ip[13])) + "." + strconv.Itoa(int(ip[12])) + ".in-addr.arpa."
		return r, nil
	}
	// Must be IPv6
	buf := make([]byte, 0, len(ip)*4+len("ip6.arpa."))
	// Add it, in reverse, to the buffer
	for i := len(ip) - 1; i >= 0; i-- {
		v := ip[i]
		buf = append(buf,
			hexDigit[v&0xF],
			'.',
			hexDigit[v>>4],
			'.',
		)
	}
	// Append "ip6.arpa." and return (buf already has the final .)
	buf = append(buf, "ip6.arpa."...)
	return string(buf), nil
}

// FitsInReverseZone calculates whether an IPv4 or IPv6 address fits in a
// specified rDNS zone.
func FitsInReverseZone(addr string, zone string) (bool, error) {
	addrKind, err := DetermineAddressKind(addr)
	if err != nil {
		return false, err
	}
	zoneKind, err := DetermineReverseZoneKind(zone)
	if err != nil {
		return false, err
	}

	// mismatched ipv4 / ipv6
	if addrKind != zoneKind {
		return false, nil
	}

	if addrKind == AddressKindIPv4 {
		ipOctets := strings.Split(addr, ".")
		zoneOctets := utils.Reversed(strings.Split(strings.TrimSuffix(zone, ".in-addr.arpa."), "."))
		for i, zoneOctet := range zoneOctets {
			if zoneOctet != ipOctets[i] {
				return false, nil
			}
		}
		return true, nil
	}

	reverseAddr, err := ReverseAddr(addr)
	if err != nil {
		return false, fmt.Errorf("failed to parse address %q: %w", addr, err)
	}
	ptrNibbles := utils.Reversed(strings.Split(reverseAddr, "."))
	zoneNibbles := utils.Reversed(strings.Split(zone, "."))
	for i, zoneNibble := range zoneNibbles {
		if zoneNibble != ptrNibbles[i] {
			return false, nil
		}
	}
	return true, nil
}
