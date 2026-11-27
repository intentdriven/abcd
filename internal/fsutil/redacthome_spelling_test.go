package fsutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// homeDir makes a real home directory under a fresh temporary base and returns
// the base, so a test can spell the same directory several ways.
func homeDir(t *testing.T, name string) string {
	t.Helper()
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, name), 0o755); err != nil {
		t.Fatal(err)
	}
	return base
}

// A HOME spelled uncleanly — a doubled slash or a dot segment in it — names
// the same directory as its cleaned spelling, and a message can carry either:
// one built by concatenating onto HOME keeps the raw spelling, and one built
// with filepath.Join (or reported by the kernel) carries the cleaned one.
// RedactHome must redact both, or an absolute home path reaches stderr whole
// (iss-2611270714040589).
func TestRedactHomeRedactsTheRawAndTheCleanedSpellingOfHome(t *testing.T) {
	for _, tc := range []struct{ name, spell string }{
		{"doubled slash", "//alex"},
		{"dot segment", "/./alex"},
		{"trailing dot segment", "/alex/."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := homeDir(t, "alex")
			raw := base + tc.spell
			clean := filepath.Clean(raw)
			t.Setenv("HOME", raw)
			t.Setenv(abcdhome.StageHomeEnv, "")
			for _, c := range []struct{ from, in, want string }{
				{"raw", "open " + raw + "/wt/a: denied", "open ~/wt/a: denied"},
				{"raw", "cannot access " + raw, "cannot access ~"},
				{"cleaned", "open " + clean + "/wt/a: denied", "open ~/wt/a: denied"},
				{"cleaned", "cannot access " + clean, "cannot access ~"},
			} {
				if got := fsutil.RedactHome(c.in); got != c.want {
					t.Errorf("built from the %s HOME: RedactHome(%q) = %q, want %q", c.from, c.in, got, c.want)
				}
			}
		})
	}
}

// One span of text is redacted once: the spellings are matched in a single
// pass, longest first, so the replacement a longer spelling leaves is never
// matched again by a shorter one (a "~" does not continue a path segment, so a
// second pass would read the rest of the path as starting a new one).
func TestRedactHomeRedactsEachSpanOnce(t *testing.T) {
	base := homeDir(t, "alex")
	raw := base + "//alex"
	clean := filepath.Clean(raw)
	t.Setenv("HOME", raw)
	t.Setenv(abcdhome.StageHomeEnv, "")
	in := raw + clean + "/x"
	if got, want := fsutil.RedactHome(in), "~"+clean+"/x"; got != want {
		t.Errorf("RedactHome(%q) = %q, want %q", in, got, want)
	}
}

// A trailing slash on HOME is no part of the directory's name: the raw
// spelling is matched without it, so the separator after it in a path is kept
// ("~/wt", never "~wt").
func TestRedactHomeKeepsTheSeparatorAfterATrailingSlashHome(t *testing.T) {
	base := homeDir(t, "alex")
	raw := base + "/alex/"
	t.Setenv("HOME", raw)
	t.Setenv(abcdhome.StageHomeEnv, "")
	for in, want := range map[string]string{
		raw + "wt/a":           "~/wt/a",
		base + "/alex//wt/a":   "~//wt/a",
		"at " + base + "/alex": "at ~",
	} {
		if got := fsutil.RedactHome(in); got != want {
			t.Errorf("RedactHome(%q) = %q, want %q", in, got, want)
		}
	}
}

// Inside a walled stage the process's HOME is the stage's, and the person's
// own home is named by ABCD_PERSON_HOME; the machine's shared records (the
// status store among them) sit under the person's home, so an error about one
// of them names that home. RedactHome redacts both, the person's in each of
// its spellings too.
func TestRedactHomeRedactsThePersonsHomeInsideAStage(t *testing.T) {
	base := t.TempDir()
	for _, d := range []string{"stage", "person"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	stage := filepath.Join(base, "stage")
	personRaw := base + "//person"
	person := filepath.Clean(personRaw)
	t.Setenv("HOME", stage)
	t.Setenv(abcdhome.StageHomeEnv, stage)
	t.Setenv(abcdhome.PersonHomeEnv, personRaw)
	for in, want := range map[string]string{
		"stamp " + person + "/store/status: denied":    "stamp ~/store/status: denied",
		"stamp " + personRaw + "/store/status: denied": "stamp ~/store/status: denied",
		"open " + stage + "/x: denied":                 "open ~/x: denied",
	} {
		if got := fsutil.RedactHome(in); got != want {
			t.Errorf("RedactHome(%q) = %q, want %q", in, got, want)
		}
	}
}
