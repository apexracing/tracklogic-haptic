package simagic

import "strings"

// canonicalise upper-cases the name and trims whitespace; it is
// used to make VID/PID matching case-insensitive.
func canonicalise(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

// containsAny reports whether s contains any of the given
// case-sensitive substrings. s should already be canonicalised.
func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
