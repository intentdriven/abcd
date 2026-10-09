package gitutil

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestFilterOverridesBlanksEveryConfiguredFilter: no filter is an empty list
// and no error; each configured name (a dotted one included) is blanked once
// for clean, smudge and process; a name -c cannot carry intact is refused
// rather than blanked under a different key.
func TestFilterOverridesBlanksEveryConfiguredFilter(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = gitEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")

	got, err := FilterOverrides(repo)
	if err != nil || len(got) != 0 {
		t.Fatalf("no filter configured: got %q, %v; want an empty list and no error", got, err)
	}

	git("config", "filter.lfs.clean", "x")
	git("config", "filter.lfs.required", "true")
	git("config", "filter.a.b.process", "y")
	got, err = FilterOverrides(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := "-c filter.lfs.clean= -c filter.lfs.smudge= -c filter.lfs.process= " +
		"-c filter.a.b.clean= -c filter.a.b.smudge= -c filter.a.b.process="
	if strings.Join(got, " ") != want {
		t.Fatalf("got %q\nwant %q", strings.Join(got, " "), want)
	}

	git("config", "filter.x=y.clean", "z")
	if got, err := FilterOverrides(repo); err == nil {
		t.Fatalf("a filter name holding '=' must be refused, got %q", got)
	}
}
