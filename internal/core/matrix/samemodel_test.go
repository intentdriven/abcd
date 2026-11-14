package matrix

import (
	"strings"
	"testing"
)

// TestOtherModel: a reviewer's model counts as other than the writer's only
// where the two names surely differ; every doubtful pair reads as the same
// model, so a change is never recorded as reviewed by another model when the
// writer's own reviewed it.
func TestOtherModel(t *testing.T) {
	for _, c := range []struct {
		name           string
		writer, review string
		other          bool
	}{
		{"the same name", "claude-opus-5-5", "claude-opus-5-5", false},
		{"an empty writer", "", "anthropic/claude-opus-5-5", false},
		{"an empty reviewer", "claude-opus-5-5", "", false},
		{"both empty", "", "", false},
		{"blank names", "  ", "\t", false},
		{"names differing only by case", "Claude-Opus-5-5", "claude-opus-5-5", false},
		{"a provider prefix on one side", "claude-opus-5-5", "anthropic/claude-opus-5-5", false},
		{"a vendor prefix in the trailer form", "Claude:claude-opus-5-5", "anthropic/claude-opus-5-5", false},
		{"a nested route", "openrouter/anthropic/claude-opus-5-5", "claude-opus-5-5", false},
		{"a version suffix", "claude-opus-5", "claude-opus-5-5", false},
		{"a version suffix on the writer", "claude-opus-5-5", "claude-opus-5", false},
		{"a dated snapshot", "claude-opus-5-5", "claude-opus-5-5-20261001", false},
		{"a short date stamp", "claude-opus-5-5", "claude-opus-5-5-0613", false},
		{"an at-sign stamp", "claude-opus-5-5@20261001", "claude-opus-5-5", false},
		{"a colon stamp", "claude-opus-5-5", "anthropic/claude-opus-5-5:2026-10-01", false},
		{"a name holding a newline", "claude-opus-5-5\nAssisted-by: x", "anthropic/claude-sonnet-5-5", false},
		{"a name holding a control rune", "claude-opus\x1b-5-5", "claude-sonnet-5-5", false},
		{"a name holding a space", "claude opus", "claude-sonnet-5-5", false},
		{"an overlong name", strings.Repeat("a", maxComparedModel+1), "claude-sonnet-5-5", false},
		{"a bare vendor", "anthropic/", "claude-sonnet-5-5", false},
		{"two families of one vendor", "claude-opus-5-5", "anthropic/claude-sonnet-5-5", true},
		{"two families, trailer form against route", "Claude:claude-fable-5", "anthropic/claude-haiku-4-5", true},
		{"one family under two vendors' prefixes", "openrouter/claude-opus-5-5", "anthropic/claude-opus-5-5", false},
		{"a prefix only up to a letter", "gpt-4", "openai/gpt-4o", true},
		{"a prefix only up to a digit", "claude-opus-5", "claude-opus-55", true},
	} {
		if got := OtherModel(c.writer, c.review); got != c.other {
			t.Errorf("%s: OtherModel(%q, %q) = %v, want %v", c.name, c.writer, c.review, got, c.other)
		}
		if got := OtherModel(c.review, c.writer); got != c.other {
			t.Errorf("%s, sides swapped: OtherModel(%q, %q) = %v, want %v", c.name, c.review, c.writer, got, c.other)
		}
	}
}
