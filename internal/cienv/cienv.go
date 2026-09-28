// Package cienv holds the one canonical answer to "is this a CI runner?". Two
// verbs ask it — the load check (core/implement), which skips on a runner, and
// the tool installer (core/tools), which never installs on one — and a detector
// spelled twice is one the two disagree about. It is a leaf below both, because
// core/implement reaches core/tools through the verbs it composes, so neither
// can import the other.
package cienv

import "github.com/intentdriven/abcd/internal/termsafe"

// Runner reports whether the environment is a CI runner, and names the
// variable that says so: GITHUB_ACTIONS=true, or a CI value other than empty,
// "false" or "0". The named value is sanitised and capped, since it is the
// environment's text and a caller prints it.
func Runner(getenv func(string) string) (string, bool) {
	if getenv("GITHUB_ACTIONS") == "true" {
		return "GITHUB_ACTIONS=true", true
	}
	switch v := getenv("CI"); v {
	case "", "false", "0":
		return "", false
	default:
		shown := termsafe.Sanitize(v)
		if len(shown) > 32 {
			shown = shown[:32]
		}
		return "CI=" + shown, true
	}
}
