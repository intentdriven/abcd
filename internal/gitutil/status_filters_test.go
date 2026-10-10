package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// filterFixture is a repository whose committed file runs through a clean
// filter the repository configures, with the file's saved stat made stale so
// a status re-reads it through that filter. home is the HOME the test runs
// under, holding no ~/.abcd.noindex yet.
func filterFixture(t *testing.T) (r *gittest.Repo, home, mark string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	home = t.TempDir()
	t.Setenv("HOME", home)
	r = gittest.NewRepo(t)
	mark = filepath.Join(t.TempDir(), "filter-ran")
	filter := filepath.Join(t.TempDir(), "evil-clean.sh")
	if err := os.WriteFile(filter, []byte("#!/bin/sh\ntouch "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Write("a.txt", "a\n")
	r.Commit("seed")
	r.Git("config", "filter.evil.clean", filter)
	if err := os.WriteFile(filepath.Join(r.Root(), ".git", "info", "attributes"), []byte("* filter=evil\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale(t, r)
	return r, home, mark
}

// stale moves the file's mtime so the index's saved stat no longer matches
// and git must hash the file again to say whether it changed.
func stale(t *testing.T, r *gittest.Repo) {
	t.Helper()
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(r.Root(), "a.txt"), later, later); err != nil {
		t.Fatal(err)
	}
}

// declare writes ~/.abcd.noindex/filter-roots under home with body at mode.
func declare(t *testing.T, home, body string, mode os.FileMode) string {
	t.Helper()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	p := abcdhome.Path(home, "filter-roots")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func ran(mark string) bool {
	_, err := os.Stat(mark)
	return err == nil
}

// TestStatusRunsNoContentFilterUnlessTheOwnerSwitchesThemOn is the everyday
// reads' half of iss-2610090821548169. Status runs a working-tree git status,
// which re-reads a file whose saved stat is stale through the repository's
// filter.<name>.clean, a program the repository names. Filters are off for it
// by default; the owner switches them back on for a checkout by listing its
// absolute path in ~/.abcd.noindex/filter-roots, a file in their own home that
// a repository cannot write. A listing in a file anyone else can write, or one
// reached through a symlink, switches nothing on.
func TestStatusRunsNoContentFilterUnlessTheOwnerSwitchesThemOn(t *testing.T) {
	t.Run("off by default", func(t *testing.T) {
		r, _, mark := filterFixture(t)
		// The fixture is live: a plain git status runs the filter.
		plain := exec.Command("git", "-C", r.Root(), "status", "--porcelain")
		plain.Env = r.Env()
		if out, err := plain.CombinedOutput(); err != nil {
			t.Fatalf("fixture: git status: %v\n%s", err, out)
		}
		if !ran(mark) {
			t.Fatal("fixture: a plain git status did not run the clean filter, so this test proves nothing")
		}
		if err := os.Remove(mark); err != nil {
			t.Fatal(err)
		}
		stale(t, r)

		entries, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if ran(mark) {
			t.Fatal("Status ran the repository's clean filter with no declaration")
		}
		if len(entries) != 0 {
			t.Fatalf("an unchanged file reads as unchanged with filters off: %+v", entries)
		}
	})

	t.Run("on for a listed checkout", func(t *testing.T) {
		r, home, mark := filterFixture(t)
		declare(t, home, "# checkouts whose filters run\n/elsewhere\n"+r.Root()+"\n", 0o600)
		if _, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{}); err != nil {
			t.Fatal(err)
		}
		if !ran(mark) {
			t.Fatal("Status must run the clean filter for a checkout listed in ~/.abcd.noindex/filter-roots")
		}
	})

	t.Run("off for a checkout not listed", func(t *testing.T) {
		r, home, mark := filterFixture(t)
		declare(t, home, "/elsewhere\n# "+r.Root()+"\n", 0o600)
		if _, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{}); err != nil {
			t.Fatal(err)
		}
		if ran(mark) {
			t.Fatal("a commented-out entry switched the filters on")
		}
	})

	for _, mode := range []os.FileMode{0o620, 0o602} {
		t.Run("ignored when writable by others "+mode.String(), func(t *testing.T) {
			r, home, mark := filterFixture(t)
			declare(t, home, r.Root()+"\n", mode)
			if _, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{}); err != nil {
				t.Fatal(err)
			}
			if ran(mark) {
				t.Fatalf("a filter-roots file at mode %v switched the filters on", mode)
			}
		})
	}

	t.Run("ignored when the file is a symlink", func(t *testing.T) {
		r, home, mark := filterFixture(t)
		real := filepath.Join(t.TempDir(), "roots")
		if err := os.WriteFile(real, []byte(r.Root()+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, abcdhome.Path(home, "filter-roots")); err != nil {
			t.Fatal(err)
		}
		if _, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{}); err != nil {
			t.Fatal(err)
		}
		if ran(mark) {
			t.Fatal("a symlinked filter-roots file switched the filters on")
		}
	})

	t.Run("ignored behind a symlinked home folder", func(t *testing.T) {
		r, home, mark := filterFixture(t)
		elsewhere := t.TempDir()
		if err := os.WriteFile(filepath.Join(elsewhere, "filter-roots"), []byte(r.Root()+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(elsewhere, abcdhome.Path(home)); err != nil {
			t.Fatal(err)
		}
		if _, err := gitutil.Status(r.Root(), 1<<20, gitutil.StatusOptions{}); err != nil {
			t.Fatal(err)
		}
		if ran(mark) {
			t.Fatal("a filter-roots file behind a symlinked ~/.abcd.noindex switched the filters on")
		}
	})
}

// TestFilterRootsIgnoredNamesTheFileAndTheCheck is iss-2610091920437492: a
// filter-roots file abcd ignores is named with the check it failed, for a
// front door to report; an absent or honoured file names nothing.
func TestFilterRootsIgnoredNamesTheFileAndTheCheck(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if note := gitutil.FilterRootsIgnored(); note != "" {
		t.Fatalf("an absent file names nothing, got %q", note)
	}
	p := declare(t, home, "/some/checkout\n", 0o600)
	if note := gitutil.FilterRootsIgnored(); note != "" {
		t.Fatalf("an honoured file names nothing, got %q", note)
	}
	if err := os.Chmod(p, 0o602); err != nil {
		t.Fatal(err)
	}
	note := gitutil.FilterRootsIgnored()
	if !strings.Contains(note, abcdhome.Display("filter-roots")) || !strings.Contains(note, "writable by others") {
		t.Fatalf("the note must name the file and the check it failed, got %q", note)
	}
	if on, why := gitutil.FiltersSwitchedOn("/some/checkout"); on || why != note {
		t.Fatalf("FiltersSwitchedOn must give the same note and switch nothing on: %v %q", on, why)
	}
}
