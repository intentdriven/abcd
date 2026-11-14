package matrix

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

// TestOtherModel: a reviewer's model counts as other than the writer's only
// where that is shown, both names naming one listed family each and the
// families or their versions differing; every other pair reads as the same
// model, so a change is never recorded as reviewed by another model when the
// writer's own reviewed it. Each pair is asked both ways round.
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
		{"a letter after the version is a tag", "gpt-4", "openai/gpt-4o", false},
		{"a prefix of the version's digits", "claude-opus-5", "claude-opus-55", false},
		{"two mode suffixes after a colon", "openrouter/anthropic/claude-opus-5-5:thinking", "openrouter/anthropic/claude-opus-5-5:nitro", false},
		{"two qualifier words", "claude-opus-5-5-thinking", "claude-opus-5-5-fast", false},
		{"a fast against a preview", "claude-opus-5-5-fast", "claude-opus-5-5-preview", false},
		{"a high against a low", "claude-opus-5-5-high", "claude-opus-5-5-low", false},
		{"an exp against a latest", "gemini-2-5-pro-exp", "gemini-2.5-pro-latest", false},
		{"two context sizes", "claude-opus-5-5-1m", "claude-opus-5-5-200k", false},
		{"two four-digit snapshot tags", "gpt-4-0613", "gpt-4-1106", false},
		{"two three-digit snapshot tags", "gemini-1.5-pro-001", "gemini-1.5-pro-002", false},
		{"two cloud revisions", "us.anthropic.claude-opus-5-5-v1:0", "us.anthropic.claude-opus-5-5-v2:0", false},
		{"a mode after the model's segment", "anthropic/claude-opus-5-5/thinking", "claude-opus-5-5", false},
		{"two unlisted carriers before a colon", "fireworks:llama-3", "together:llama-3", false},
		{"a run-together spelling", "opus5.5", "claude-opus-5-5", false},
		{"fused digits against a dashed opus", "opus5", "opus-5", false},
		{"fused digits against dashed ones", "gpt51", "gpt-5-1", false},
		{"a carrier word with a latest alias", "claude-latest", "claude-sonnet-5-5", false},
		{"an alias with no family word", "default", "claude-sonnet-5-5", false},
		{"a best alias", "best", "claude-opus-5-5", false},
		{"a planning alias run together", "opusplan", "claude-sonnet-5-5", false},
		{"a name holding two families", "claude-opus-sonnet-5", "gpt-5", false},
		{"names of one family that nest", "gpt-5", "gpt-5-mini", false},
		{"an unlisted family", "acme-turbo-2", "claude-opus-5-5", false},
		{"a family written after a vendor alone", "fable", "claude-fable-5-2", false},
		{"a version against a later one", "claude-opus-5-5", "claude-opus-4-6", true},
		{"a number before the family is no version", "claude-3-opus", "claude-opus-4", false},
		{"a version written fused against a dashed other", "opus5", "opus-4", true},
		{"two families of two vendors", "claude-opus-5-5", "openai/gpt-5", true},
		{"two o-series models", "o1", "openai/o3-mini", true},
		{"an o-series model against gpt", "o3", "gpt-4o", true},
		{"two gemini versions", "gemini-2.5-pro", "gemini-2.0-flash", true},
		{"two llama versions at one size", "llama-3.1-70b", "llama-3.3-70b", true},
		{"a v before the number ends the version", "deepseek-v3", "deepseek-v2", false},
		{"a local devstral against a dated one", "local/devstral", "devstral-small-2505", false},
		{"two of Mistral's families", "local/devstral", "codestral-2501", true},
		{"magistral against mistral", "magistral-medium-1", "mistral-medium-3", true},
		{"ministral against pixtral", "ministral-8b", "pixtral-12b", true},
		// A number that is not the model's version never joins it.
		{"a Bedrock inference-profile ARN", "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-opus-5-5-v1:0", "anthropic/claude-opus-5-5", false},
		{"a Bedrock ARN against another family", "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-opus-5-5-v1:0", "anthropic/claude-sonnet-5-5", true},
		{"a numbered region before the family", "us-east-1:claude-opus-5-5", "claude-opus-5-5", false},
		{"a region glued to its number in a path", "vertex_ai/us-central1/claude-opus-5-5", "anthropic/claude-opus-5-5", false},
		{"a region number in a deployment path", "azure/eastus2/gpt-5", "openai/gpt-5", false},
		{"a European region in a path", "vertex_ai/europe-west4/claude-opus-5-5", "claude-opus-5-5", false},
		{"a region before the family in one segment", "google-vertex:us-east5:claude-opus-5-5@20261001", "claude-opus-5-5", false},
		{"two dashed dates of one model", "claude-opus-5-5-2026-10-14", "claude-opus-5-5-2026-11-13", false},
		{"two dashed dates with months and days past ten", "anthropic/claude-opus-5-5-2026-10-15", "claude-opus-5-5-2026-11-20", false},
		{"two dashed dates of a dotted version", "gpt-5.1-2025-11-13", "gpt-5.1-2025-12-01", false},
		{"two dashed dates after a family with no version", "claude-3-7-sonnet-2025-02-19", "claude-3-7-sonnet-2025-02-20", false},
		{"two revision tags with no colon", "claude-opus-4-5-v2", "claude-opus-4-5-v3", false},
		{"a revision tag against none", "claude-opus-5-5-v1:0", "claude-opus-5-5", false},
		{"a vendor's p spelling", "accounts/fireworks/models/llama-v3p1-70b-instruct", "llama-3.1-70b", false},
		{"an older order reads as any of its family", "claude-3-5-sonnet-20241022", "claude-sonnet-4-5", false},
		// A carrier word that is itself a family stands before the slash.
		{"a provider-prefixed codestral", "mistral/codestral-latest", "anthropic/claude-opus-5-5", true},
		{"a provider-prefixed codestral against a dated one", "mistral/codestral-latest", "codestral-2501", false},
		{"two provider-prefixed Mistral families", "mistral/devstral-medium", "mistral/pixtral-large", true},
		{"ministral against magistral, both prefixed", "mistral/ministral-8b", "mistral/magistral-medium", true},
		{"mistral against codestral, both prefixed", "mistral/mistral-large-latest", "mistral/codestral-latest", true},
		// A family glued to another word is no family.
		{"codellama is unknown", "codellama-70b", "llama-3.1-70b", false},
		{"chatgpt is unknown", "chatgpt-4o-latest", "claude-opus-5-5", false},
		// Pairs that stay other.
		{"opus against sonnet of one version", "claude-opus-5-5", "claude-sonnet-5-5", true},
		{"two versions of opus", "claude-opus-5-5", "claude-opus-4-8", true},
		{"two versions of gpt", "gpt-5", "gpt-4", true},
		{"the bundled top against the default", "anthropic/claude-opus-5-5", "anthropic/claude-sonnet-5-5", true},
		{"the bundled research class against the small", "anthropic/claude-fable-5", "anthropic/claude-haiku-4-5", true},
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

