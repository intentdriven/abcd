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
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[core\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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

// TestOutboundCauseNamesNoPath: a file error is told by what was done and why,
// and git's failure by its exit status, never by a path or git's own message.
func TestOutboundCauseNamesNoPath(t *testing.T) {
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
		got := outboundCause(c.err)
		if !strings.HasPrefix(got, c.want) || strings.Contains(got, "private-name") {
			t.Errorf("outboundCause(%v) = %q, want it to begin %q and name no path", c.err, got, c.want)
		}
	}
}
