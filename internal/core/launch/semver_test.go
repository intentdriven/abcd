package launch

import "testing"

func TestIsStrictSemver(t *testing.T) {
	valid := []string{"0.0.0", "1.2.3", "10.20.30", "1.0.0-alpha", "1.0.0-alpha.1", "1.0.0+build", "1.0.0-rc.1+exp.sha.5"}
	for _, v := range valid {
		if !IsStrictSemver(v) {
			t.Errorf("expected %q valid", v)
		}
	}
	invalid := []string{"v1.2.3", "1.2", "1.2.3.4", "01.2.3", "1.02.3", "1.2.3rc1", "1.2.3\n", "", "1.2.3 "} // abcd-audit:allow — VERSION strings, not addresses
	for _, v := range invalid {
		if IsStrictSemver(v) {
			t.Errorf("expected %q invalid", v)
		}
	}
}

func TestParseSemver(t *testing.T) {
	s, err := ParseSemver("1.2.3-rc.1+b.2")
	if err != nil {
		t.Fatal(err)
	}
	if s.Major != 1 || s.Minor != 2 || s.Patch != 3 || s.Prerelease != "rc.1" || s.Build != "b.2" {
		t.Errorf("bad parse: %+v", s)
	}
	if s.Line() != "1.2" {
		t.Errorf("Line=%q want 1.2", s.Line())
	}
	if s.Tag() != "v1.2.3" {
		t.Errorf("Tag=%q want v1.2.3", s.Tag())
	}
	if _, err := ParseSemver("1.2.3\n"); err == nil {
		t.Error("trailing newline must be rejected")
	}
}

// TestParseSemverRejectsOutOfRangeComponent pins that a component too wide for
// int is rejected rather than silently clamped to MaxInt64 (which would collide
// distinct versions and wrap negative on the next Patch++).
func TestParseSemverRejectsOutOfRangeComponent(t *testing.T) {
	for _, v := range []string{
		"0.0.99999999999999999999",
		"99999999999999999999.0.0",
		"0.88888888888888888888.0",
	} {
		if _, err := ParseSemver(v); err == nil {
			t.Errorf("ParseSemver(%q): want out-of-range error, got nil", v)
		}
	}
}

// TestParseReleaseTag pins the one release-version parser (spc-2610170635097509
// step 1): exactly vMAJOR.MINOR.PATCH, digits only, no leading zero, no
// pre-release or build part, each component bounded, ordered by CoreGreater.
func TestParseReleaseTag(t *testing.T) {
	for _, c := range []struct {
		tag  string
		want Semver
	}{
		{"v0.0.0", Semver{}},
		{"v0.16.0", Semver{Minor: 16}},
		{"v1.2.3", Semver{Major: 1, Minor: 2, Patch: 3}},
		{"v999999999.999999999.999999999", Semver{Major: 999999999, Minor: 999999999, Patch: 999999999}},
	} {
		got, err := ParseReleaseTag(c.tag)
		if err != nil {
			t.Errorf("ParseReleaseTag(%q): %v", c.tag, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseReleaseTag(%q) = %+v, want %+v", c.tag, got, c.want)
		}
	}
	for _, tag := range []string{
		"", "dev", "1.2.3", "V1.2.3", "v1.2", "v1.2.3.4", "v01.2.3", "v1.02.3", "v1.2.03",
		"v1.2.3-rc.1", "v1.2.3+build", "v1.2.3\n", " v1.2.3", "v1.2.3 ", "vv1.2.3",
		"v1.2.3\u0000", "v１.2.3", "v1000000000.0.0", "v0.0.99999999999999999999",
		"v1.2.3\x1b]0;x\x07", "Run rm -rf now",
	} {
		if _, err := ParseReleaseTag(tag); err == nil {
			t.Errorf("ParseReleaseTag(%q) accepted a non-release tag", tag)
		}
	}
	a, _ := ParseReleaseTag("v0.10.0")
	b, _ := ParseReleaseTag("v0.9.12")
	if !CoreGreater(a, b) || CoreGreater(b, a) {
		t.Errorf("v0.10.0 must order above v0.9.12")
	}
}
