package repolint_test

import (
	"testing"
)

// TestPrivacyGoCallArgumentSelectorIsGoOnly pins where the call-argument
// selector exemption applies (iss-2610020840196926, narrowed after
// sec-scannerSel): the privacy rule knows the file it reads, so a Go selector
// closing a call's argument list — the field local of anchor in
// internal/core/capture/eligible.go — is code in a .go file and is not a LAN
// host there. The very same line in a markdown file is not Go, and a host
// written as a call's argument in prose, a log or a diagram is reported.
func TestPrivacyGoCallArgumentSelectorIsGoOnly(t *testing.T) {
	line := "\tcase deferredOK && launch.CoreGreater(deferred, " + joinDots("anchor", "local") + "):\n"
	upper := "\treturn ssh(" + joinDots("NAS", "local") + ")\n"
	cases := []struct {
		name, rel, body string
		flagged         bool
	}{
		{"go selector in a go file", "internal/x/eligible.go", "package x\n" + line, false},
		{"same line in markdown", "reference/notes.md", line, true},
		{"same line in a text log", "reference/run.log", line, true},
		{"upper-case label in a go file", "internal/x/dial.go", "package x\n" + upper, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := newFixtureRepo(t).conforming().file(c.rel, c.body).commit().run()
			f := findingFor(res, "privacy-hygiene")
			if c.flagged {
				if f == nil {
					t.Fatalf("no privacy-hygiene finding for %s", c.rel)
				}
				if f.File != c.rel {
					t.Errorf("finding cites %s, want %s", f.File, c.rel)
				}
				return
			}
			if f != nil {
				t.Fatalf("Go selector flagged in %s: %+v", c.rel, f)
			}
		})
	}
}