// TestTheBundledClassesAreKnown: each model the bundled class map ships
// names a family the comparison reads, so a run on the bundled map is never
// handed back for a class no comparison can read.
func TestTheBundledClassesAreKnown(t *testing.T) {
	for _, m := range []string{"anthropic/claude-fable-5", "anthropic/claude-opus-5-5", "anthropic/claude-sonnet-5-5", "anthropic/claude-haiku-4-5"} {
		if !KnownModel(m) {
			t.Errorf("KnownModel(%q) = false, want true", m)
		}
	}
}

// TestAFamilyGluedToAnotherWordIsUnknown: a name whose family stands only
// inside a longer word names no family, and is unknown.
func TestAFamilyGluedToAnotherWordIsUnknown(t *testing.T) {
	for _, m := range []string{"codellama", "codellama-70b", "codegemma-7b", "chatgpt-4o-latest", "codex-mini-latest"} {
		if KnownModel(m) {
			t.Errorf("KnownModel(%q) = true, want false", m)
		}
	}
	for _, m := range []string{"mistral/codestral-latest", "mistral/devstral-medium", "mistral/pixtral-large", "mistral/ministral-8b", "azure/eastus2/gpt-5"} {
		if !KnownModel(m) {
			t.Errorf("KnownModel(%q) = false, want true", m)
		}
	}
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
