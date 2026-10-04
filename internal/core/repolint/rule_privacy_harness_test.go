package repolint_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/repolint"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// synthSessionURL builds a session-identifier-shaped URL at RUNTIME, seeded per
// case, for the reason the scanner's own fixtures do (internal/adapter/scanner/
// harnessleak_test.go, and the secret-shaped-fixtures-at-runtime principle):
// full-history scanning plus a main that cannot be force-pushed makes a
// committed identifier literal permanent. The generated value is not real, so
// this file needs none of the escapes a literal would — no split string, no
// line waiver, and no reserved documentation host standing in for a specimen.
func synthSessionURL(host string, seed uint64) string {
	return "https://" + host + "/code/session_" + testsecret.Synthetic(seed, 22)
}

// harnessFooter is the banned attribution shape, spelled out. A Go source line
// puts prose before the match and the pattern requires a footer to occupy its
// own line, so it needs no evasion either.
const harnessFooter = "Generated with [Some Tool](https://tool.dev)"

// The harness-leak class reaches `abcd lint`'s privacy rule over ANY committed
// file, not only over a freshly created pull-request body: the leak found in
// this repository was model-authored and landed in tracked prose, and the
// post-create strip cannot see a file that was committed rather than posted.
func TestAC_PrivacyHarnessLeakInCommittedFile(t *testing.T) {
	sessionURL := synthSessionURL("agent-host.dev", 17)

	cases := []struct {
		name string
		body string
	}{
		{"live session url", "Run recorded at " + sessionURL + "\n"},
		{"tool attribution footer", harnessFooter + "\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newFixtureRepo(t).conforming().
				file("reference/notes.md", c.body).
				commit()
			res := b.run()

			f := findingFor(res, "privacy-hygiene")
			if f == nil {
				t.Fatalf("no privacy-hygiene finding for %q", c.body)
			}
			if f.Severity != repolint.SeverityError {
				t.Errorf("severity = %q, want %q", f.Severity, repolint.SeverityError)
			}
			if f.File != "reference/notes.md" || f.Line != 1 {
				t.Errorf("citation = %s:%d, want reference/notes.md:1", f.File, f.Line)
			}
			if !strings.Contains(f.Fix, "Assisted-by") {
				t.Errorf("the finding does not name the sanctioned alternative; Fix = %q", f.Fix)
			}
		})
	}
}

// The negative half: prose ABOUT the ban, and a worked example pointing at a
// reserved documentation host, are how the convention gets written down. A rule
// that flags its own documentation is a rule people route around.
func TestAC_PrivacyHarnessLeakSparesProseAndExamples(t *testing.T) {
	body := strings.Join([]string{
		"We ban the \"Generated with [tool](url)\" footer in public text.",
		"- Generated with [Some Tool](https://example.invalid) is refused.",
		"A session link looks like " + synthSessionURL("example.invalid", 23) + ".",
	}, "\n") + "\n"

	b := newFixtureRepo(t).conforming().
		file("reference/attribution.md", body).
		commit()
	res := b.run()

	if f := findingFor(res, "privacy-hygiene"); f != nil {
		t.Fatalf("unexpected privacy-hygiene finding: %s:%d %s", f.File, f.Line, f.Message)
	}
}

// TestAC_PrivacyLeavesAPlainSKKeyToTheRedactors: the plain sk- rule only warns,
// so a hash-like string in a committed file cannot fail a gate on it. The
// write-side sweeps and backstops treat it as the token it names whatever its
// severity; abcd lint judges files already committed and stays out of it, bare
// or glued behind a word character. The sample is built at runtime.
func TestAC_PrivacyLeavesAPlainSKKeyToTheRedactors(t *testing.T) {
	key := "sk-" + testsecret.Synthetic(63, 40)
	res := newFixtureRepo(t).conforming().
		file("reference/notes.md", "bare "+key+"\nglued my_"+key+"\n").
		commit().
		run()
	if len(res.Findings) != 0 || res.ExitCode != 0 {
		t.Fatalf("a committed plain sk- key failed lint (exit %d): %+v", res.ExitCode, res.Findings)
	}
}
