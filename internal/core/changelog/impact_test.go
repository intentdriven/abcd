package changelog

import (
	"strings"
	"testing"
)

// TestParseImpactAcceptsEveryConstant pins the wire spelling of each enum member:
// the frontmatter value a record carries must round-trip to its constant, so a
// renamed constant can never silently change what records mean.
func TestParseImpactAcceptsEveryConstant(t *testing.T) {
	cases := []struct {
		value string
		want  Impact
	}{
		{"additive", ImpactAdditive},
		{"breaking", ImpactBreaking},
		{"fix", ImpactFix},
		{"internal", ImpactInternal},
	}
	for _, tc := range cases {
		got, err := ParseImpact(tc.value)
		if err != nil {
			t.Errorf("ParseImpact(%q) returned error %v, want %v", tc.value, err, tc.want)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseImpact(%q) = %q, want %q", tc.value, got, tc.want)
		}
		if string(tc.want) != tc.value {
			t.Errorf("constant %q does not carry its wire spelling %q", tc.want, tc.value)
		}
	}
}

// TestParseImpactRejectsEverythingElse proves the exact-match contract. Nothing
// is case-folded and nothing is trimmed: frontmatter.Fields already trims the
// value it hands over, so any surviving whitespace is a real authoring defect
// and must be reported rather than absorbed.
func TestParseImpactRejectsEverythingElse(t *testing.T) {
	invalid := []string{
		"Additive",     // case is not folded
		"BREAKING",     // case is not folded
		"major",        // a SemVer bump name is not an impact
		"minor",        //
		"patch",        //
		"  fix  ",      // no trimming beyond what frontmatter parsing did
		"internal ",    // trailing space is a defect, not an internal record
		"fix\n",        // a stray newline is not a fix
		"additive,fix", // one judgement per record
		"null",         // a YAML null is not an impact
		"~",            //
	}
	for _, v := range invalid {
		got, err := ParseImpact(v)
		if err == nil {
			t.Errorf("ParseImpact(%q) = %q, want an error", v, got)
			continue
		}
		if got != "" {
			t.Errorf("ParseImpact(%q) returned %q alongside its error, want the zero Impact", v, got)
		}
	}
}

// TestParseImpactEmptyIsRequiredNotDefaulted is the guard for outcome 8: the old
// "issues default to fix" rule under-bumped feature-adding issues, so an absent
// value must fail loudly and the message must say the field is required.
func TestParseImpactEmptyIsRequiredNotDefaulted(t *testing.T) {
	got, err := ParseImpact("")
	if err == nil {
		t.Fatalf("ParseImpact(\"\") = %q, want an error (there is no default)", got)
	}
	if got != "" {
		t.Errorf("ParseImpact(\"\") returned %q alongside its error, want the zero Impact", got)
	}
	msg := err.Error()
	if !strings.Contains(msg, "required") {
		t.Errorf("ParseImpact(\"\") error = %q, want it to say the field is required", msg)
	}
	if !strings.Contains(msg, "no default") {
		t.Errorf("ParseImpact(\"\") error = %q, want it to say there is no default", msg)
	}
}

// TestParseImpactErrorNamesTheEnum keeps the boundary message actionable: a
// maintainer who mistypes the field learns the whole legal set from the error.
func TestParseImpactErrorNamesTheEnum(t *testing.T) {
	for _, v := range []string{"", "Additive"} {
		_, err := ParseImpact(v)
		if err == nil {
			t.Fatalf("ParseImpact(%q) unexpectedly succeeded", v)
		}
		msg := err.Error()
		for _, want := range []string{"additive", "breaking", "fix", "internal"} {
			if !strings.Contains(msg, want) {
				t.Errorf("ParseImpact(%q) error = %q, want it to name %q", v, msg, want)
			}
		}
	}
}

// TestMaxImpactOrdering pins the single ordering breaking > additive > fix >
// internal, including ties and the empty cut.
func TestMaxImpactOrdering(t *testing.T) {
	cases := []struct {
		name string
		in   []Impact
		want Impact
	}{
		{"empty cut is internal (nothing to release)", nil, ImpactInternal},
		{"empty slice is internal", []Impact{}, ImpactInternal},
		{"single fix", []Impact{ImpactFix}, ImpactFix},
		{"single internal", []Impact{ImpactInternal}, ImpactInternal},
		{"tie on fix", []Impact{ImpactFix, ImpactFix}, ImpactFix},
		{"tie on breaking", []Impact{ImpactBreaking, ImpactBreaking}, ImpactBreaking},
		{"additive beats fix", []Impact{ImpactFix, ImpactAdditive}, ImpactAdditive},
		{"additive beats fix, order-independent", []Impact{ImpactAdditive, ImpactFix}, ImpactAdditive},
		{"fix beats internal", []Impact{ImpactInternal, ImpactFix}, ImpactFix},
		{"additive beats internal", []Impact{ImpactInternal, ImpactAdditive}, ImpactAdditive},
		{"breaking beats additive", []Impact{ImpactAdditive, ImpactBreaking}, ImpactBreaking},
		{"breaking beats everything", []Impact{ImpactInternal, ImpactFix, ImpactBreaking, ImpactAdditive}, ImpactBreaking},
		{"all internal stays internal", []Impact{ImpactInternal, ImpactInternal}, ImpactInternal},
		{"an unrecognised value never wins", []Impact{Impact("wat"), ImpactFix}, ImpactFix},
		{"an unrecognised value alone falls to internal", []Impact{Impact("wat")}, ImpactInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := maxImpact(tc.in); got != tc.want {
				t.Errorf("MaxImpact(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestMaxImpactDoesNotMutateInput proves the helper is pure: the next stage
// passes the cut's impacts around and must not find them reordered.
func TestMaxImpactDoesNotMutateInput(t *testing.T) {
	in := []Impact{ImpactFix, ImpactBreaking, ImpactInternal}
	_ = maxImpact(in)
	want := []Impact{ImpactFix, ImpactBreaking, ImpactInternal}
	for i := range want {
		if in[i] != want[i] {
			t.Fatalf("MaxImpact reordered its input: got %v, want %v", in, want)
		}
	}
}

// TestPredicates pins outcome 8's single rule from both sides: internal drives
// no bump AND is excluded from the changelog; every other member does both.
func TestPredicates(t *testing.T) {
	cases := []struct {
		in          Impact
		drivesBump  bool
		inChangelog bool
	}{
		{ImpactBreaking, true, true},
		{ImpactAdditive, true, true},
		{ImpactFix, true, true},
		{ImpactInternal, false, false},
		{Impact(""), false, false},
		{Impact("wat"), false, false},
	}
	for _, tc := range cases {
		if got := tc.in.DrivesBump(); got != tc.drivesBump {
			t.Errorf("Impact(%q).DrivesBump() = %v, want %v", tc.in, got, tc.drivesBump)
		}
		if got := tc.in.InChangelog(); got != tc.inChangelog {
			t.Errorf("Impact(%q).InChangelog() = %v, want %v", tc.in, got, tc.inChangelog)
		}
	}
}
