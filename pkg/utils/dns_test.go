package utils

import (
	"strings"
	"testing"
)

func TestDNSSafeName(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"name with space": {
			input:    "host name",
			expected: "host-name",
		},
	}

	for desc, tc := range tests {
		got := DNSSafeName(tc.input)
		if tc.expected != got {
			t.Errorf("%s: expected %q, got %q", desc, tc.expected, got)
		}
	}
}

func TestReverseAddr(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
		errMsg   string
	}{
		"valid IPv4": {
			input:    "192.0.2.69",
			expected: "69.2.0.192.in-addr.arpa.",
		},
		"invalid IPv4": {
			input:  "256.999.24.2",
			errMsg: "failed to parse address",
		},
		"valid IPv6": {
			input:    "2001:db8::1",
			expected: "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.",
		},
		"invalid IPv6": {
			input:  "ffff::ffff::ffff",
			errMsg: "failed to parse address",
		},
	}
	for desc, tc := range tests {
		got, err := ReverseAddr(tc.input)
		if tc.errMsg == "" && err != nil {
			t.Errorf("%s: expected no error but got error %v", desc, err)
		}
		if tc.errMsg != "" && err == nil {
			t.Errorf("%s: expected error %q but got nil", desc, tc.errMsg)
		}
		if tc.errMsg != "" && err != nil && !strings.Contains(err.Error(), tc.errMsg) {
			t.Errorf("%s: expected error %q but got %v", desc, tc.errMsg, err)
		}
		if tc.expected != got {
			t.Errorf("%s: expected %q, got %q", desc, tc.expected, got)
		}
	}
}

func TestFitsInReverseZone(t *testing.T) {
	tests := map[string]struct {
		addr   string
		zone   string
		fits   bool
		errMsg string
	}{
		"fitting valid IPv4": {
			addr: "192.0.2.69",
			zone: "0.192.in-addr.arpa.",
			fits: true,
		},
		"fitting valid IPv6": {
			addr: "2001:db8::1",
			zone: "8.b.d.0.1.0.0.2.ip6.arpa.",
			fits: true,
		},
		"unfitting valid IPv4": {
			addr: "203.0.113.69",
			zone: "0.192.in-addr.arpa.",
			fits: false,
		},
		"unfitting valid IPv6": {
			addr: "fe80::1",
			zone: "8.b.d.0.1.0.0.2.ip6.arpa.",
			fits: false,
		},
		"invalid IPv4": {
			addr:   "256.999.24.2",
			zone:   "0.192.in-addr.arpa.",
			fits:   false,
			errMsg: "failed to parse address",
		},
		"invalid IPv6": {
			addr:   "ffff::ffff::ffff",
			zone:   "8.b.d.0.1.0.0.2.ip6.arpa.",
			fits:   false,
			errMsg: "failed to parse address",
		},
		"invalid zone for IPv4": {
			addr:   "192.0.2.69",
			zone:   "8.b.d.0.1.0.0.2.ip6.arpa.",
			fits:   false,
			errMsg: "not a valid IPv4 reverse lookup zone",
		},
		"invalid zone for IPv6": {
			addr:   "2001:db8::1",
			zone:   "0.192.in-addr.arpa.",
			fits:   false,
			errMsg: "not a valid IPv6 reverse lookup zone",
		},
	}
	for desc, tc := range tests {
		got, err := FitsInReverseZone(tc.addr, tc.zone)
		if tc.errMsg == "" && err != nil {
			t.Errorf("%s: expected no error but got error %v", desc, err)
		}
		if tc.errMsg != "" && err == nil {
			t.Errorf("%s: expected error %q but got nil", desc, tc.errMsg)
		}
		if tc.errMsg != "" && err != nil && !strings.Contains(err.Error(), tc.errMsg) {
			t.Errorf("%s: expected error %q but got %v", desc, tc.errMsg, err)
		}
		if tc.fits != got {
			t.Errorf("%s: expected %v, got %v", desc, tc.fits, got)
		}
	}
}
