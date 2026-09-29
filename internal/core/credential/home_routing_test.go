//go:build unix

package credential

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// linkedDotfilesHome returns a home whose ~/.abcd is a symlink into a git
// working tree (a dotfiles repository beside it, the layout AGENTS.md names),
// and the directory the link points at. The home itself is not a working
// tree, so a judgement of ~/.abcd by its lexical path sees no repository.
func linkedDotfilesHome(t *testing.T) (home, target string) {
	t.Helper()
	home = t.TempDir()
	repo := filepath.Join(home, "dotfiles")
	target = filepath.Join(repo, "abcd")
	for _, dir := range []string{filepath.Join(repo, ".git"), target} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(target, filepath.Join(home, ".abcd")); err != nil {
		t.Fatal(err)
	}
	return home, target
}

// TestSetRefusesEveryHomeThroughAnAbcdHomeLinkedIntoARepository is the
// review's probe A (itd-2609221017023290, criterion 3): a ~/.abcd symlinked
// into a dotfiles repository is refused, naming the link, in every home, before
// anything is created: not the value, not the index, not either lock. A
// working-tree check on the lexical path does not see the repository, so the
// link itself is what is refused.
func TestSetRefusesEveryHomeThroughAnAbcdHomeLinkedIntoARepository(t *testing.T) {
	const envName = "ABCD_TEST_ROUTING_TOKEN"
	t.Setenv(envName, secretValue)
	keychain := withFakeKeychain(t, "security")
	for _, c := range []struct {
		name   string
		choice Choice
	}{
		{"abcd", Choice{Home: HomeABCD, Value: secretValue}},
		{"keychain", Choice{Home: HomeKeychain, Value: secretValue}},
		{"external", Choice{Home: HomeExternal, Pointer: Pointer{Env: envName}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			home, target := linkedDotfilesHome(t)
			changed, err := Set(home, "svc", c.choice)
			if err == nil || changed {
				t.Fatalf("Set wrote through a ~/.abcd linked into a repository: changed %v, err %v", changed, err)
			}
			if !strings.Contains(err.Error(), "~/.abcd is a symlink") {
				t.Errorf("the refusal must name the link: %v", err)
			}
			if strings.Contains(err.Error(), secretValue) {
				t.Fatal("the refusal echoes the value")
			}
			entries, rerr := os.ReadDir(target)
			if rerr != nil {
				t.Fatal(rerr)
			}
			if len(entries) != 0 {
				t.Fatalf("Set left %d file(s) in the repository, first %q", len(entries), entries[0].Name())
			}
			assertNowhere(t, filepath.Join(home, "dotfiles"), secretValue)
		})
	}
	if entries, _ := os.ReadDir(keychain); len(entries) != 0 {
		t.Fatalf("the keychain was written: %d item(s)", len(entries))
	}
}

// TestAPointerWhoseDirectoryLinksIntoARepositoryIsRefused is the review's
// probe C: an external pointer at ~/.config/tool.json, where ~/.config is a
// symlink into a dotfiles repository, is refused naming the link, rather than
// followed and read from inside a working tree; and Set records no such
// pointer.
func TestAPointerWhoseDirectoryLinksIntoARepositoryIsRefused(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "dotfiles")
	cfg := filepath.Join(repo, "cfg")
	for _, dir := range []string{filepath.Join(repo, ".git"), cfg} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(cfg, "tool.json"), []byte(`{"auth":{"token":"`+secretValue+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(cfg, filepath.Join(home, ".config")); err != nil {
		t.Fatal(err)
	}
	p := Pointer{File: "~/.config/tool.json", Field: "auth.token"}
	v, err := resolvePointer(home, "svc", p)
	if err == nil || v != "" {
		t.Fatalf("a pointer through ~/.config linked into a repository was read: value %q, err %v", v, err)
	}
	if !strings.Contains(err.Error(), "~/.config is a symlink") {
		t.Errorf("the refusal must name the link: %v", err)
	}
	if strings.Contains(err.Error(), secretValue) {
		t.Fatal("the refusal echoes the value")
	}
	if _, err := Set(home, "svc", Choice{Home: HomeExternal, Pointer: p}); err == nil {
		t.Fatal("Set recorded a pointer through a link into a repository")
	}
	if _, err := os.Lstat(filepath.Join(home, ".abcd", IndexFileName)); !os.IsNotExist(err) {
		t.Fatalf("the index was written: %v", err)
	}
}

