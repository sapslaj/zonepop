package utils

import (
	"regexp"
	"strings"
)

// DNSSafeName converts an externally-supplied hostname into one that is safe to
// be used in a DNS record.
func DNSSafeName(name string) string {
	re := regexp.MustCompile(`\s+`)
	return strings.Trim(re.ReplaceAllString(name, "-"), ".")
}
