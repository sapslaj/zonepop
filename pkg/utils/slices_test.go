package utils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
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
		t.Run(desc, func(t *testing.T) {
			got := Reversed(tc.input)
			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("%s: mismatch:\n%s", desc, diff)
			}
		})
	}
}

func TestFilterErr(t *testing.T) {
	tests := map[string]struct {
		input    []string
		expected []string
		err      error
		f        func(string) (bool, error)
	}{
		"filter with no error": {
			input:    []string{"a", "b", "c", "a", "b", "c"},
			expected: []string{"b", "c", "b", "c"},
			f:        func(s string) (bool, error) { return s != "a", nil },
		},
		"filter with error": {
			input:    []string{"a", "c", "b"},
			expected: []string{},
			err:      errors.New("dummy error"),
		},
	}

	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			if tc.f == nil {
				tc.f = func(s string) (bool, error) { return false, tc.err }
			}
			got, err := FilterErr(tc.f, tc.input)
			if !errors.Is(err, tc.err) {
				t.Fatalf("%s: expected error %v, got %v", desc, tc.err, err)
			}
			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("%s: mismatch:\n%s", desc, diff)
			}
		})
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
		t.Run(desc, func(t *testing.T) {
			got := Filter(tc.f, tc.input)
			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("%s: mismatch:\n%s", desc, diff)
			}
		})
	}
}

func TestMapErr(t *testing.T) {
	tests := map[string]struct {
		input    []string
		expected []string
		err      error
		f        func(string) (string, error)
	}{
		"func with no error": {
			input:    []string{"69", "420"},
			expected: []string{"69", "420"},
			f:        func(s string) (string, error) { return s, nil },
		},
		"func with error": {
			input:    []string{"69", "420"},
			expected: []string{},
			err:      errors.New("dummy error"),
		},
	}

	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			if tc.f == nil {
				tc.f = func(s string) (string, error) { return s, tc.err }
			}
			got, err := MapErr(tc.f, tc.input)
			if !errors.Is(err, tc.err) {
				t.Fatalf("%s: expected error %v, got %v", desc, tc.err, err)
			}
			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("%s: mismatch:\n%s", desc, diff)
			}
		})
	}
}

func TestMap(t *testing.T) {
	tests := map[string]struct {
		input    []int
		expected []string
		f        func(int) string
	}{
		"run ints through Sprint": {
			input:    []int{69, 420},
			expected: []string{"69", "420"},
			f:        func(i int) string { return fmt.Sprint(i) },
		},
	}

	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			got := Map(tc.f, tc.input)
			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("%s: mismatch:\n%s", desc, diff)
			}
		})
	}
}
