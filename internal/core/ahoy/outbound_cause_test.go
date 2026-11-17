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

	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
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
		// The errno's own words, which differ between systems ("invalid
		// cross-device link" on Linux, "cross-device link" on macOS).
		{&fs.PathError{Op: "open", Path: secret, Err: syscall.EACCES}, "open: " + syscall.EACCES.Error()},
		{fmt.Errorf("opening %s: %w", secret, &fs.PathError{Op: "mkdirat", Path: secret, Err: syscall.ENOTDIR}), "mkdirat: " + syscall.ENOTDIR.Error()},
		{&os.LinkError{Op: "rename", Old: secret, New: secret, Err: syscall.EXDEV}, "rename: " + syscall.EXDEV.Error()},
		{fmt.Errorf("%w: %s: %w", gitutil.ErrNoRootAnswer, secret, gitutil.ErrLazyFetchFloor), gitutil.ErrLazyFetchFloor.Error()},
		{fmt.Errorf("%w: %s: asking: %w", gitutil.ErrNoRootAnswer, secret, gitutil.ErrGitVersionUnread), gitutil.ErrGitVersionUnread.Error()},
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
	for _, leak := range []string{"could not lock", "config.lock", repo, displayPath(repo)} {
		if strings.Contains(all, leak) {
			t.Errorf("the refusal carries %q:\n%s", leak, all)
		}
	}
}

