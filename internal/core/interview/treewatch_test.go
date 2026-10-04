package interview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// TestARoleWritingWhereCodeOrAPushRunsFromStopsTheInterview: the guard
// watches what git status alone does not show. A role that plants a hook,
// edits the repository's git configuration or its info files, writes into
// the local tier (a push receipt, the handover), writes a gitignored file or
// one inside a gitignored directory, rewrites an ignored file at the same
// size, or writes into a hooks directory core.hooksPath names outside the
// tree, is stopped after that dispatch with the path named, nothing more
// drawn, and the answer given before it recorded.
func TestARoleWritingWhereCodeOrAPushRunsFromStopsTheInterview(t *testing.T) {
	outside := t.TempDir()
	for _, c := range []struct {
		name string
		// write is the path the role writes and the content it writes,
		// given the repository; named is what the refusal must name.
		write func(r *writtenRun) (path, content, named string)
		setup func(r *writtenRun)
	}{
		{name: "a planted pre-commit hook", write: func(*writtenRun) (string, string, string) {
			return ".git/hooks/pre-commit", "#!/bin/sh\nexit 0\n", ".git/hooks/pre-commit"
		}},
		{name: "an alias in the git configuration", write: func(r *writtenRun) (string, string, string) {
			b, err := os.ReadFile(filepath.Join(r.repo, ".git", "config"))
			if err != nil {
				t.Fatal(err)
			}
			return ".git/config", string(b) + "[alias]\n\tst = !sh -c true\n", ".git/config"
		}},
		{name: "an attributes file under info", write: func(*writtenRun) (string, string, string) {
			return ".git/info/attributes", "* filter=planted\n", ".git/info/attributes"
		}},
		{name: "a push receipt in the local tier", write: func(*writtenRun) (string, string, string) {
			p := ".abcd/.work.local/preflight-receipts/0123456789abcdef0123456789abcdef01234567"
			return p, "ok\n", p
		}},
		{name: "a gitignored file", setup: ignoring("*.log\n"), write: func(*writtenRun) (string, string, string) {
			return "build.log", "planted\n", "build.log"
		}},
		{name: "a file inside a gitignored directory", setup: ignoring("out/\n"), write: func(*writtenRun) (string, string, string) {
			return "out/deep/tool.sh", "#!/bin/sh\n", "out/deep/tool.sh"
		}},
		{name: "an ignored file rewritten at the same size", setup: func(r *writtenRun) {
			ignoring(".abcd/.work.local/\n")(r)
			r.git.Write(".abcd/.work.local/NEXT.md", "aaaa\n")
		}, write: func(*writtenRun) (string, string, string) {
			return ".abcd/.work.local/NEXT.md", "bbbb\n", ".abcd/.work.local/NEXT.md"
		}},
		{name: "a hook under core.hooksPath outside the tree", setup: func(r *writtenRun) {
			r.git.Git("config", "core.hooksPath", outside)
		}, write: func(*writtenRun) (string, string, string) {
			p := filepath.ToSlash(filepath.Join(outside, "pre-push"))
			return p, "#!/bin/sh\nexit 0\n", "/pre-push"
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			if c.setup != nil {
				c.setup(r)
			}
			p, body, named := c.write(r)
			stubAlso(t, script, 2, map[string]string{p: body})
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) {
				t.Fatalf("err = %v, want the interview stopped on %s", err, named)
			}
			if !slices.ContainsFunc(uc.Paths, func(got string) bool { return strings.HasSuffix(got, named) }) || !strings.Contains(err.Error(), named) {
				t.Fatalf("the refusal does not name %s: %v", named, err)
			}
			if len(r.asked) != 1 || len(r.finished) != 0 {
				t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the change", len(r.asked), len(r.finished))
			}
			if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
				t.Fatalf("record %+v", rec)
			}
		})
	}
}

// ignoring commits a .gitignore holding body.
func ignoring(body string) func(*writtenRun) {
	return func(r *writtenRun) {
		r.git.Write(".gitignore", body)
		r.git.Commit("ignore")
	}
}

