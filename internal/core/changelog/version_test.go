package changelog

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
)

func mustSemver(t *testing.T, v string) launch.Semver {
	t.Helper()
	s, err := launch.ParseSemver(v)
	if err != nil {
		t.Fatalf("ParseSemver(%q): %v", v, err)
	}
	return s
}

// TestDeriveNextTable walks every cell of the policy table (plan §3): the
// standard SemVer rows at >= 1.0 and the pre-1.0 rows, where `breaking`
// deliberately bumps the MINOR so a 0.x line can break without claiming the
// stability a 1.0.0 asserts.
func TestDeriveNextTable(t *testing.T) {
	cases := []struct {
		name   string
		prev   string
		bump   Impact
		want   string
		bumped bool
	}{
		{"stable breaking bumps major", "1.2.3", ImpactBreaking, "2.0.0", true},
		{"stable additive bumps minor", "1.2.3", ImpactAdditive, "1.3.0", true},
		{"stable fix bumps patch", "1.2.3", ImpactFix, "1.2.4", true},
		{"stable internal does not bump", "1.2.3", ImpactInternal, "", false},

		{"pre-1.0 breaking bumps minor", "0.3.0", ImpactBreaking, "0.4.0", true},
		{"pre-1.0 additive bumps patch", "0.3.0", ImpactAdditive, "0.3.1", true},
		{"pre-1.0 fix bumps patch", "0.3.0", ImpactFix, "0.3.1", true},
		{"pre-1.0 internal does not bump", "0.3.0", ImpactInternal, "", false},

		{"pre-1.0 breaking never reaches 1.0.0", "0.9.0", ImpactBreaking, "0.10.0", true},
		{"pre-1.0 breaking clears the patch", "0.3.7", ImpactBreaking, "0.4.0", true},
		{"1.0.0 fix bumps patch", "1.0.0", ImpactFix, "1.0.1", true},
		{"0.0.x additive bumps patch", "0.0.1", ImpactAdditive, "0.0.2", true},
		{"an unparsed impact does not bump", "0.3.0", Impact("nonsense"), "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			next, bumped := DeriveNext(mustSemver(t, tc.prev), tc.bump)
			if bumped != tc.bumped {
				t.Fatalf("DeriveNext(%s, %s) bumped = %v, want %v", tc.prev, tc.bump, bumped, tc.bumped)
			}
			if !bumped {
				return
			}
			if got := next.String(); got != tc.want {
				t.Errorf("DeriveNext(%s, %s) = %s, want %s", tc.prev, tc.bump, got, tc.want)
			}
		})
	}
}

// TestDeriveNextNeverDerivesFirstStable pins the policy's hardest line: the
// first 1.0.0 is a deliberate human override, so NO input from a 0.x base may
// produce it. A regression here would silently promise API stability abcd has
// not declared.
func TestDeriveNextNeverDerivesFirstStable(t *testing.T) {
	prevs := []string{"0.0.0", "0.0.9", "0.1.0", "0.3.0", "0.9.9", "0.99.99"}
	for _, prev := range prevs {
		for _, bump := range []Impact{ImpactAdditive, ImpactBreaking, ImpactFix, ImpactInternal} {
			next, bumped := DeriveNext(mustSemver(t, prev), bump)
			if bumped && next.Major != 0 {
				t.Errorf("DeriveNext(%s, %s) = %s: derivation left the 0.x line", prev, bump, next)
			}
		}
	}
}

// TestDeriveNextDropsPrereleaseMetadata pins that a derived version is a
// release core: a base carrying prerelease/build metadata must not smuggle it
// into the next version, which becomes a tag and a manifest value.
func TestDeriveNextDropsPrereleaseMetadata(t *testing.T) {
	prev := mustSemver(t, "1.2.3-rc1+build5")
	next, bumped := DeriveNext(prev, ImpactFix)
	if !bumped {
		t.Fatal("a fix must bump")
	}
	if next.Prerelease != "" || next.Build != "" {
		t.Errorf("derived version carries metadata: %+v", next)
	}
	if next.String() != "1.2.4" {
		t.Errorf("DeriveNext = %s, want 1.2.4", next.String())
	}
}

// TestDeriveNextEmptySetDoesNotBump pins the "nothing to release" signal: the
// max impact of an empty record set is ImpactInternal, which must report a
// distinct no-bump rather than silently returning the previous version (which
// the caller would write as a duplicate heading).
func TestDeriveNextEmptySetDoesNotBump(t *testing.T) {
	prev := mustSemver(t, "0.3.0")
	if _, bumped := DeriveNext(prev, maxImpact(nil)); bumped {
		t.Error("an empty record set must not bump")
	}
}
