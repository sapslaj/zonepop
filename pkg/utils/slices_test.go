package utils

import (
	"reflect"
	"testing"
)

func TestReversed(t *testing.T) {
	tests := map[string]struct {
		input    []string
		expected []string
	}{
		"List of letters": {
			input:    []string{"a", "c", "b"},
			expected: []string{"b", "c", "a"},
		},
		"List of Names": {
			input:    []string{"Ganyu", "Keqing", "Hu Tao"},
			expected: []string{"Hu Tao", "Keqing", "Ganyu"},
		},
	}

	for desc, tc := range tests {
		got := Reversed(tc.input)
		if !reflect.DeepEqual(tc.expected, got) {
			t.Errorf("%s: expected %q, got %q", desc, tc.expected, got)
		}
	}
}

func TestFilter(t *testing.T) {
	tests := map[string]struct {
		input    []string
		expected []string
		f        func(string) bool
	}{
		"filter out a's": {
			input:    []string{"a", "b", "c", "a", "b", "c"},
			expected: []string{"b", "c", "b", "c"},
			f:        func(s string) bool { return s != "a" },
		},
	}

	for desc, tc := range tests {
		got := Filter(tc.f, tc.input)
		if !reflect.DeepEqual(tc.expected, got) {
			t.Errorf("%s: expected %q, got %q", desc, tc.expected, got)
		}
	}
}
