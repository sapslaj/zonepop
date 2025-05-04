package utils

import (
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
		t.Run(desc, func(t *testing.T) {
			got := DNSSafeName(tc.input)
			if tc.expected != got {
				t.Errorf("%s: expected %q, got %q", desc, tc.expected, got)
			}
		})
	}
}
