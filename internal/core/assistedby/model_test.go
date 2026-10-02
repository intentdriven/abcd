package assistedby

import (
	"regexp"
	"testing"
)

// TestModelValueGrammarMatchesTheGate ties the model form's value grammar to
// the attribution gate's TRAILER_RE, the one place it is decided. The gate runs
// in CI without Go, so the two share a test rather than code; every Go reader of
// the model form (the site's chart, the loop's receipt check) goes through this
// package, so this one tie covers all of them.
func TestModelValueGrammarMatchesTheGate(t *testing.T) {
	trailerRE := gateAssignment(t, "TRAILER_RE")
	if want := `^Assisted-by: ` + modelValuePattern + `$`; trailerRE != want {
		t.Fatalf("the gate's TRAILER_RE is\n\t%s\nbut internal/core/assistedby reconstructs\n\t%s\none of the two moved alone", trailerRE, want)
	}
	gate := regexp.MustCompile(trailerRE)
	for _, v := range []string{
		"Claude:claude-opus-5-5", "Vendor:model-a", "Claude:claude-opus-4-8[1m]",
		"abcd:dev", "None", "Claude", "Claude:", ":model", "1Vendor:model",
		"Vendor:model, with edits", "Vendor:model[", "Vendor:model[1m] extra",
	} {
		if got, want := IsModelValue(v), gate.MatchString("Assisted-by: "+v); got != want {
			t.Errorf("IsModelValue(%q) = %v, but the gate's TRAILER_RE says %v", v, got, want)
		}
	}
}

// TestModelHeadIsTheConformingPrefix: a value whose head conforms is clipped to
// that head; a value with no conforming head names no model.
func TestModelHeadIsTheConformingPrefix(t *testing.T) {
	for _, tc := range []struct{ value, want string }{
		{"Vendor:model-a", "Vendor:model-a"},
		{"Vendor:model-a, with edits", "Vendor:model-a"},
		{"Claude:claude-opus-4-8[1m] and more", "Claude:claude-opus-4-8[1m]"},
		{"Claude", ""},
		{"a sentence somebody typed", ""},
		{"None", ""},
	} {
		if got := ModelHead(tc.value); got != tc.want {
			t.Errorf("ModelHead(%q) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