// TestSetWritesNoIndexThroughAnAbcdHomeSwappedForALink is the index's half of
// iss-2609281310017733: ~/.abcd is a real directory when Set judges it and a
// symlink into a dotfiles checkout by a later use (a same-uid race, staged
// through the vetting hook of the ~/.abcd walk). The index, its lock and the
// read of the index are reached through the walk of the directory that was
// judged, so the swap is refused, the link is named, and the checkout is left
// as it was. A Set that judges ~/.abcd by path and writes by path never
// reaches the walk at all, which the test reports as a race it could not stage.
func TestSetWritesNoIndexThroughAnAbcdHomeSwappedForALink(t *testing.T) {
	const envName = "ABCD_TEST_ROUTING_SWAP_TOKEN"
	t.Setenv(envName, secretValue)
	for _, c := range []struct {
		name string
		// swapAt is the vetting of ~/.abcd (1-based) that swaps it: the first
		// is Set's own walk, before the lock; the second is the index read
		// under the lock, once an index is there to read.
		swapAt    int
		seedIndex bool
	}{
		{"before the lock", 1, false},
		{"under the lock", 2, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			abcd := filepath.Join(home, ".abcd")
			dotfiles := filepath.Join(home, "dotfiles", "abcd")
			for _, dir := range []string{abcd, dotfiles} {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if c.seedIndex {
				if err := os.WriteFile(filepath.Join(abcd, IndexFileName), []byte(`{"other":{"home":"keychain"}}`+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			seen, swapped := 0, false
			t.Cleanup(fsutil.SwapHomeScopeVettedForTest(func(dir string) {
				if swapped || dir != abcd {
					return
				}
				if seen++; seen < c.swapAt {
					return
				}
				swapped = true
				if err := os.Rename(dir, filepath.Join(home, "moved-aside")); err != nil {
					t.Fatalf("swap: %v", err)
				}
				if err := os.Symlink(dotfiles, dir); err != nil {
					t.Fatalf("swap: %v", err)
				}
			}))
			changed, err := Set(home, "svc", Choice{Home: HomeExternal, Pointer: Pointer{Env: envName}})
			if !swapped {
				t.Fatalf("Set never reached ~/.abcd through the walk that judges it (%d vetting(s) seen), so the race could not be staged: changed %v, err %v", seen, changed, err)
			}
			if err == nil || changed || !strings.Contains(err.Error(), "~/.abcd is a symlink") {
				t.Errorf("Set must refuse a ~/.abcd swapped for a link, naming it: changed %v, err %v", changed, err)
			}
			entries, rerr := os.ReadDir(dotfiles)
			if rerr != nil {
				t.Fatal(rerr)
			}
			if len(entries) != 0 {
				t.Fatalf("Set wrote %d file(s) through the swapped link, first %q", len(entries), entries[0].Name())
			}
		})
	}
}

// TestAHomeThatIsItselfALinkIntoACheckoutIsJudgedWhereItLeads is
// iss-2609290259108077 (review-integ14 LOW (a)): the home directory is itself
// a symlink into a git checkout (~ -> <repo>/home). The home is never refused
// for being a link, but the working-tree check judges where it leads as well
// as its lexical path, so the abcd home is refused with nothing written in the
// checkout and the keychain untouched, and a pointer at a file under that home
// is not read.
func TestAHomeThatIsItselfALinkIntoACheckoutIsJudgedWhereItLeads(t *testing.T) {
	keychain := withFakeKeychain(t, "security")
	base := t.TempDir()
	repo := filepath.Join(base, "checkout")
	real := filepath.Join(repo, "home")
	for _, dir := range []string{filepath.Join(repo, ".git"), filepath.Join(real, ".config")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	home := filepath.Join(base, "account")
	if err := os.Symlink(real, home); err != nil {
		t.Fatal(err)
	}

	changed, err := Set(home, "svc", Choice{Home: HomeABCD, Value: secretValue})
	if err == nil || changed {
		t.Fatalf("Set kept the value under a home that leads into a checkout: changed %v, err %v", changed, err)
	}
	if !strings.Contains(err.Error(), "git working tree") {
		t.Errorf("the refusal must name the working tree: %v", err)
	}
	if strings.Contains(err.Error(), secretValue) {
		t.Fatal("the refusal echoes the value")
	}
	if _, serr := os.Lstat(filepath.Join(real, ".abcd", StoreFileName)); !os.IsNotExist(serr) {
		t.Fatalf("%s was written inside the checkout: %v", StoreFileName, serr)
	}
	assertNowhere(t, repo, secretValue)
	if entries, _ := os.ReadDir(keychain); len(entries) != 0 {
		t.Fatalf("the keychain was written: %d item(s)", len(entries))
	}

	if err := os.WriteFile(filepath.Join(real, ".config", "tool.json"), []byte(`{"auth":{"token":"`+secretValue+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	v, err := resolvePointer(home, "svc", Pointer{File: "~/.config/tool.json", Field: "auth.token"})
	if err == nil || v != "" {
		t.Fatalf("a pointer under a home that leads into a checkout was read: value %q, err %v", v, err)
	}
	if !strings.Contains(err.Error(), "git working tree") {
		t.Errorf("the pointer's refusal must name the working tree: %v", err)
	}
}

// TestAPointerThroughALinkIsRefusedInAPointersWords is review-integ14 LOW (c):
// the file a pointer names is the tool's, not abcd's, so the refusal of a
// pointer whose directory passes through a symlink says what a pointer needs
// (directories that are real, or the environment-variable pointer) and never
// tells the person to keep abcd's files there. The link here leads outside any
// repository: the rule is the link, wherever it leads.
func TestAPointerThroughALinkIsRefusedInAPointersWords(t *testing.T) {
	home := t.TempDir()
	elsewhere := filepath.Join(t.TempDir(), "cfg")
	if err := os.MkdirAll(elsewhere, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(elsewhere, "tool.json"), []byte(`{"auth":{"token":"`+secretValue+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(home, ".config")); err != nil {
		t.Fatal(err)
	}
	p := Pointer{File: "~/.config/tool.json", Field: "auth.token"}
	_, rerr := resolvePointer(home, "svc", p)
	_, serr := Set(home, "svc", Choice{Home: HomeExternal, Pointer: p})
	for what, err := range map[string]error{"resolve": rerr, "Set": serr} {
		if err == nil {
			t.Fatalf("%s: a pointer through a symlinked ~/.config was followed", what)
		}
		msg := err.Error()
		for _, want := range []string{"~/.config is a symlink", "a pointer", "real directories", "environment-variable pointer"} {
			if !strings.Contains(msg, want) {
				t.Errorf("%s: the refusal lacks %q: %s", what, want, msg)
			}
		}
		if strings.Contains(msg, "abcd's files") {
			t.Errorf("%s: the refusal calls the tool's file abcd's: %s", what, msg)
		}
		if strings.Contains(msg, secretValue) {
			t.Fatalf("%s: the refusal echoes the value", what)
		}
	}
}
