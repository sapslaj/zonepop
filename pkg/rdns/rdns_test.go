package rdns

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineAddressKind(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    string
		expected AddressKind
		errMsg   string
	}{
		"valid IPv4": {
			input:    "192.0.2.69",
			expected: AddressKindIPv4,
		},
		"invalid IPv4": {
			input:  "256.999.24.2",
			errMsg: "failed to parse address",
		},
		"valid IPv6": {
			input:    "2001:db8::1",
			expected: AddressKindIPv6,
		},
		"invalid IPv6": {
			input:  "ffff::ffff::ffff",
			errMsg: "failed to parse address",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := DetermineAddressKind(tc.input)
			if tc.errMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			}
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestIsReverseDNSZone(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		for _, zone := range []string{
			"192.in-addr.arpa.",
			"0.0.2.ip6.arpa.",
		} {
			assert.True(t, IsReverseDNSZone(zone))
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		for _, zone := range []string{
			"192.0.2.69",
			"2001:db8::1",
		} {
			assert.False(t, IsReverseDNSZone(zone))
		}
	})
}

func TestReverseAddr(t *testing.T) {
	t.Parallel()

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
		tc := tc
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

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
		})
	}
}

func TestFitsInReverseZone(t *testing.T) {
	t.Parallel()

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
		"fails parsing address": {
			addr:   "invalid",
			errMsg: "failed to parse address",
		},
	}
	for desc, tc := range tests {
		tc := tc
		t.Run(desc, func(t *testing.T) {
			t.Parallel()

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
		})
	}
}
