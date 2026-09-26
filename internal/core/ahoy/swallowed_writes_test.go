package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/identity"
)

// notesCarryAll reports whether any note carries every one of want.
func notesCarryAll(notes []string, want ...string) bool {
	for _, n := range notes {
		all := true
		for _, w := range want {
			if !strings.Contains(n, w) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// TestIdentityPinFailureIsNoted is iss-2609260057127611's first write: the pin
// step dropped a failed identity.WritePin, and an unset git identity, without a
// word, so a run that recorded no identity said nothing about it.
func TestIdentityPinFailureIsNoted(t *testing.T) {
	for name, gitName := range map[string]string{
		"write refused":  `Quote "Person"`, // WritePin refuses a double-quote
		"identity unset": "",
	} {
		t.Run(name, func(t *testing.T) {
			email := "someone@example.com"
			if gitName == "" {
				email = ""
			}
			dir := idGitRepo(t, gitName, email)
			a := &applyCtx{cwd: dir, approved: map[GapCategory]bool{ConfigChange: true}, gapPresent: map[string]bool{OptionalPinGapID: true}}
			a.stepIdentityPin()
			if _, ok, _ := identity.LoadPin(dir); ok {
				t.Fatal("precondition: no pin is written")
			}
			if !notesCarryAll(a.notes, identity.PinRelPath) {
				t.Errorf("no note says the identity was not recorded; notes: %v", a.notes)
			}
		})
	}
}

// TestGitignoreBlockFailureIsNoted is the second: the visibility step dropped
// applyVisibilityBlock's refusal, so a symlinked .gitignore left the run with
// no fence and no reason.
func TestGitignoreBlockFailureIsNoted(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(t.TempDir(), "elsewhere"), filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatal(err)
	}
	a := &applyCtx{cwd: dir, approved: map[GapCategory]bool{ConfigChange: true}}
	a.stepVisibility(&InstallConfig{Visibility: "private"})
	if !notesCarryAll(a.notes, ".gitignore", "symlink") {
		t.Errorf("no note says the .gitignore block was not written, and why; notes: %v", a.notes)
	}
}

// TestMarkerBlockFailureIsNoted is the third: the marker step dropped a failed
// placement or retraction of abcd's block in a conventions file.
func TestMarkerBlockFailureIsNoted(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if err := os.Symlink(filepath.Join(t.TempDir(), name), filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	a := &applyCtx{cwd: dir, approved: map[GapCategory]bool{PluginOwned: true}, markerRetract: []string{"AGENTS.md"}}
	a.stepMarker(&InstallConfig{DocsTarget: "claude_md"})
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if !notesCarryAll(a.notes, name, "symlink") {
			t.Errorf("no note says abcd's block in %s was left alone, and why; notes: %v", name, a.notes)
		}
	}
}

// TestSessionStoreFailureIsNoted is the history step's share of the rule the
// brief states for every install write: a session store abcd could not create
// is a note naming the store and the reason, never a silent omission. Both
// halves are driven — the store's directory refused (a file where ~/.abcd
// belongs) and a home directory the process cannot name at all.
func TestSessionStoreFailureIsNoted(t *testing.T) {
	t.Run("the store cannot be created", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		if err := os.WriteFile(filepath.Join(home, ".abcd"), []byte("not a directory\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		a := &applyCtx{cwd: t.TempDir(), approved: map[GapCategory]bool{SafeAutocreate: true}}
		a.stepHistory()
		if !notesCarryAll(a.notes, "session store", "not a directory") {
			t.Errorf("no note says the session store was not created, and why; notes: %v", a.notes)
		}
	})
	t.Run("the transcript store cannot be created", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".abcd", "transcripts"), []byte("not a directory\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		a := &applyCtx{cwd: t.TempDir(), approved: map[GapCategory]bool{SafeAutocreate: true}}
		a.det.RepoIdentity.RootSHA = strings.Repeat("a", 40)
		a.stepHistory()
		if !notesCarryAll(a.notes, "session store", "transcripts", "not a real directory") {
			t.Errorf("no note says the transcript store was not created, and why; notes: %v", a.notes)
		}
	})
	t.Run("the home directory is unknown", func(t *testing.T) {
		t.Setenv("HOME", "")
		a := &applyCtx{cwd: t.TempDir(), approved: map[GapCategory]bool{SafeAutocreate: true}}
		a.stepHistory()
		if !notesCarryAll(a.notes, "session store", "HOME") {
			t.Errorf("no note says the session store was not created, and why; notes: %v", a.notes)
		}
	})
}
