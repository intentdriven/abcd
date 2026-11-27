package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// linkedHome makes a home that is a symlink to a directory not named after
// the user, and returns the home as written and the spelling the kernel
// reports for it.
func linkedHome(t *testing.T) (home, resolved string) {
	t.Helper()
	base := t.TempDir()
	target := filepath.Join(base, "Volumes", "Data", "u1042")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "Users"), 0o755); err != nil {
		t.Fatal(err)
	}
	home = filepath.Join(base, "Users", "maya")
	if err := os.Symlink(target, home); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	return home, resolved
}

// The literal backstop reads the home in every spelling a path carries it in
// (fsutil.HomeSpellings), through the same escaped views it reads the home as
// written in: the symlink-resolved spelling, which names no user at all; the
// cleaned spelling of a HOME written with a doubled slash; and the other
// Unicode normal form of a non-ASCII home (iss-2609290055092630).
func TestSweepCallerHomeReadsEverySpellingOfTheHome(t *testing.T) {
	linked, resolved := linkedHome(t)
	base := t.TempDir()
	const nfc, nfd = "café", "café"
	for _, c := range []struct{ name, home, in, want string }{
		{"resolved spelling", linked, "open " + resolved + "/notes: denied", "open ~/notes: denied"},
		{"resolved spelling, JSON-escaped", linked, `{"cwd":"` + escapeSeparators(resolved+"/notes", `\/`) + `"}`, `{"cwd":"~\/notes"}`},
		{"resolved spelling, percent-encoded", linked, "u=" + escapeSeparators(resolved+"/notes", `%2F`), "u=~%2Fnotes"},
		{"cleaned spelling of a doubled slash", base + "//maya", "open " + base + "/maya/notes: denied", "open ~/notes: denied"},
		{"raw spelling of a doubled slash", base + "//maya", "open " + base + "//maya/notes: denied", "open ~/notes: denied"},
		{"NFD text under an NFC home", "/Users/" + nfc, "open /Users/" + nfd + "/notes: denied", "open ~/notes: denied"},
		{"NFC text under an NFD home", "/Users/" + nfd, "open /Users/" + nfc + "/notes: denied", "open ~/notes: denied"},
		{"NFD text under an NFC home, JSON-escaped", "/Users/" + nfc, escapeSeparators("/Users/"+nfd+"/notes", `\/`), `~\/notes`},
		// A dot-segment home keeps the separator before a name under it.
		{"a trailing dot segment", "/Users/maya/.", "/Users/maya/._x and /Users/maya/./y", "~/._x and ~/y"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := SweepCallerHome(c.in, c.home); got != c.want {
				t.Errorf("SweepCallerHome(%q, home=%q) = %q, want %q", c.in, c.home, got, c.want)
			}
		})
	}
}

// The residual check judges the swept text by the same spellings, so a home
// the sweep took out is not reported as surviving, and one left in any
// spelling is. The username it rewrites is the cleaned home's last segment,
// so a HOME written with a trailing dot segment still names its user.
func TestSurvivingCallerHomeReadsEverySpellingOfTheHome(t *testing.T) {
	linked, resolved := linkedHome(t)
	if _, resid := SurvivingCallerHome("open "+resolved+"/notes: denied", linked); len(resid) == 0 {
		t.Error("the home left whole in its resolved spelling was not reported")
	}
	if _, resid := SurvivingCallerHome(SweepCallerHome("open "+resolved+"/notes", linked), linked); len(resid) != 0 {
		t.Errorf("the swept text was reported as still carrying the home: %+v", resid)
	}
	got, _ := SurvivingCallerHome("see /home/maya/x", "/Users/maya/.")
	if want := "see /home/" + redactionReplacement(Finding{Kind: kindLocalUser}) + "/x"; got != want {
		t.Errorf("SurvivingCallerHome under a trailing-dot home = %q, want %q", got, want)
	}
}

// RedactRefusal, the refusal path's redaction, sweeps the caller's home in
// every spelling too.
func TestRedactRefusalSweepsTheResolvedSpellingOfTheHome(t *testing.T) {
	linked, resolved := linkedHome(t)
	t.Setenv("HOME", linked)
	t.Setenv(abcdhome.StageHomeEnv, "")
	in := "refused: open " + resolved + "/notes: denied"
	if got, want := RedactRefusal(t.TempDir(), in), "refused: open ~/notes: denied"; got != want {
		t.Errorf("RedactRefusal(%q) = %q, want %q", in, got, want)
	}
}

// The home_path_self detector reads the same spellings, so a path naming the
// home cleaned, under a HOME written with a doubled slash, is reported and
// redacted as the caller's home whole, not left with only its username
// masked ("/srv/people/[redacted-user]/x", the layout around the name
// surviving).
func TestHomePathSelfDetectsTheCleanedSpellingOfAnUncleanHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("HOME", base+"//zzmaya42")
	t.Setenv(abcdhome.StageHomeEnv, "")
	id := ProbeIdentity(t.TempDir())
	if id.HomeUser != "zzmaya42" {
		t.Errorf("HomeUser = %q, want the cleaned home's last segment", id.HomeUser)
	}
	in := "open " + filepath.Join(base, "zzmaya42", "notes") + ": denied"
	got, _ := Redact(in, ScanText(in, id, nil, DefaultIdentitySeverities(), "x"))
	if want := "open ~/notes: denied"; got != want {
		t.Errorf("Redact(%q) = %q, want %q", in, got, want)
	}
}
