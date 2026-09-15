package repolint_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/repolint"
)

// iss-2609100505145554: privacy-hygiene flagged the two path shapes the
// conventions themselves mandate, so a repo that follows the convention could
// not pass the lint. Both classes are asserted here, together with the real
// leaks that must keep firing — the record's requirement is that the mandated
// shapes stop firing WITHOUT blinding the detector to a real leak of the same
// shape.

// Class A: a home path whose username segment is a name in the persona registry
// is fixture material the conventions ask examples to use, not a leak.
func TestAC_PrivacyPersonaHomePathIsNotALeak(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"persona posix home", "the fixture lives at /Users/alice/notes.md\n", false},
		{"persona linux home", "the fixture lives at /home/bob/notes.md\n", false},
		{"persona windows home", `the fixture lives at C:\Users\carol\notes.md` + "\n", false},
		{"persona home bare", "HOME=/home/dave\n", false},
		{"late-roster persona", "the fixture lives at /Users/nia/notes.md\n", false},
		// A name that is NOT in the registry is an ordinary username and stays a
		// finding: the exemption is the roster, not "any given name".
		{"non-persona username", "keys at /Users/" + strings.Join([]string{"zq", "xwv"}, "") + "/secret\n", true},
		// The registry spells personas as given names. A segment that merely
		// CONTAINS one is a different account.
		{"persona as a prefix", "keys at /Users/alicexyz/secret\n", true},
		{"persona as a suffix", "keys at /Users/xyzalice/secret\n", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := newFixtureRepo(t).conforming().
				file("reference/paths.md", c.body).
				commit().run()
			got := findingFor(res, "privacy-hygiene") != nil
			if got != c.want {
				t.Fatalf("finding = %v, want %v for %q", got, c.want, c.body)
			}
		})
	}
}

// Class A, the narrowing that keeps the exemption honest: a persona name is
// exempt only while it is not THIS machine's own home user. A developer whose
// account really is "alice" is the one person whose /Users/alice is a real leak,
// and the exemption must not cover them.
func TestAC_PrivacyPersonaExemptionYieldsToTheCallersOwnHome(t *testing.T) {
	b := newFixtureRepo(t).conforming().
		file("reference/paths.md", "the fixture lives at /Users/alice/notes.md\n").
		commit()
	// Set HOME only once the fixture is built: gittest.Env redirects HOME to a
	// temp dir it owns while the repo is created, so an earlier Setenv is
	// replaced. scanner.CallerHome reads $HOME at scan time, so from here the
	// caller's own home IS the persona home — the one machine on which
	// /Users/alice is a real leak rather than a fixture.
	t.Setenv("HOME", "/Users/alice")
	res := b.run()

	if f := findingFor(res, "privacy-hygiene"); f == nil {
		t.Fatal("the caller's OWN home path was exempted as a persona; the persona exemption must yield to it")
	}
}

// Class B: a shared system root is not a home root, so nothing beneath it sits
// in the username position. The product creates such a directory and has to name
// it in comments, tests and install docs.
func TestAC_PrivacySharedRootSubtreeIsNotALeak(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"product data dir", "data at /Users/Shared/abcd-data/x\n", false},
		{"product dir", "notes at /Users/Shared/abcd/notes.md\n", false},
		{"plain file", "report at /Users/Shared/report.txt\n", false},
		{"deep subtree", "cache at /Users/Shared/abcd/cache/v2/blob\n", false},
		{"guest root subtree", "state at /Users/Guest/abcd/state\n", false},
		{"windows public subtree", `report at C:\Users\Public\report.txt` + "\n", false},
		// The anti-shield property iss-153's implementation was defending stays:
		// a traversal segment walks back OUT of the shared root, so the name that
		// follows it IS in the username position again.
		{"parent marker then name", "keys at /Users/Shared/../" + strings.Join([]string{"j", "doe"}, "") + "/keys.txt\n", true},
		{"relative marker then name", "keys at /Users/Shared/./" + strings.Join([]string{"j", "doe"}, "") + "/keys.txt\n", true},
		{"doubled separator then name", "keys at /Users/Shared//" + strings.Join([]string{"j", "doe"}, "") + "/keys.txt\n", true},
		{"windows parent marker then name", `keys at C:\Users\Public\..\` + strings.Join([]string{"j", "doe"}, "") + "\n", true},
		{"windows mixed separator traversal", `keys at C:\Users\Public/../` + strings.Join([]string{"j", "doe"}, "") + "\n", true},
		// A segment that merely BEGINS with a system-directory name is an
		// ordinary account and is not a shared root at all.
		{"segment beginning with a system name", "notes at /Users/sharedstuff/notes.md\n", true},
		// The bare directory and the prose forms stay clean, as iss-153 fixed.
		{"bare system directory", "the installer writes to /Users/Shared\n", false},
		{"prose ellipsis", "privacy-hygiene flags /Users/Shared/... in committed files\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := newFixtureRepo(t).conforming().
				file("reference/paths.md", c.body).
				commit().run()
			got := findingFor(res, "privacy-hygiene") != nil
			if got != c.want {
				t.Fatalf("finding = %v, want %v for %q", got, c.want, c.body)
			}
		})
	}
}

// The record's headline symptom: a repo following the conventions is GREEN, not
// red at baseline, while a repo carrying a real leak still blocks.
func TestAC_PrivacyConventionFollowingRepoIsGreen(t *testing.T) {
	body := "" +
		"Examples use persona homes: /Users/alice/p, /home/bob/q, /Users/carol/r.\n" +
		"The installer writes under /Users/Shared/abcd/ and /Users/Shared/abcd-data/.\n"
	res := newFixtureRepo(t).conforming().
		file("reference/install.md", body).
		commit().run()

	if f := findingFor(res, "privacy-hygiene"); f != nil {
		t.Fatalf("a convention-following repo is not clean: %+v", f)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit = %d, want 0 for a repo that is clean on this rule", res.ExitCode)
	}
	if n := countRulePrivacy(res); n != 0 {
		t.Errorf("privacy-hygiene findings = %d, want 0", n)
	}
}

func countRulePrivacy(res repolint.Result) int {
	n := 0
	for _, f := range res.Findings {
		if f.RuleID == "privacy-hygiene" {
			n++
		}
	}
	return n
}
