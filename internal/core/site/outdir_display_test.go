package site

import (
	"path/filepath"
	"testing"
)

// The output directory the site verbs report travels into `--json`, and machine
// output never carries an absolute developer-identity path (iss-81). An output
// directory inside the repository is reported relative to it; one outside it is
// reported with the home directory redacted to "~", so an absolute --out under
// the home — the shape that named the developer — reads as ~/… on every verb
// (iss-2608291957114882).
func TestTheSiteVerbsReportTheOutputDirectoryWithoutTheHomePath(t *testing.T) {
	f := newFixture(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	for name, tc := range map[string]struct{ out, want string }{
		"inside the repository": {filepath.Join(f.Root(), "public"), "public"},
		"under the home":        {filepath.Join(home, "public"), "~/public"},
	} {
		st, err := Describe(f.Root(), tc.out)
		if err != nil {
			t.Fatalf("%s: describe: %v", name, err)
		}
		if st.OutDir != tc.want {
			t.Errorf("%s: Status.OutDir = %q, want %q", name, st.OutDir, tc.want)
		}

		res := buildFixture(t, f, tc.out)
		if res.OutDir != tc.want {
			t.Errorf("%s: Result.OutDir = %q, want %q", name, res.OutDir, tc.want)
		}

		chk, err := Check(CheckRequest{RepoRoot: f.Root(), OutDir: tc.out})
		if err != nil {
			t.Fatalf("%s: check: %v", name, err)
		}
		if chk.OutDir != tc.want {
			t.Errorf("%s: CheckResult.OutDir = %q, want %q", name, chk.OutDir, tc.want)
		}
	}

	// A relative --out is reported as it was given.
	st, err := Describe(f.Root(), DefaultOutDir)
	if err != nil {
		t.Fatal(err)
	}
	if st.OutDir != DefaultOutDir {
		t.Errorf("Status.OutDir = %q, want %q", st.OutDir, DefaultOutDir)
	}
}
