package ahoy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestOutboundNotesKeepGitsReasonAndPathsOut (iss-2610152204588951): a note
// the outbound install gives when git cannot read this clone's settings says
// what failed in words, and never carries git's message, which names the
// file git could not read, here the person's own git config outside the
// working tree.
func TestOutboundNotesKeepGitsReasonAndPathsOut(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := gittest.NewRepo(t)
	global := filepath.Join(home, ".gitconfig")
	if err := os.WriteFile(global, []byte("[core\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The host's own setting would point git past the broken file.
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	note := planOutboundShim(repo.Root()).note
	if note == "" {
		t.Fatal("a clone whose git config git cannot read must draw a note")
	}
	for _, leak := range []string{home, filepath.Base(home), "bad config", ".gitconfig"} {
		if strings.Contains(note, leak) {
			t.Errorf("the note carries %q:\n%s", leak, note)
		}
	}
	if !strings.Contains(note, "git exited with status") {
		t.Errorf("the note must say git failed, in words:\n%s", note)
	}
}

// TestCauseInWordsNamesNoPath: a file error is told by what was done and why,
// and git's failure by its exit status, never by a path or git's own message.
func TestCauseInWordsNamesNoPath(t *testing.T) {
	const secret = "/Users/private-name/elsewhere/.git"
	exit := exec.Command("sh", "-c", "exit 3").Run()
	for _, c := range []struct {
		err  error
		want string
	}{
		{&fs.PathError{Op: "open", Path: secret, Err: syscall.EACCES}, "open: permission denied"},
		{fmt.Errorf("opening %s: %w", secret, &fs.PathError{Op: "mkdirat", Path: secret, Err: syscall.ENOTDIR}), "mkdirat: not a directory"},
		{&os.LinkError{Op: "rename", Old: secret, New: secret, Err: syscall.EXDEV}, "rename: cross-device link"},
		{fmt.Errorf("%w (stderr: %q)", exit, "fatal: bad config line 1 in file "+secret), "git exited with status 3"},
		{fmt.Errorf("git: %w", exec.ErrNotFound), "git was not found"},
		{errors.New("somewhere at " + secret), "an error abcd keeps out of this note"},
	} {
		got := causeInWords(c.err)
		if !strings.HasPrefix(got, c.want) || strings.Contains(got, "private-name") {
			t.Errorf("causeInWords(%v) = %q, want it to begin %q and name no path", c.err, got, c.want)
		}
	}
}

// TestArmRefusalKeepsGitsMessageOut: when git refuses to set core.hooksPath
// after a yes (here a lock a crashed git left), the refusal and what it asks
// the person say why in words, never git's message, which names the config
// file by its full path, outside the working tree for a linked worktree.
func TestArmRefusalKeepsGitsMessageOut(t *testing.T) {
	repo := hooksRepo(t)
	lock := filepath.Join(repo, ".git", "config.lock")
	p := &planPrompter{choice: PlanProceed, confirm: func(q string) bool {
		if !isArmQuestion(q) {
			return false
		}
		if err := os.WriteFile(lock, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		return true
	}}
	res, err := Install(repo, planOpts(), p)
	if err != nil {
		t.Fatal(err)
	}
	all := notesJoined(res.Notes) + "\n" + needsJoined(res.Report)
	if !strings.Contains(all, "core.hooksPath was not set: git exited with status") {
		t.Fatalf("want the refusal in words:\n%s", all)
	}
	for _, leak := range []string{"could not lock", "config.lock", filepath.Base(repo)} {
		if strings.Contains(all, leak) {
			t.Errorf("the refusal carries %q:\n%s", leak, all)
		}
	}
}