// TestAHookPlantedThroughALinkedWorktreeStopsTheInterview: in a linked
// worktree the hooks and the configuration live in the repository's common
// git directory, outside the worktree, and are watched there.
func TestAHookPlantedThroughALinkedWorktreeStopsTheInterview(t *testing.T) {
	script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
	r := newWrittenRun(t, routedToClaude)
	r.git.Commit("base")
	lane := filepath.Join(t.TempDir(), "lane")
	r.git.Git("worktree", "add", "-q", "-b", "lane", lane)
	if err := os.MkdirAll(filepath.Join(lane, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	r.w.Repo = lane
	stubAlso(t, script, 2, map[string]string{filepath.ToSlash(filepath.Join(r.repo, ".git", "hooks", "post-checkout")): "#!/bin/sh\n"})
	res, err := r.w.Run(context.Background())
	var uc *UnexpectedChangesError
	if !errors.As(err, &uc) || len(uc.Paths) != 1 || !strings.HasSuffix(uc.Paths[0], "/.git/hooks/post-checkout") {
		t.Fatalf("err = %v, want the common directory's planted hook named", err)
	}
	if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
		t.Fatalf("record %+v", rec)
	}
}

// TestALinkedWorktreesGitFileIsWatched: a linked worktree's .git file names
// the git directory every git command there uses, so a role rewriting it is
// stopped too.
func TestALinkedWorktreesGitFileIsWatched(t *testing.T) {
	script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
	r := newWrittenRun(t, routedToClaude)
	r.git.Commit("base")
	lane := filepath.Join(t.TempDir(), "lane")
	r.git.Git("worktree", "add", "-q", "-b", "lane", lane)
	if err := os.MkdirAll(filepath.Join(lane, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	r.w.Repo = lane
	b, err := os.ReadFile(filepath.Join(lane, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	stubAlso(t, script, 2, map[string]string{".git": string(b) + "\n"})
	_, err = r.w.Run(context.Background())
	var uc *UnexpectedChangesError
	if !errors.As(err, &uc) || !slices.Equal(uc.Paths, []string{".git"}) {
		t.Fatalf("err = %v, want the worktree's .git file named", err)
	}
}

// TestARoleWritingOnlyItsTurnDirectoryIsNotStopped: the run's own turn
// directory is the one place in the local tier the role must write, so a
// role writing its receipt and a note beside it there runs to its outcome.
func TestARoleWritingOnlyItsTurnDirectoryIsNotStopped(t *testing.T) {
	script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
	r := newWrittenRun(t, routedToClaude)
	ignoring(".abcd/.work.local/\n")(r)
	stubAlso(t, script, 1, map[string]string{"turns/scratch.md": "the role's note\n"})
	res, err := r.w.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Changed) != 0 || len(r.asked) != 1 || len(r.finished) != 1 {
		t.Fatalf("changed %v, asked %d, finished %d", res.Changed, len(r.asked), len(r.finished))
	}
}

// TestThePersonsOwnEditsToWatchedPlacesAreNotTheRoles: the tree, the git
// directory and the ignored paths are read around each dispatch, so what the
// person changes while a question is put to them (a hook, the git
// configuration, an ignored file, the handover) is not laid at the role's
// door.
func TestThePersonsOwnEditsToWatchedPlacesAreNotTheRoles(t *testing.T) {
	stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
	r := newWrittenRun(t, routedToClaude)
	ignoring("*.log\n")(r)
	answer := r.w.Answer
	r.w.Answer = func(a question.Ask) ([]Reply, error) {
		for rel, body := range map[string]string{
			".git/hooks/pre-commit": "#!/bin/sh\n", "debug.log": "the person's log\n", ".abcd/.work.local/NEXT.md": "handover\n",
		} {
			r.git.Write(rel, body)
		}
		r.git.Git("config", "user.name", "Person")
		return answer(a)
	}
	res, err := r.w.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Changed) != 0 || len(r.finished) != 1 {
		t.Fatalf("changed %v, finished %d", res.Changed, len(r.finished))
	}
}