// TestAbcdBinaryWriteRefusalNamesThePathToSet: when git refuses to write
// abcd.binary (a lock a crashed git left), the refusal says why in words and
// names the command to run in full, the verified copy's path included, since
// nothing else in the run's output may name it.
func TestAbcdBinaryWriteRefusalNamesThePathToSet(t *testing.T) {
	setupHermetic(t)
	target := binTarget()
	repo := gittest.NewRepo(t)
	if err := os.WriteFile(filepath.Join(repo.Root(), ".git", "config.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	all := notesJoined(res.Notes)
	if !strings.Contains(all, "git config could not write it (git exited with status") {
		t.Fatalf("want the write refusal in words:\n%s", all)
	}
	if !strings.Contains(all, "git config --local abcd.binary "+displayPath(target)+" shows git's reason") {
		t.Errorf("the refusal does not name the command to run in full:\n%s", all)
	}
	if strings.Contains(all, "could not lock") {
		t.Errorf("the refusal carries git's message:\n%s", all)
	}
}

// brokenHistoryRepo is a clone whose branch names an object git does not
// hold, so git refuses the root-commit read with a message of its own.
func brokenHistoryRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Commit("first")
	ref := strings.TrimSpace(repo.Git("symbolic-ref", "HEAD"))
	missing := strings.Repeat("d", len(strings.TrimSpace(repo.Git("rev-parse", "HEAD"))))
	if err := os.WriteFile(filepath.Join(repo.Root(), ".git", filepath.FromSlash(ref)), []byte(missing+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestRootCommitRefusalsKeepGitsMessageOut (iss-2611162118486104): where git
// gives no root commit, the install's refusal and the remote verbs' refusal say
// why in words and name the command that shows git's reason, never git's
// message, which names objects and paths, nor the checkout's own location.
func TestRootCommitRefusalsKeepGitsMessageOut(t *testing.T) {
	setupHermetic(t)
	check := func(t *testing.T, where, all, root string) {
		t.Helper()
		if !strings.Contains(all, "git gave no root commit") || !strings.Contains(all, rootCommitCommand+" shows git's reason") {
			t.Fatalf("%s: want the refusal in words with the command:\n%s", where, all)
		}
		for _, leak := range []string{"fatal", "bad object", strings.Repeat("d", 12), root, displayPath(root)} {
			if strings.Contains(all, leak) {
				t.Errorf("%s: the refusal carries %q:\n%s", where, leak, all)
			}
		}
	}
	t.Run("install", func(t *testing.T) {
		repo := brokenHistoryRepo(t)
		_, err := Install(repo.Root(), planOpts(), RefusingPrompter{})
		if err == nil {
			t.Fatal("an install that cannot tell whether abcd manages the clone went ahead")
		}
		if !errors.Is(err, gitutil.ErrNoRootAnswer) {
			t.Errorf("the refusal lost its cause from the chain: %v", err)
		}
		check(t, "install", err.Error(), repo.Root())
	})
	t.Run("install with --adopt", func(t *testing.T) {
		repo := brokenHistoryRepo(t)
		o := planOpts()
		adopt := true
		o.Adopt = &adopt
		o.Yes = true
		res, err := Install(repo.Root(), o, RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		check(t, "install with --adopt", notesJoined(res.Notes), repo.Root())
	})
	t.Run("remote", func(t *testing.T) {
		repo := brokenHistoryRepo(t)
		res, _ := RemoteRead(repo.Root())
		check(t, "remote", strings.Join(res.Notes, "\n"), repo.Root())
	})
}

// TestIdentityProposalRefusalKeepsGitsMessageOut (iss-2611162118486104): the
// identity step reads the root commit to find the person's record; when git
// gives none, the refusal says so in words and names the command to run.
func TestIdentityProposalRefusalKeepsGitsMessageOut(t *testing.T) {
	setupHermetic(t)
	repo := brokenHistoryRepo(t)
	p := &identityPrompter{terminal: true, answer: true}
	a := establishCtx(repo.Root(), p, MismatchGapID)
	a.stepGitIdentity()
	notes := strings.Join(a.notes, "\n")
	if !strings.Contains(notes, "the identity to propose could not be read: git gave no root commit (git exited with status") ||
		!strings.Contains(notes, rootCommitCommand+" shows git's reason") {
		t.Fatalf("want the refusal in words with the command:\n%s", notes)
	}
	for _, leak := range []string{"fatal", "bad object", repo.Root(), displayPath(repo.Root())} {
		if strings.Contains(notes, leak) {
			t.Errorf("the refusal carries %q:\n%s", leak, notes)
		}
	}
}

// TestRootCommitCauseNamesTheCommandOnlyWhenGitRefused: git's own refusal
// names the command that shows its reason; abcd's refusal of a git below the
// lazy-fetch floor is said in full, with no command that would answer in the
// person's shell, where lazy fetch is on.
func TestRootCommitCauseNamesTheCommandOnlyWhenGitRefused(t *testing.T) {
	const secret = "/Users/private-name/elsewhere"
	exit := exec.Command("sh", "-c", "exit 128").Run()
	got := rootCommitCause(fmt.Errorf("%w: %s: %w (stderr: %q)", gitutil.ErrNoRootAnswer, secret, exit, "fatal: bad object"))
	if got != "git exited with status 128; "+rootCommitCommand+" shows git's reason" {
		t.Errorf("git's refusal: %q", got)
	}
	got = rootCommitCause(fmt.Errorf("%w: %s: %w", gitutil.ErrNoRootAnswer, secret, gitutil.ErrLazyFetchFloor))
	if got != gitutil.ErrLazyFetchFloor.Error() || strings.Contains(got, rootCommitCommand) {
		t.Errorf("abcd's refusal: %q", got)
	}
}

// TestIdentityReadCauseKeepsAbcdsOwnSentence: a failure that is not git's
// root-commit read is abcd's own sentence about the record or the pin, kept.
func TestIdentityReadCauseKeepsAbcdsOwnSentence(t *testing.T) {
	own := fmt.Errorf("%w: the record is not valid JSON", identity.ErrRecordRefused)
	if got := identityReadCause(own); !strings.Contains(got, "is not valid JSON") || strings.Contains(got, rootCommitCommand) {
		t.Errorf("abcd's own sentence was not kept: %q", got)
	}
	root := fmt.Errorf("%w: /Users/private-name/x: %w", gitutil.ErrNoRootAnswer, exec.Command("sh", "-c", "exit 128").Run())
	if got := identityReadCause(root); strings.Contains(got, "private-name") || !strings.HasPrefix(got, "git gave no root commit (git exited with status 128") {
		t.Errorf("git's root-commit failure: %q", got)
	}
}
