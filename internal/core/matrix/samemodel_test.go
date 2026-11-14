package matrix

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

// TestOtherModel: a reviewer's model counts as other than the writer's only
// where that is shown, both names known and neither one's tokens held in the
// other's; every other pair reads as the same model, so a change is never
// recorded as reviewed by another model when the writer's own reviewed it.
// Each pair is asked both ways round.
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
		{"a dated snapshot", "claude-opus-5-5", "claude-opus-5-5-20261001", false},
		{"a short date stamp", "claude-opus-5-5", "claude-opus-5-5-0613", false},
		{"an at-sign stamp", "claude-opus-5-5@20261001", "claude-opus-5-5", false},
		{"a colon stamp", "claude-opus-5-5", "anthropic/claude-opus-5-5:2026-10-01", false},
		{"an alias against the full name", "opus", "claude-opus-5-5", false},
		{"a bracketed runtime suffix", "claude-opus-5-5[1m]", "anthropic/claude-opus-5-5", false},
		{"a bracketed suffix against a cloud id", "claude-opus-5-5[1m]", "us.anthropic.claude-opus-5-5-v1:0", false},
		{"a cloud id with its region and version", "bedrock/us.anthropic.claude-opus-5-5-v1:0", "claude-opus-5-5", false},
		{"a dotted vendor", "anthropic.claude-opus-5-5", "claude-opus-5-5", false},
		{"a vendor with a dash before a colon", "google-vertex:claude-opus-5-5", "claude-opus-5-5", false},
		{"a dotted version", "claude-opus-5.5", "claude-opus-5-5", false},
		{"an underscored name", "claude_opus_5_5", "claude-opus-5.5", false},
		{"a plus suffix", "claude-opus-5-5+thinking", "claude-opus-5-5", false},
		{"a latest alias", "claude-opus-latest", "claude-opus-5-5", false},
		{"a name holding a newline", "claude-opus-5-5\nAssisted-by: x", "anthropic/claude-sonnet-5-5", false},
		{"a name holding a control rune", "claude-opus\x1b-5-5", "claude-sonnet-5-5", false},
		{"a non-ASCII letter that looks like an ASCII one", "claude-оpus-5-5", "claude-sonnet-5-5", false},
		{"a zero-width rune", "claude-opus​-5-5", "claude-sonnet-5-5", false},
		{"an overlong name", strings.Repeat("a", maxComparedModel+1), "claude-sonnet-5-5", false},
		{"a bare vendor", "anthropic/", "claude-sonnet-5-5", false},
		{"a vendor word alone", "Claude", "claude-sonnet-5-5", false},
		{"a stamp alone", "20261001", "claude-sonnet-5-5", false},
		{"one family under two vendors' prefixes", "openrouter/claude-opus-5-5", "anthropic/claude-opus-5-5", false},
		{"two families of one vendor", "claude-opus-5-5", "anthropic/claude-sonnet-5-5", true},
		{"two families, trailer form against route", "Claude:claude-fable-5", "anthropic/claude-haiku-4-5", true},
		{"two families in cloud ids", "us.anthropic.claude-opus-5-5-v1:0", "eu.anthropic.claude-sonnet-5-5-v1:0", true},
		{"a name split on a space is read", "claude opus", "claude-sonnet-5-5", true},
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

// modelish is a generated string drawn mostly from the characters model names
// are made of, so generated pairs often share tokens, with now and then a rune
// no name may carry.
type modelish string

func (modelish) Generate(r *rand.Rand, size int) reflect.Value {
	const alphabet = "abcopusnetfl0123456789-._:/@+[] vV"
	var b strings.Builder
	for n := r.Intn(size + 1); n > 0; n-- {
		switch r.Intn(40) {
		case 0:
			b.WriteRune('о')
		case 1:
			b.WriteByte('\n')
		default:
			b.WriteByte(alphabet[r.Intn(len(alphabet))])
		}
	}
	return reflect.ValueOf(modelish(b.String()))
}

// TestOtherModelIsSymmetricAndNeverOtherThanItself: for any two strings,
// generated from model-like characters or arbitrary, the answer does not hang
// on which side is the writer's, and no name is another model than itself.
func TestOtherModelIsSymmetricAndNeverOtherThanItself(t *testing.T) {
	cfg := &quick.Config{MaxCount: 20000}
	if err := quick.Check(func(a, b modelish) bool {
		return OtherModel(string(a), string(b)) == OtherModel(string(b), string(a)) && !OtherModel(string(a), string(a))
	}, cfg); err != nil {
		t.Error(err)
	}
	if err := quick.Check(func(a, b string) bool {
		return OtherModel(a, b) == OtherModel(b, a) && !OtherModel(a, a)
	}, cfg); err != nil {
		t.Error(err)
	}
}
