package assistedby

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestLabelVersionIsTheReleaseVersionOrDev: a release build names its version;
// anything else is a development build and says so. A value that is not a
// release version is never copied into the trailer.
func TestLabelVersionIsTheReleaseVersionOrDev(t *testing.T) {
	for _, tc := range []struct{ version, want string }{
		{"dev", "dev"},
		{"", "dev"},
		{"v0.12.0", "v0.12.0"},
		{"v1.0.0-rc.1", "v1.0.0-rc.1"},
		{"v1.2.3+build.5", "v1.2.3+build.5"},
		{"0.12.0", "dev"},
		{"v1.2", "dev"},
		{"latest", "dev"},
		{"v1.0.0\nAssisted-by: None", "dev"},
		{"v1.0.0 extra", "dev"},
	} {
		if got := labelVersion(tc.version); got != tc.want {
			t.Errorf("labelVersion(%q) = %q, want %q", tc.version, got, tc.want)
		}
		if got, want := ComposedValue(tc.version), "abcd:"+tc.want; got != want {
			t.Errorf("ComposedValue(%q) = %q, want %q", tc.version, got, want)
		}
	}
}

// gateAssignment reads one single-quoted assignment out of the attribution gate.
func gateAssignment(t *testing.T, name string) string {
	t.Helper()
	rel := filepath.Join("..", "..", "..", "scripts", "check-attribution.sh")
	b, err := os.ReadFile(rel)
	if err != nil {
		t.Fatalf("read the gate: %v", err)
	}
	m := regexp.MustCompile(`(?m)^` + name + `='([^']*)'$`).FindStringSubmatch(string(b))
	if m == nil {
		t.Fatalf("scripts/check-attribution.sh: no %s assignment found; the gate or this parser changed shape", name)
	}
	return m[1]
}

// TestComposedLabelGrammarMatchesTheGate ties this package's grammar to the
// attribution gate's ABCD_RE and ABCD_ANY_RE, the one place the third form is
// decided: the gate runs in CI without Go, so the two share a test rather than
// code. Every reader of the label (the loop's composer and receipt check, the
// site's tally) goes through this package, so this one tie covers all of them.
func TestComposedLabelGrammarMatchesTheGate(t *testing.T) {
	abcdRE := gateAssignment(t, "ABCD_RE")
	if want := `^Assisted-by: ` + Vendor + `:(` + DevLabel + `|` + ReleaseVersionPattern + `)$`; abcdRE != want {
		t.Fatalf("the gate's ABCD_RE is\n\t%s\nbut internal/core/assistedby reconstructs\n\t%s\none of the two moved alone", abcdRE, want)
	}
	if got, want := gateAssignment(t, "ABCD_ANY_RE"), `^Assisted-by: [Aa][Bb][Cc][Dd]:`; got != want {
		t.Fatalf("the gate's ABCD_ANY_RE is\n\t%s\nbut internal/core/assistedby's NamesAbcd reads\n\t%s\none of the two moved alone", got, want)
	}
	gate, gateAny := regexp.MustCompile(abcdRE), regexp.MustCompile(gateAssignment(t, "ABCD_ANY_RE"))
	for _, v := range []string{
		"abcd:dev", "abcd:v0.12.0", "abcd:v1.0.0-rc.1", "abcd:v1.2.3+build.5",
		"abcd:latest", "ABCD:v0.12.0", "Abcd:dev", "abcd:", "abcd:dev extra",
		"Claude:claude-opus-5-5", "None", "abcdx:dev", "abcd",
	} {
		line := "Assisted-by: " + v
		if isComposed(v) != gate.MatchString(line) {
			t.Errorf("isComposed(%q) = %v, but the gate's ABCD_RE says %v", v, isComposed(v), gate.MatchString(line))
		}
		if NamesAbcd(v) != gateAny.MatchString(line) {
			t.Errorf("NamesAbcd(%q) = %v, but the gate's ABCD_ANY_RE says %v", v, NamesAbcd(v), gateAny.MatchString(line))
		}
	}
	for _, v := range []string{"dev", "", "v0.12.0", "v1.0.0-rc.1", "v1.2.3+build.5", "latest", "0.12.0"} {
		if c := ComposedValue(v); !isComposed(c) || !gate.MatchString("Assisted-by: "+c) {
			t.Errorf("the gate refuses the composer's own value %q (version %q)", c, v)
		}
	}
}
