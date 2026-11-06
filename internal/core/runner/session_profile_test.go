package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestSessionOnProfile holds every shape the setting can take. The answer
// reads one setting and proves no wall, and every doubt reads as off a
// profile, since a warning printed once too often costs a line.
func TestSessionOnProfile(t *testing.T) {
	mkdir := func(t *testing.T, path string) string {
		t.Helper()
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		return path
	}
	link := func(t *testing.T, target, path string) string {
		t.Helper()
		mkdir(t, filepath.Dir(path))
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		return path
	}
	profile := func(t *testing.T, home string) string {
		t.Helper()
		return mkdir(t, abcdhome.Path(home, "profiles", "drive"))
	}

	cases := []struct {
		name string
		// value builds the setting under a fresh home.
		value func(t *testing.T, home string) string
		on    bool
		why   string
	}{
		{"a directory directly under the store", profile, true, ""},
		{"the same directory through a link outside the store", func(t *testing.T, home string) string {
			return link(t, profile(t, home), filepath.Join(home, "wall"))
		}, true, ""},
		{"a trailing slash and a dot segment", func(t *testing.T, home string) string {
			return profile(t, home) + "/./"
		}, true, ""},
		{"unset or empty, as another host reports nothing", func(t *testing.T, home string) string {
			return ""
		}, false, "is not set"},
		{"relative, a name for whatever directory the tool starts in", func(t *testing.T, home string) string {
			return filepath.Join(abcdhome.Rel(), "profiles", "drive")
		}, false, "is relative"},
		{"a tilde no shell expanded", func(t *testing.T, home string) string {
			profile(t, home)
			return abcdhome.Display("profiles", "drive")
		}, false, "is relative"},
		{"a directory that does not exist", func(t *testing.T, home string) string {
			return abcdhome.Path(home, "profiles", "missing")
		}, false, "names no directory"},
		{"a file", func(t *testing.T, home string) string {
			mkdir(t, abcdhome.Path(home, "profiles"))
			p := abcdhome.Path(home, "profiles", "file")
			if err := os.WriteFile(p, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			return p
		}, false, "names no directory"},
		{"the person's own directory", func(t *testing.T, home string) string {
			return mkdir(t, filepath.Join(home, ".claude"))
		}, false, "names no profile"},
		{"the store itself, which is no profile", func(t *testing.T, home string) string {
			profile(t, home)
			return abcdhome.Path(home, "profiles")
		}, false, "names no profile"},
		{"a directory inside a profile, which is no profile", func(t *testing.T, home string) string {
			return mkdir(t, filepath.Join(profile(t, home), "inner"))
		}, false, "names no profile"},
		{"a name that only begins like the store", func(t *testing.T, home string) string {
			return mkdir(t, abcdhome.Path(home, "profiles-old", "drive"))
		}, false, "names no profile"},
		{"a dot-dot out of the store", func(t *testing.T, home string) string {
			profile(t, home)
			mkdir(t, filepath.Join(home, ".claude"))
			return abcdhome.Path(home, "profiles") + "/../../.claude"
		}, false, "names no profile"},
		{"a link in the store that leaves it", func(t *testing.T, home string) string {
			return link(t, mkdir(t, filepath.Join(home, ".claude")), abcdhome.Path(home, "profiles", "drive"))
		}, false, "names no profile"},
		{"a store that is itself a link", func(t *testing.T, home string) string {
			mkdir(t, filepath.Join(home, "elsewhere", "drive"))
			link(t, filepath.Join(home, "elsewhere"), abcdhome.Path(home, "profiles"))
			return abcdhome.Path(home, "profiles", "drive")
		}, false, "names no profile"},
		{"abcd's home that is itself a link", func(t *testing.T, home string) string {
			mkdir(t, filepath.Join(home, "dotfiles", "profiles", "drive"))
			link(t, filepath.Join(home, "dotfiles"), abcdhome.Path(home))
			return abcdhome.Path(home, "profiles", "drive")
		}, false, "names no profile"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			on, why := SessionOnProfile(home, c.value(t, home))
			if on != c.on {
				t.Fatalf("on = %v (%q), want %v", on, why, c.on)
			}
			if !strings.Contains(why, c.why) {
				t.Fatalf("why = %q, want it to carry %q", why, c.why)
			}
			if c.on && why != "" {
				t.Fatalf("a session on a profile has no why: %q", why)
			}
			for _, never := range []string{home, ConfigDirEnv, abcdhome.Display("profiles"), "profiles", "under"} {
				if strings.Contains(why, never) {
					t.Fatalf("why carries %q: %q", never, why)
				}
			}
		})
	}
}

// TestSessionOnProfileBehindALinkedHome: a home reached through a link, as
// macOS's temporary directories and some home directories are, judges the same
// directory the setting resolves to.
func TestSessionOnProfileBehindALinkedHome(t *testing.T) {
	real := t.TempDir()
	dir := abcdhome.Path(real, "profiles", "drive")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(real, linked); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{dir, abcdhome.Path(linked, "profiles", "drive")} {
		if on, why := SessionOnProfile(linked, value); !on {
			t.Errorf("%s: off a profile (%q)", value, why)
		}
	}
}

// TestSessionOnProfileWithNoHome: with no home to read the store under, nothing
// can be judged on a profile.
func TestSessionOnProfileWithNoHome(t *testing.T) {
	dir := t.TempDir()
	for _, home := range []string{"", "relative/home", filepath.Join(dir, "missing")} {
		on, why := SessionOnProfile(home, dir)
		if on || !strings.Contains(why, "no home") {
			t.Errorf("home %q: on = %v, why = %q", home, on, why)
		}
	}
}

// TestASessionOnEitherOfTwoProfilesReadsAsOn: the folder may hold several, and a
// session on any of them is on a profile.
func TestASessionOnEitherOfTwoProfilesReadsAsOn(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		dir := abcdhome.Path(home, "profiles", name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if on, why := SessionOnProfile(home, dir); !on {
			t.Errorf("a session on %s reads as off: it %s", name, why)
		}
	}
}
