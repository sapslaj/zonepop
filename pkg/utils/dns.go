package utils

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// DNSSafeName converts an externally-supplied hostname into one that is safe to
// be used in a DNS record.
func DNSSafeName(name string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(name, "-")
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
	ip := net.ParseIP(addr)
	if ip == nil {
		return false, fmt.Errorf("failed to parse address %q", addr)
	}
	if ip.To4() != nil {
		if !strings.Contains(zone, ".in-addr.arpa.") {
			return false, fmt.Errorf("zone name %q is not a valid IPv4 reverse lookup zone for IP address %q", zone, addr)
		}
		ipOctets := strings.Split(addr, ".")
		zoneOctets := Reversed(strings.Split(strings.TrimSuffix(zone, ".in-addr.arpa."), "."))
		for i, zoneOctet := range zoneOctets {
			if zoneOctet != ipOctets[i] {
				return false, nil
			}
		}
		return true, nil
	}
	if !strings.Contains(zone, ".ip6.arpa.") {
		return false, fmt.Errorf("zone name %q is not a valid IPv6 reverse lookup zone for IP address %q", zone, addr)
	}
	reverseAddr, err := ReverseAddr(addr)
	if err != nil {
		return false, fmt.Errorf("failed to parse address %q: %w", addr, err)
	}
	ptrNibbles := Reversed(strings.Split(reverseAddr, "."))
	zoneNibbles := Reversed(strings.Split(zone, "."))
	for i, zoneNibble := range zoneNibbles {
		if zoneNibble != ptrNibbles[i] {
			return false, nil
		}
	}
	return true, nil
}
