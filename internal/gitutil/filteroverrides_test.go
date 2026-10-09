package gitutil_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
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
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")

	got, err := gitutil.FilterOverrides(repo)
	if err != nil || len(got) != 0 {
		t.Fatalf("no filter configured: got %q, %v; want an empty list and no error", got, err)
	}

	git("config", "filter.lfs.clean", "x")
	git("config", "filter.lfs.required", "true")
	git("config", "filter.a.b.process", "y")
	got, err = gitutil.FilterOverrides(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := "-c filter.lfs.clean= -c filter.lfs.smudge= -c filter.lfs.process= " +
		"-c filter.a.b.clean= -c filter.a.b.smudge= -c filter.a.b.process="
	if strings.Join(got, " ") != want {
		t.Fatalf("got %q\nwant %q", strings.Join(got, " "), want)
	}

	git("config", "filter.x=y.clean", "z")
	if got, err := gitutil.FilterOverrides(repo); err == nil {
		t.Fatalf("a filter name holding '=' must be refused, got %q", got)
	}
}

// TestMergeDriverOverridesReplaceEveryConfiguredDriver: with no driver
// configured only merge.default is pinned; each configured name (a dotted one
// included, and one with no driver line) gets git's built-in merge once; a key
// with no name (merge.ff) names nothing; a name -c cannot carry intact is
// refused.
func TestMergeDriverOverridesReplaceEveryConfiguredDriver(t *testing.T) {
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
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")

	got, err := gitutil.MergeDriverOverrides(repo)
	if err != nil || strings.Join(got, " ") != "-c merge.default=text" {
		t.Fatalf("no driver configured: got %q, %v; want only merge.default pinned", got, err)
	}

	git("config", "merge.ff", "false")
	git("config", "merge.evil.driver", "x %A")
	git("config", "merge.evil.recursive", "binary")
	git("config", "merge.a.b.name", "named only")
	got, err = gitutil.MergeDriverOverrides(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := "-c merge.default=text -c merge.evil.driver=" + gitutil.BuiltinMergeDriver + " -c merge.a.b.driver=" + gitutil.BuiltinMergeDriver
	if strings.Join(got, " ") != want {
		t.Fatalf("got %q\nwant %q", strings.Join(got, " "), want)
	}

	git("config", "merge.x=y.driver", "z")
	if got, err := gitutil.MergeDriverOverrides(repo); err == nil {
		t.Fatalf("a driver name holding '=' must be refused, got %q", got)
	}
}
