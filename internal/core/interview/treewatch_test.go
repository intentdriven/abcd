package interview

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestARoleWritingWhereCodeOrAPushRunsFromStopsTheInterview: the guard
// watches what git status alone does not show. A role that plants a hook,
// edits the repository's git configuration or its info files, moves HEAD,
// adds a ref or rewrites the packed refs, plants a submodule's hook or edits
// its configuration, writes into the local tier (a push receipt, the
// handover), writes a gitignored file or one inside a gitignored directory,
// rewrites an ignored file at the same size, writes into a hooks directory
// core.hooksPath names outside the tree (through a link included), or writes
// the host's local settings, is stopped after that dispatch with the path
// named, nothing more drawn, and the answer given before it recorded. The
// two exemptions are held to their exact shape: a scheduler lock anywhere
// but the root's .claude, and a file under a directory named .DS_Store, are
// watched.
func TestARoleWritingWhereCodeOrAPushRunsFromStopsTheInterview(t *testing.T) {
	outside := t.TempDir()
	// linkedHooks is a hooks directory reached through a link, as a
	// dotfiles-managed ~/.githooks is; linkedFromTree is one an in-tree link
	// points at.
	linkedHooks, linkedFromTree, linkedFromRealTree := t.TempDir(), t.TempDir(), t.TempDir()
	linkTo := func(target, link string) {
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
	}
	// gitDirOf makes a submodule's git directory under .git/modules.
	gitDirOf := func(name string) func(r *writtenRun) {
		return func(r *writtenRun) {
			r.git.Write(".git/modules/"+name+"/HEAD", "ref: refs/heads/main\n")
			r.git.Write(".git/modules/"+name+"/config", "[core]\n\tbare = false\n")
		}
	}
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
		{name: "a hook under a core.hooksPath that is a link to a directory", setup: func(r *writtenRun) {
			link := filepath.Join(t.TempDir(), "githooks")
			linkTo(linkedHooks, link)
			r.git.Git("config", "core.hooksPath", link)
		}, write: func(*writtenRun) (string, string, string) {
			return filepath.ToSlash(filepath.Join(linkedHooks, "pre-commit")), "#!/bin/sh\nexit 0\n", "/pre-commit"
		}},
		{name: "a hook under an in-tree core.hooksPath that links outside the tree", setup: func(r *writtenRun) {
			linkTo(linkedFromTree, filepath.Join(r.repo, "tools", "hooks"))
			r.git.Git("config", "core.hooksPath", "tools/hooks")
		}, write: func(*writtenRun) (string, string, string) {
			return filepath.ToSlash(filepath.Join(linkedFromTree, "post-checkout")), "#!/bin/sh\nexit 0\n", "/post-checkout"
		}},
		{name: "a hook under an in-tree core.hooksPath that links outside, spelled by the tree's real path", setup: func(r *writtenRun) {
			linkTo(linkedFromRealTree, filepath.Join(r.repo, "tools", "hooks"))
			real, err := filepath.EvalSymlinks(r.repo)
			if err != nil {
				t.Fatal(err)
			}
			r.git.Git("config", "core.hooksPath", filepath.Join(real, "tools", "hooks"))
		}, write: func(*writtenRun) (string, string, string) {
			return filepath.ToSlash(filepath.Join(linkedFromRealTree, "pre-rebase")), "#!/bin/sh\nexit 0\n", "/pre-rebase"
		}},
		{name: "HEAD pointed at another branch", setup: func(r *writtenRun) { r.git.Commit("base") }, write: func(*writtenRun) (string, string, string) {
			return ".git/HEAD", "ref: refs/heads/other\n", ".git/HEAD"
		}},
		{name: "a new ref", setup: func(r *writtenRun) { r.git.Commit("base") }, write: func(r *writtenRun) (string, string, string) {
			return ".git/refs/heads/planted", r.git.Git("rev-parse", "HEAD") + "\n", ".git/refs/heads/planted"
		}},
		{name: "the packed refs", setup: func(r *writtenRun) { r.git.Commit("base") }, write: func(r *writtenRun) (string, string, string) {
			return ".git/packed-refs", r.git.Git("rev-parse", "HEAD") + " refs/heads/packed\n", ".git/packed-refs"
		}},
		{name: "a submodule's hook", setup: gitDirOf("sub"), write: func(*writtenRun) (string, string, string) {
			return ".git/modules/sub/hooks/pre-commit", "#!/bin/sh\n", ".git/modules/sub/hooks/pre-commit"
		}},
		{name: "a submodule's configuration", setup: gitDirOf("sub"), write: func(*writtenRun) (string, string, string) {
			return ".git/modules/sub/config", "[core]\n\tbare = false\n[alias]\n\tst = !sh -c true\n", ".git/modules/sub/config"
		}},
		{name: "the hook of a submodule whose name holds a slash", setup: gitDirOf("lib/sub"), write: func(*writtenRun) (string, string, string) {
			return ".git/modules/lib/sub/hooks/post-checkout", "#!/bin/sh\n", ".git/modules/lib/sub/hooks/post-checkout"
		}},
		{name: "the hook of a submodule nested in a submodule", setup: func(r *writtenRun) {
			gitDirOf("sub")(r)
			gitDirOf("sub/modules/inner")(r)
		}, write: func(*writtenRun) (string, string, string) {
			return ".git/modules/sub/modules/inner/hooks/pre-push", "#!/bin/sh\n", ".git/modules/sub/modules/inner/hooks/pre-push"
		}},
		{name: "the host's local settings, which can name hooks", write: func(*writtenRun) (string, string, string) {
			return ".claude/settings.local.json", `{"hooks":{}}` + "\n", ".claude/settings.local.json"
		}},
		{name: "a scheduler lock's name anywhere but the root's .claude", write: func(*writtenRun) (string, string, string) {
			return "sub/.claude/scheduled_tasks.lock", "1\n", "sub/.claude/scheduled_tasks.lock"
		}},
		{name: "a file inside a directory named like Finder's metadata", write: func(*writtenRun) (string, string, string) {
			return ".DS_Store/tool.sh", "#!/bin/sh\n", ".DS_Store/tool.sh"
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
			// What changed in the dispatch's window is named as that, not
			// as the role's own doing: abcd cannot tell who wrote it.
			if want := fmt.Sprintf("%d path(s) changed while the %s ran", len(uc.Paths), RoleReflectionComposer); !strings.Contains(err.Error(), want) {
				t.Fatalf("the refusal does not say %q: %v", want, err)
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

// TestALinkedWorktreesOwnHEADIsWatched: a linked worktree keeps its own HEAD
// in its own git directory, apart from the common one, and it decides what
// the next commit there extends, so a role moving it is stopped.
func TestALinkedWorktreesOwnHEADIsWatched(t *testing.T) {
	script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
	r := newWrittenRun(t, routedToClaude)
	r.git.Commit("base")
	lane := filepath.Join(t.TempDir(), "lane")
	r.git.Git("worktree", "add", "-q", "-b", "lane", lane)
	if err := os.MkdirAll(filepath.Join(lane, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	r.w.Repo = lane
	head := filepath.ToSlash(filepath.Join(r.repo, ".git", "worktrees", "lane", "HEAD"))
	stubAlso(t, script, 2, map[string]string{head: "ref: refs/heads/main\n"})
	_, err := r.w.Run(context.Background())
	var uc *UnexpectedChangesError
	if !errors.As(err, &uc) || len(uc.Paths) != 1 || !strings.HasSuffix(uc.Paths[0], "/.git/worktrees/lane/HEAD") {
		t.Fatalf("err = %v, want the worktree's own HEAD named", err)
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

// TestAnInTreeHooksDirectoryIsNamedOnce: a hooks directory core.hooksPath
// names inside the working tree is read through the status listing alone, so
// a hook the role changes there is named once, by its path in the tree,
// whether the value is relative or spells the tree through a link (and on a
// platform whose temporary root is itself a link, the real path and the
// repository's path are spelled differently either way).
func TestAnInTreeHooksDirectoryIsNamedOnce(t *testing.T) {
	for _, c := range []struct {
		name  string
		value func(r *writtenRun) string
	}{
		{name: "relative", value: func(*writtenRun) string { return ".githooks" }},
		{name: "through a link to the tree", value: func(r *writtenRun) string {
			via := filepath.Join(t.TempDir(), "via")
			if err := os.Symlink(r.repo, via); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(via, ".githooks")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			r.git.Write(".githooks/pre-commit", "#!/bin/sh\n")
			r.git.Commit("hooks")
			r.git.Git("config", "core.hooksPath", c.value(r))
			stubAlso(t, script, 2, map[string]string{".githooks/pre-commit": "#!/bin/sh\nexit 0\n"})
			_, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) || !slices.Equal(uc.Paths, []string{".githooks/pre-commit"}) {
				t.Fatalf("err = %v, want the in-tree hook named once", err)
			}
		})
	}
}

// TestFinderMetadataAndTheSchedulerLockAreNotTheRoles: two paths something
// else on the machine writes while a role runs, and that execute nothing, are
// left out of the reading: a file named .DS_Store anywhere (Finder's
// metadata) and the one lock file .claude/scheduled_tasks.lock at the
// repository's root (the host's scheduler). A role run during which they
// change runs to its outcome, whether git ignores them or not.
func TestFinderMetadataAndTheSchedulerLockAreNotTheRoles(t *testing.T) {
	for _, c := range []struct {
		name   string
		ignore string
	}{
		{name: "untracked"},
		{name: "ignored", ignore: ".DS_Store\n.claude/\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			if c.ignore != "" {
				ignoring(c.ignore)(r)
			}
			r.git.Write("docs/.DS_Store", "before\n")
			stubAlso(t, script, 1, map[string]string{
				".DS_Store":                    "finder\n",
				"docs/.DS_Store":               "finder, again\n",
				".git/hooks/.DS_Store":         "finder\n",
				".claude/scheduled_tasks.lock": "4242\n",
			})
			res, err := r.w.Run(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Changed) != 0 || len(r.finished) != 1 {
				t.Fatalf("changed %v, finished %d", res.Changed, len(r.finished))
			}
		})
	}
}

// TestATreePastTheBoundIsRefusedBeforeTheFirstDispatch: a tree holding more
// watched paths than one reading may hold is refused before any role runs,
// rather than read in part, naming the bound; a tree at the bound is read.
func TestATreePastTheBoundIsRefusedBeforeTheFirstDispatch(t *testing.T) {
	keep := maxWatched
	t.Cleanup(func() { maxWatched = keep })
	r := newWrittenRun(t, routedToClaude)
	for i := range 5 {
		r.git.Write(fmt.Sprintf("untracked-%d.txt", i), "x\n")
	}
	st, err := readTree(r.repo, "")
	if err != nil {
		t.Fatal(err)
	}
	maxWatched = len(st)
	if _, err := readTree(r.repo, ""); err != nil {
		t.Fatalf("a tree at the bound of %d is refused: %v", maxWatched, err)
	}
	maxWatched = len(st) - 1
	if _, err := readTree(r.repo, ""); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("more than %d paths", maxWatched)) {
		t.Fatalf("err = %v, want a tree past the bound of %d refused", err, maxWatched)
	}

	script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
	res, err := r.w.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("more than %d paths", maxWatched)) {
		t.Fatalf("err = %v, want the run refused past the bound", err)
	}
	if _, serr := os.Stat(filepath.Join(script, "turn-1.brief.seen.md")); !errors.Is(serr, os.ErrNotExist) || len(r.asked) != 0 || res.Record != "" {
		t.Fatalf("a role ran (%v), %d asked, record %q; nothing runs past the bound", serr, len(r.asked), res.Record)
	}
}

// TestAGitEntryReachedThroughALinkIsReadWhereItLeads: git follows a link
// standing in its own directory, so a hooks directory, an info directory, a
// configuration file or a single hook there that is a link (a
// dotfiles-managed one) is read where it leads, and a role writing there is
// stopped, the path named where it was written.
func TestAGitEntryReachedThroughALinkIsReadWhereItLeads(t *testing.T) {
	for _, c := range []struct {
		name string
		// link is the entry under .git made a link to a directory or a file
		// elsewhere; inDir, for a directory, is the file the role writes in it.
		link, inDir string
	}{
		{name: "the hooks directory", link: "hooks", inDir: "pre-commit"},
		{name: "the info directory", link: "info", inDir: "attributes"},
		{name: "the configuration", link: "config"},
		{name: "one hook", link: "hooks/pre-push"},
		{name: "a submodule's hooks directory", link: "modules/sub/hooks", inDir: "pre-commit"},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			r.git.Write(".git/modules/sub/HEAD", "ref: refs/heads/main\n")
			at := filepath.Join(r.repo, ".git", filepath.FromSlash(c.link))
			target := filepath.Join(t.TempDir(), "target")
			written, body := target, "#!/bin/sh\nexit 0\n"
			if c.inDir != "" {
				if err := os.Mkdir(target, 0o755); err != nil {
					t.Fatal(err)
				}
				written = filepath.Join(target, c.inDir)
			} else {
				before, err := os.ReadFile(at)
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, before, 0o755); err != nil {
					t.Fatal(err)
				}
				body = string(before) + "[alias]\n\tst = !sh -c true\n"
			}
			if err := os.RemoveAll(at); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, at); err != nil {
				t.Fatal(err)
			}
			stubAlso(t, script, 2, map[string]string{filepath.ToSlash(written): body})
			_, err := r.w.Run(context.Background())
			named := strings.TrimPrefix(written, filepath.Dir(target))
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) || !slices.ContainsFunc(uc.Paths, func(p string) bool { return strings.HasSuffix(p, filepath.ToSlash(named)) }) {
				t.Fatalf("err = %v, want %s, written through the link at .git/%s, named", err, named, c.link)
			}
		})
	}
}

// TestAWorktreeRouteToAPushOrAHookStopsTheInterview: git keeps every linked
// worktree's metadata in the common directory's worktrees/, and the pre-push
// gate takes a receipt from the local tier of any worktree git lists. So a
// role registering a worktree (one whose gitdir names the role's own turn
// directory, holding a push receipt there), planting a receipt in a sibling
// worktree's local tier, or repointing a sibling's commondir or gitdir (a
// fake common directory whose configuration carries an alias), locking one
// or giving one its own configuration, is stopped after that dispatch with
// the path named, nothing more drawn, and the answer given before it kept.
func TestAWorktreeRouteToAPushOrAHookStopsTheInterview(t *testing.T) {
	receipts := receiptsRel + "/"
	for _, c := range []struct {
		name string
		// write is what the role writes, given the repository, its HEAD and a
		// sibling worktree; named is every path the refusal must name.
		write func(r *writtenRun, head, lane string) (files map[string]string, named []string)
		// listed: git must list the turn directory as a worktree after the
		// run, so the probe is shown to be one git honours.
		listed bool
		// setup, when set, runs once the sibling worktree stands.
		setup func(r *writtenRun, lane string)
	}{
		{name: "a worktree registered at the turn directory, holding a receipt", write: func(r *writtenRun, head, _ string) (map[string]string, []string) {
			return map[string]string{
				".git/worktrees/planted/gitdir":    turnDirToken + "/.git\n",
				".git/worktrees/planted/HEAD":      head + "\n",
				".git/worktrees/planted/commondir": "../..\n",
				"turns/" + receipts + head:         "commit " + head + "\n",
			}, []string{".git/worktrees/planted/gitdir", ".git/worktrees/planted/HEAD", ".git/worktrees/planted/commondir", receipts + head}
		}, listed: true},
		{name: "a receipt in a sibling worktree's local tier", write: func(r *writtenRun, head, lane string) (map[string]string, []string) {
			p := filepath.ToSlash(filepath.Join(lane, filepath.FromSlash(receiptsRel), head))
			return map[string]string{p: "commit " + head + "\n"}, []string{"/" + receipts + head}
		}},
		{name: "a sibling worktree's commondir repointed", write: func(r *writtenRun, _, _ string) (map[string]string, []string) {
			fake := t.TempDir()
			return map[string]string{
				filepath.ToSlash(filepath.Join(fake, "HEAD")):   "ref: refs/heads/main\n",
				filepath.ToSlash(filepath.Join(fake, "config")): "[core]\n\trepositoryformatversion = 0\n[alias]\n\tst = !sh -c true\n",
				".git/worktrees/lane/commondir":                 fake + "\n",
			}, []string{".git/worktrees/lane/commondir"}
		}},
		{name: "a sibling worktree's gitdir repointed", write: func(r *writtenRun, _, _ string) (map[string]string, []string) {
			return map[string]string{".git/worktrees/lane/gitdir": turnDirToken + "/.git\n"}, []string{".git/worktrees/lane/gitdir"}
		}},
		{name: "a sibling worktree's own configuration", write: func(r *writtenRun, _, _ string) (map[string]string, []string) {
			return map[string]string{".git/worktrees/lane/config.worktree": "[alias]\n\tst = !sh -c true\n"}, []string{".git/worktrees/lane/config.worktree"}
		}},
		{name: "a sibling worktree locked", write: func(r *writtenRun, _, _ string) (map[string]string, []string) {
			return map[string]string{".git/worktrees/lane/locked": "kept\n"}, []string{".git/worktrees/lane/locked"}
		}},
		{name: "a worktree entry holding nothing git reads", write: func(r *writtenRun, _, _ string) (map[string]string, []string) {
			return map[string]string{".git/worktrees/planted/index": "x\n"}, []string{".git/worktrees/planted"}
		}},
		{name: "a worktree registered at a file's path", write: func(r *writtenRun, head, _ string) (map[string]string, []string) {
			return map[string]string{
				".git/worktrees/planted/gitdir":    filepath.ToSlash(filepath.Join(r.repo, "notes.txt", ".git")) + "\n",
				".git/worktrees/planted/HEAD":      head + "\n",
				".git/worktrees/planted/commondir": "../..\n",
			}, []string{".git/worktrees/planted/gitdir"}
		}, setup: func(r *writtenRun, _ string) { r.git.Write("notes.txt", "a file\n") }},
		{name: "the commondir of a worktree entry that is a link", write: func(r *writtenRun, _, _ string) (map[string]string, []string) {
			moved := filepath.Join(filepath.Dir(r.repo), "moved-lane-entry")
			return map[string]string{filepath.ToSlash(filepath.Join(moved, "commondir")): t.TempDir() + "\n"}, []string{"/moved-lane-entry/commondir"}
		}, setup: func(r *writtenRun, _ string) {
			entry := filepath.Join(r.repo, ".git", "worktrees", "lane")
			moved := filepath.Join(filepath.Dir(r.repo), "moved-lane-entry")
			if err := os.Rename(entry, moved); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(moved, entry); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "a sibling worktree's HEAD moved", write: func(r *writtenRun, _, _ string) (map[string]string, []string) {
			return map[string]string{".git/worktrees/lane/HEAD": "ref: refs/heads/main\n"}, []string{".git/worktrees/lane/HEAD"}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			r.git.Commit("base")
			lane := filepath.Join(t.TempDir(), "lane")
			r.git.Git("worktree", "add", "-q", "-b", "lane", lane)
			if c.setup != nil {
				c.setup(r, lane)
			}
			head := r.git.Git("rev-parse", "HEAD")
			files, named := c.write(r, head, lane)
			stubAlso(t, script, 2, files)
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) {
				t.Fatalf("err = %v, want the interview stopped on %v", err, named)
			}
			for _, n := range named {
				if !slices.ContainsFunc(uc.Paths, func(got string) bool { return strings.HasSuffix(got, n) }) || !strings.Contains(err.Error(), n) {
					t.Fatalf("the refusal does not name %s: %v", n, err)
				}
			}
			if len(r.asked) != 1 || len(r.finished) != 0 {
				t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the change", len(r.asked), len(r.finished))
			}
			if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
				t.Fatalf("record %+v", rec)
			}
			if c.listed {
				list := r.git.Git("worktree", "list", "--porcelain")
				if !strings.Contains(list, "/"+TurnsRel+"/") {
					t.Fatalf("git does not list the planted worktree, so the probe is not one git honours:\n%s", list)
				}
			}
		})
	}
}

// TestAWorktreeThePersonMakesBetweenDispatchesIsNotTheRoles: the worktrees
// and their local tiers are read again before each dispatch, so a worktree
// the person adds while a question is put to them, and a push receipt its
// own preflight mints there, are not laid at the role's door.
func TestAWorktreeThePersonMakesBetweenDispatchesIsNotTheRoles(t *testing.T) {
	stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubDone)
	r := newWrittenRun(t, routedToClaude)
	r.git.Commit("base")
	answer := r.w.Answer
	r.w.Answer = func(a question.Ask) ([]Reply, error) {
		lane := filepath.Join(t.TempDir(), "lane")
		r.git.Git("worktree", "add", "-q", "-b", "lane", lane)
		receipt := filepath.Join(lane, ".abcd", ".work.local", "preflight-receipts", r.git.Git("rev-parse", "HEAD"))
		if err := os.MkdirAll(filepath.Dir(receipt), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(receipt, []byte("commit\n"), 0o600); err != nil {
			t.Fatal(err)
		}
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

// TestALinkCycleUnderGitsOwnDirectoryIsReadOnce: a link in git's own
// directory is followed once and never a link found where it leads, so a
// hook that is a link to a directory holding a link back to itself is read
// and the reading ends.
func TestALinkCycleUnderGitsOwnDirectoryIsReadOnce(t *testing.T) {
	r := newWrittenRun(t, "")
	target := filepath.Join(t.TempDir(), "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(target, "b")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(r.repo, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(r.repo, ".git", "hooks", "a")); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	var st treeState
	go func() {
		var err error
		st, err = readTree(r.repo, "")
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the reading did not end on a link cycle under .git/hooks")
	}
	if _, ok := st[".git/hooks/a"]; !ok {
		t.Fatalf("the link itself was not read: %v", st)
	}
	if !slices.ContainsFunc(slices.Collect(maps.Keys(st)), func(k string) bool { return strings.HasSuffix(k, "/target/b") }) {
		t.Fatalf("what the link leads to was not read: %v", st)
	}
}

// TestAFollowedLinkIsBounded: what the links one reading follows lead to is
// read within a small budget of entries and bytes, shared by every link that
// reading follows, and a reading past it is refused promptly, naming the
// link, rather than walking toward the general bound. So a role planting a
// link to a large tree where the guard follows links (its own turn
// directory's push receipts with the turn directory registered as a
// worktree, a hook, the hooks directory, the worktrees directory or one
// worktree's entry) cannot slow a turn for minutes before the guard refuses.
func TestAFollowedLinkIsBounded(t *testing.T) {
	// within runs read and fails the test when it does not end within the
	// time bound: reading the large trees below whole takes minutes.
	within := func(t *testing.T, read func() error) error {
		t.Helper()
		done := make(chan error, 1)
		go func() { done <- read() }()
		select {
		case err := <-done:
			return err
		case <-time.After(20 * time.Second):
			t.Fatal("the reading did not end within 20s: a followed link was read past its budget")
			return nil
		}
	}
	// huge is a directory holding name, a sparse file far past the byte
	// budget: making it costs nothing, hashing it whole takes minutes.
	huge := func(t *testing.T, name string) string {
		t.Helper()
		dir := filepath.Join(t.TempDir(), "huge")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := errors.Join(f.Truncate(128<<30), f.Close()); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	// many is a directory holding n empty files.
	many := func(t *testing.T, n int) string {
		t.Helper()
		dir := filepath.Join(t.TempDir(), "many")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for i := range n {
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%05d", i)), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}
	link := func(t *testing.T, target, at string) {
		t.Helper()
		if err := os.RemoveAll(at); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, at); err != nil {
			t.Fatal(err)
		}
	}
	refused := func(t *testing.T, err error, named string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), "leads past what one reading follows links to") || !strings.Contains(err.Error(), named) {
			t.Fatalf("err = %v, want the reading refused past the followed-link budget, naming %s", err, named)
		}
	}

	t.Run("a turn directory's receipts link, the turn directory registered as a worktree", func(t *testing.T) {
		script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
		r := newWrittenRun(t, routedToClaude)
		r.git.Commit("base")
		head := r.git.Git("rev-parse", "HEAD")
		stubAlso(t, script, 2, map[string]string{
			".git/worktrees/planted/gitdir":    turnDirToken + "/.git\n",
			".git/worktrees/planted/HEAD":      head + "\n",
			".git/worktrees/planted/commondir": "../..\n",
			"turns/" + receiptsRel:             linkToken + huge(t, head),
		})
		var res WrittenResult
		err := within(t, func() (err error) {
			res, err = r.w.Run(context.Background())
			return err
		})
		refused(t, err, receiptsRel)
		if !strings.Contains(err.Error(), TurnsRel+"/") {
			t.Fatalf("err = %v, want the link in the turn directory named", err)
		}
		if len(r.asked) != 1 || len(r.finished) != 0 {
			t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the refusal", len(r.asked), len(r.finished))
		}
		if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
			t.Fatalf("record %+v", rec)
		}
	})

	for _, c := range []struct {
		name string
		// plant makes the links under the repository, returning the path the
		// refusal must name.
		plant func(t *testing.T, repo string) string
	}{
		{name: "a hook linked to a file past the byte budget", plant: func(t *testing.T, repo string) string {
			link(t, filepath.Join(huge(t, "pre-push"), "pre-push"), filepath.Join(repo, ".git", "hooks", "pre-push"))
			return ".git/hooks/pre-push"
		}},
		{name: "the hooks directory linked to a tree past the entry budget", plant: func(t *testing.T, repo string) string {
			link(t, many(t, maxFollowedEntries+1), filepath.Join(repo, ".git", "hooks"))
			return ".git/hooks"
		}},
		{name: "two hook links within the budget alone, past it together", plant: func(t *testing.T, repo string) string {
			link(t, many(t, maxFollowedEntries/2+1), filepath.Join(repo, ".git", "hooks", "a"))
			if _, err := readTree(repo, ""); err != nil {
				t.Fatalf("one link within the budget is refused: %v", err)
			}
			link(t, many(t, maxFollowedEntries/2+1), filepath.Join(repo, ".git", "hooks", "b"))
			return ".git/hooks/b"
		}},
		{name: "the worktrees directory linked to a tree past the entry budget", plant: func(t *testing.T, repo string) string {
			link(t, many(t, maxFollowedEntries+1), filepath.Join(repo, ".git", "worktrees"))
			return ".git/worktrees"
		}},
		{name: "a worktree entry linked to a directory whose HEAD is past the byte budget", plant: func(t *testing.T, repo string) string {
			link(t, huge(t, "HEAD"), filepath.Join(repo, ".git", "worktrees", "planted"))
			return ".git/worktrees/planted"
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newWrittenRun(t, "")
			named := c.plant(t, r.repo)
			refused(t, within(t, func() error { _, err := readTree(r.repo, ""); return err }), named)
		})
	}
}

// TestAReadingOutsideTheTreeIsBounded: a hooks directory core.hooksPath
// names outside the tree, and the push receipts of a worktree git lists, are
// read within the budget the followed links are held to, so a role pointing
// core.hooksPath at a large tree, or registering a worktree whose receipts
// directory is one, is refused promptly, naming the value or the directory,
// rather than read toward the general bound. The hooks directory shares the
// links' budget of entries and bytes; each receipts directory is held to the
// entry budget on its own and shares the byte budget, so a checkout with many
// worktrees, each holding the receipts it keeps, is read whole.
func TestAReadingOutsideTheTreeIsBounded(t *testing.T) {
	refused := func(t *testing.T, err error, named ...string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), "leads past what one reading follows links to") {
			t.Fatalf("err = %v, want the reading refused past the budget", err)
		}
		for _, n := range named {
			if !strings.Contains(err.Error(), n) {
				t.Fatalf("err = %v, want it to name %s", err, n)
			}
		}
	}
	// register makes git list a worktree at dir, holding its receipts.
	register := func(t *testing.T, r *writtenRun, name, dir string) {
		t.Helper()
		head := r.git.Git("rev-parse", "HEAD")
		r.git.Write(".git/worktrees/"+name+"/gitdir", filepath.Join(dir, ".git")+"\n")
		r.git.Write(".git/worktrees/"+name+"/HEAD", head+"\n")
		r.git.Write(".git/worktrees/"+name+"/commondir", "../..\n")
	}
	receiptsAt := func(t *testing.T, dir string) string {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(receiptsRel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}

	for _, c := range []struct {
		name string
		// plant sets the reading up, returning what the refusal must name.
		plant func(t *testing.T, r *writtenRun) []string
	}{
		{name: "core.hooksPath at a directory holding a file past the byte budget", plant: func(t *testing.T, r *writtenRun) []string {
			dir := bigSparseDir(t, "pre-push")
			r.git.Git("config", "core.hooksPath", dir)
			return []string{"core.hooksPath", dir}
		}},
		{name: "core.hooksPath at a directory past the entry budget", plant: func(t *testing.T, r *writtenRun) []string {
			dir := dirOfEmpties(t, maxFollowedEntries+1)
			r.git.Git("config", "core.hooksPath", dir)
			return []string{"core.hooksPath", dir}
		}},
		{name: "core.hooksPath at a link to a large directory, named as configured", plant: func(t *testing.T, r *writtenRun) []string {
			at := filepath.Join(t.TempDir(), "githooks")
			if err := os.Symlink(bigSparseDir(t, "pre-commit"), at); err != nil {
				t.Fatal(err)
			}
			r.git.Git("config", "core.hooksPath", at)
			return []string{"core.hooksPath", at}
		}},
		{name: "core.hooksPath and a hook link within the budget alone, past it together", plant: func(t *testing.T, r *writtenRun) []string {
			if err := os.Symlink(dirOfEmpties(t, maxFollowedEntries/2+1), filepath.Join(r.repo, ".git", "hooks", "a")); err != nil {
				t.Fatal(err)
			}
			if _, err := readTree(r.repo, ""); err != nil {
				t.Fatalf("one link within the budget is refused: %v", err)
			}
			dir := dirOfEmpties(t, maxFollowedEntries/2+1)
			r.git.Git("config", "core.hooksPath", dir)
			return []string{"core.hooksPath", dir}
		}},
		{name: "a registered worktree's receipts directory holding a file past the byte budget", plant: func(t *testing.T, r *writtenRun) []string {
			r.git.Commit("base")
			wt := t.TempDir()
			p := receiptsAt(t, wt)
			if err := os.Rename(bigSparseDir(t, r.git.Git("rev-parse", "HEAD")), p); err != nil {
				t.Fatal(err)
			}
			register(t, r, "planted", wt)
			return []string{"push receipts", filepath.ToSlash(p)}
		}},
		{name: "a registered worktree's receipts directory past the entry budget", plant: func(t *testing.T, r *writtenRun) []string {
			r.git.Commit("base")
			wt := t.TempDir()
			p := receiptsAt(t, wt)
			if err := os.Rename(dirOfEmpties(t, maxFollowedEntries+1), p); err != nil {
				t.Fatal(err)
			}
			register(t, r, "planted", wt)
			return []string{"push receipts", filepath.ToSlash(p)}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newWrittenRun(t, "")
			named := c.plant(t, r)
			refused(t, readWithin(t, func() error { _, err := readTree(r.repo, ""); return err }), named...)
		})
	}

	t.Run("many worktrees each holding the receipts a checkout keeps are read whole", func(t *testing.T) {
		r := newWrittenRun(t, "")
		r.git.Commit("base")
		const worktrees, kept = 25, 50
		for i := range worktrees {
			wt := t.TempDir()
			p := receiptsAt(t, wt)
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
			for j := range kept {
				id := fmt.Sprintf("%040x", i*kept+j)
				if err := os.WriteFile(filepath.Join(p, id), []byte("commit "+id+"\nminted 2026-10-05T00:00:00Z\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			register(t, r, fmt.Sprintf("lane%02d", i), wt)
		}
		if worktrees*(kept+1) <= maxFollowedEntries {
			t.Fatalf("the case holds %d receipts entries, within one shared entry budget; it proves nothing", worktrees*(kept+1))
		}
		if _, err := readTree(r.repo, ""); err != nil {
			t.Fatalf("a checkout's kept receipts are refused: %v", err)
		}
	})

	t.Run("a role writing core.hooksPath to a large tree is refused promptly, the answers kept", func(t *testing.T) {
		script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
		r := newWrittenRun(t, routedToClaude)
		r.git.Commit("base")
		dir := bigSparseDir(t, "pre-push")
		cfg, err := os.ReadFile(filepath.Join(r.repo, ".git", "config"))
		if err != nil {
			t.Fatal(err)
		}
		stubAlso(t, script, 2, map[string]string{".git/config": string(cfg) + "[core]\n\thooksPath = " + dir + "\n"})
		var res WrittenResult
		err = readWithin(t, func() (err error) {
			res, err = r.w.Run(context.Background())
			return err
		})
		refused(t, err, "core.hooksPath", dir)
		if len(r.asked) != 1 || len(r.finished) != 0 {
			t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the refusal", len(r.asked), len(r.finished))
		}
		if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
			t.Fatalf("record %+v", rec)
		}
	})
}

// readWithin runs read and fails the test when it does not end within the
// time bound: reading the large trees the budget tests plant whole takes
// minutes.
func readWithin(t *testing.T, read func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- read() }()
	select {
	case err := <-done:
		return err
	case <-time.After(20 * time.Second):
		t.Fatal("the reading did not end within 20s: a place outside the tree was read past its budget")
		return nil
	}
}

// bigSparseDir is a directory holding name, a sparse file far past the byte
// budget: making it costs nothing, hashing it whole takes minutes.
func bigSparseDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "huge")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(f.Truncate(128<<30), f.Close()); err != nil {
		t.Fatal(err)
	}
	return dir
}

// dirOfEmpties is a directory holding n empty files.
func dirOfEmpties(t *testing.T, n int) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "many")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range n {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%05d", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestARedirectedGitDirectoryIsNamedNotRead: git's own directory and common
// directory are placed by the run's first reading, before any role ran, and
// every later reading reads those same directories. The files that place
// them, a git directory's commondir and a linked worktree's .git file, are
// read before git is run, so a role writing either to lead elsewhere (here to
// a common directory whose packed refs take git minutes to read) is stopped
// promptly, the file named, without git or the guard reading where it leads.
func TestARedirectedGitDirectoryIsNamedNotRead(t *testing.T) {
	// elsewhere is a common directory git accepts, whose packed refs are a
	// sparse file far past what git reads quickly.
	elsewhere := func(t *testing.T, r *writtenRun) string {
		t.Helper()
		dir := filepath.Join(t.TempDir(), "elsewhere")
		for _, sub := range []string{"objects", "refs"} {
			if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		cfg, err := os.ReadFile(filepath.Join(r.repo, ".git", "config"))
		if err != nil {
			t.Fatal(err)
		}
		for name, body := range map[string]string{"HEAD": "ref: refs/heads/main\n", "config": string(cfg)} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		f, err := os.Create(filepath.Join(dir, "packed-refs"))
		if err != nil {
			t.Fatal(err)
		}
		if err := errors.Join(f.Truncate(128<<30), f.Close()); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	// lane makes a linked worktree of r and runs the interview there.
	lane := func(t *testing.T, r *writtenRun) string {
		t.Helper()
		r.git.Commit("base")
		dir := filepath.Join(t.TempDir(), "lane")
		r.git.Git("worktree", "add", "-q", "-b", "lane", dir)
		if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o700); err != nil {
			t.Fatal(err)
		}
		r.w.Repo = dir
		return dir
	}
	for _, c := range []struct {
		name string
		// write is what the role writes, given the repository; named is the
		// one path the refusal must name, matched at its end.
		write func(t *testing.T, r *writtenRun) (files map[string]string, named string)
	}{
		{name: "a commondir written into a main checkout's git directory", write: func(t *testing.T, r *writtenRun) (map[string]string, string) {
			return map[string]string{".git/commondir": elsewhere(t, r) + "\n"}, ".git/commondir"
		}},
		{name: "a linked worktree's commondir rewritten", write: func(t *testing.T, r *writtenRun) (map[string]string, string) {
			lane(t, r)
			p := filepath.ToSlash(filepath.Join(r.repo, ".git", "worktrees", "lane", "commondir"))
			return map[string]string{p: elsewhere(t, r) + "\n"}, "/.git/worktrees/lane/commondir"
		}},
		{name: "a linked worktree's .git file pointed at another git directory", write: func(t *testing.T, r *writtenRun) (map[string]string, string) {
			lane(t, r)
			gitDir := filepath.Join(t.TempDir(), "gitdir")
			if err := os.Mkdir(gitDir, 0o755); err != nil {
				t.Fatal(err)
			}
			head := r.git.Git("rev-parse", "HEAD")
			for name, body := range map[string]string{"HEAD": head + "\n", "commondir": elsewhere(t, r) + "\n"} {
				if err := os.WriteFile(filepath.Join(gitDir, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			return map[string]string{".git": "gitdir: " + gitDir + "\n"}, ".git"
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			files, named := c.write(t, r)
			stubAlso(t, script, 2, files)
			var res WrittenResult
			err := readWithin(t, func() (err error) {
				res, err = r.w.Run(context.Background())
				return err
			})
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) || len(uc.Paths) != 1 || !strings.HasSuffix(uc.Paths[0], named) {
				t.Fatalf("err = %v, want the interview stopped on %s alone", err, named)
			}
			if len(r.asked) != 1 || len(r.finished) != 0 {
				t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the change", len(r.asked), len(r.finished))
			}
			if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
				t.Fatalf("record %+v", rec)
			}
		})
	}

	t.Run("a later reading reads the directories the first one placed", func(t *testing.T) {
		r := newWrittenRun(t, "")
		pin := &gitPin{}
		if _, err := readTreePinned(r.repo, "", pin); err != nil {
			t.Fatal(err)
		}
		if pin.gitDir == "" || pin.common == "" {
			t.Fatalf("the first reading placed nothing: %+v", pin)
		}
		if _, err := readTreePinned(r.repo, "", pin); err != nil {
			t.Fatalf("a second reading of an unchanged tree is refused: %v", err)
		}
		// git naming another directory than the pinned one, by a route the
		// pointers do not show, is refused rather than read.
		placed := pin.common
		pin.common = t.TempDir()
		_, err := readTreePinned(r.repo, "", pin)
		if err == nil || !strings.Contains(err.Error(), placed) || !strings.Contains(err.Error(), pin.common) {
			t.Fatalf("err = %v, want the reading refused, naming both directories", err)
		}
	})
}

// TestTheBytesOneReadingHashesAreBounded: the paths git lists as differing
// from HEAD, and git's own directory, are hashed within a byte bound beside
// maxWatched, so a role writing one large file (a sparse one costs nothing to
// make and minutes to hash) is refused promptly, the file named, before it is
// read; a tree already past the bound is refused before the first dispatch,
// naming its largest path, and a tree at the bound is read.
func TestTheBytesOneReadingHashesAreBounded(t *testing.T) {
	refused := func(t *testing.T, err error, named string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), "one reading hashes") || !strings.Contains(err.Error(), named) {
			t.Fatalf("err = %v, want the reading refused past the byte bound, naming %s", err, named)
		}
	}
	sparse := func(t *testing.T, p string, size int64) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := errors.Join(f.Truncate(size), f.Close()); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("an untracked file past the bound, before the first dispatch", func(t *testing.T) {
		r := newWrittenRun(t, "")
		sparse(t, filepath.Join(r.repo, "data", "dump.bin"), 128<<30)
		refused(t, readWithin(t, func() error { _, err := readTree(r.repo, ""); return err }), "data/dump.bin")
	})

	t.Run("a file past the bound in git's own directory", func(t *testing.T) {
		r := newWrittenRun(t, "")
		sparse(t, filepath.Join(r.repo, ".git", "hooks", "pre-push"), 128<<30)
		refused(t, readWithin(t, func() error { _, err := readTree(r.repo, ""); return err }), ".git/hooks/pre-push")
	})

	t.Run("a role writing a large untracked file is refused promptly, the answers kept", func(t *testing.T) {
		script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
		r := newWrittenRun(t, routedToClaude)
		stubAlso(t, script, 2, map[string]string{"planted.bin": sparseToken + strconv.FormatInt(128<<30, 10)})
		var res WrittenResult
		err := readWithin(t, func() (err error) {
			res, err = r.w.Run(context.Background())
			return err
		})
		refused(t, err, "planted.bin")
		if len(r.asked) != 1 || len(r.finished) != 0 {
			t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the refusal", len(r.asked), len(r.finished))
		}
		if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
			t.Fatalf("record %+v", rec)
		}
	})

	t.Run("a tree at the bound is read, one past it refused naming its largest path", func(t *testing.T) {
		keep := maxHashedBytes
		t.Cleanup(func() { maxHashedBytes = keep })
		r := newWrittenRun(t, "")
		r.git.Write("small.txt", strings.Repeat("s", 10))
		r.git.Write("large.txt", strings.Repeat("l", 1000))
		maxHashedBytes = 1 << 40
		if _, err := readTree(r.repo, ""); err != nil {
			t.Fatal(err)
		}
		// What the git directory and the two files hash, read by a reading
		// with room to spare, is the bound the next readings are held to.
		used := hashedBy(t, r.repo)
		maxHashedBytes = used
		if _, err := readTree(r.repo, ""); err != nil {
			t.Fatalf("a reading of %d bytes at the bound is refused: %v", used, err)
		}
		maxHashedBytes = used - 1
		if _, err := readTree(r.repo, ""); err == nil || !strings.Contains(err.Error(), "one reading hashes") {
			t.Fatalf("err = %v, want a reading of %d bytes past the bound of %d refused", err, used, maxHashedBytes)
		}
		// The two files git lists hold 1,010 bytes: past a bound below that,
		// the reading is refused before any of them is hashed, naming the
		// larger.
		maxHashedBytes = 1009
		refused(t, readWithin(t, func() error { _, err := readTree(r.repo, ""); return err }), "large.txt")
	})
}

// hashedBy is the bytes one reading of repo hashes.
func hashedBy(t *testing.T, repo string) int64 {
	t.Helper()
	r := &treeReader{repo: repo, realRepo: repo, skip: []string{}, st: treeState{}}
	if err := r.read(); err != nil {
		t.Fatal(err)
	}
	return r.hashedBytes
}

// TestWhatGitReadsWholeIsSizedBeforeGitRuns: every reading runs git, and git
// reads some files in its own directory whole on every command (HEAD, the
// packed refs, info/exclude), so those are sized before git runs, against
// the bound on what one reading hashes. A role growing one of them (a sparse
// file costs one write) is refused promptly, the file named, rather than held
// while git reads it.
func TestWhatGitReadsWholeIsSizedBeforeGitRuns(t *testing.T) {
	refused := func(t *testing.T, err error, named string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), "one reading hashes") || !strings.Contains(err.Error(), named) {
			t.Fatalf("err = %v, want the reading refused before git runs, naming %s", err, named)
		}
	}
	grow := func(t *testing.T, p string) {
		t.Helper()
		f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		if err := errors.Join(f.Truncate(128<<30), f.Close()); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		name string
		// at is the repository read, given the set-up one; grow grows the
		// file, returning the path the refusal must name, matched at its end.
		at   func(t *testing.T, r *writtenRun) string
		grow func(t *testing.T, r *writtenRun) string
	}{
		{name: "the packed refs", grow: func(t *testing.T, r *writtenRun) string {
			grow(t, filepath.Join(r.repo, ".git", "packed-refs"))
			return ".git/packed-refs"
		}},
		{name: "HEAD", grow: func(t *testing.T, r *writtenRun) string {
			grow(t, filepath.Join(r.repo, ".git", "HEAD"))
			return ".git/HEAD"
		}},
		{name: "a linked worktree's own HEAD", at: func(t *testing.T, r *writtenRun) string {
			r.git.Commit("base")
			lane := filepath.Join(t.TempDir(), "lane")
			r.git.Git("worktree", "add", "-q", "-b", "lane", lane)
			return lane
		}, grow: func(t *testing.T, r *writtenRun) string {
			grow(t, filepath.Join(r.repo, ".git", "worktrees", "lane", "HEAD"))
			return "/.git/worktrees/lane/HEAD"
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newWrittenRun(t, "")
			repo := r.repo
			if c.at != nil {
				repo = c.at(t, r)
			}
			pin := &gitPin{}
			if _, err := readTreePinned(repo, "", pin); err != nil {
				t.Fatal(err)
			}
			named := c.grow(t, r)
			refused(t, readWithin(t, func() error { _, err := readTreePinned(repo, "", pin); return err }), named)
		})
	}

	t.Run("a role growing the packed refs is refused promptly, the answers kept", func(t *testing.T) {
		script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
		r := newWrittenRun(t, routedToClaude)
		r.git.Commit("base")
		stubAlso(t, script, 2, map[string]string{".git/packed-refs": sparseToken + strconv.FormatInt(128<<30, 10)})
		var res WrittenResult
		err := readWithin(t, func() (err error) {
			res, err = r.w.Run(context.Background())
			return err
		})
		refused(t, err, ".git/packed-refs")
		if len(r.asked) != 1 || len(r.finished) != 0 {
			t.Fatalf("asked %d, finished %d; nothing is drawn or finished after the refusal", len(r.asked), len(r.finished))
		}
		if rec := readRecord(t, res.Record); len(rec.Answers) != 1 {
			t.Fatalf("record %+v", rec)
		}
	})
}

// withSubmodule adds a tracked, populated submodule named sub to r, whose
// git directory git places under the common directory's modules/: git status
// in the checkout runs status inside it, which reads that git directory too.
func withSubmodule(t *testing.T, r *writtenRun) {
	t.Helper()
	src := gittest.NewRepo(t)
	src.Write("lib.txt", "lib\n")
	src.Commit("lib")
	r.git.Git("-c", "protocol.file.allow=always", "submodule", "add", "-q", src.Root(), "sub")
	r.git.Commit("submodule")
	if _, err := os.Stat(filepath.Join(r.repo, ".git", "modules", "sub", "HEAD")); err != nil {
		t.Fatalf("the submodule's git directory is not under modules/: %v", err)
	}
}

// TestASubmodulesGitDirectoryIsSizedAndWatched: git status runs status inside
// every populated submodule, which reads the submodule's git directory under
// modules/ whole as it reads the checkout's own, so those files are sized
// before git runs, against the same bound, and read as the common directory
// is. A role growing a submodule's packed refs is refused promptly, the file
// named, rather than held while git reads it, and a small write there is
// named.
func TestASubmodulesGitDirectoryIsSizedAndWatched(t *testing.T) {
	t.Run("a submodule's packed refs grown past the bound is refused before git runs", func(t *testing.T) {
		r := newWrittenRun(t, "")
		withSubmodule(t, r)
		pin := &gitPin{}
		if _, err := readTreePinned(r.repo, "", pin); err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(filepath.Join(r.repo, ".git", "modules", "sub", "packed-refs"), os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		if err := errors.Join(f.Truncate(128<<30), f.Close()); err != nil {
			t.Fatal(err)
		}
		err = readWithin(t, func() error { _, err := readTreePinned(r.repo, "", pin); return err })
		if err == nil || !strings.Contains(err.Error(), "one reading hashes") || !strings.Contains(err.Error(), ".git/modules/sub/packed-refs") {
			t.Fatalf("err = %v, want the reading refused before git runs, naming .git/modules/sub/packed-refs", err)
		}
	})

	for _, c := range []struct{ name, path string }{
		{name: "a small write to a submodule's packed refs", path: ".git/modules/sub/packed-refs"},
		{name: "a submodule's HEAD moved", path: ".git/modules/sub/HEAD"},
		{name: "a new ref in a submodule", path: ".git/modules/sub/refs/heads/planted"},
	} {
		t.Run(c.name+" is named", func(t *testing.T) {
			script := stubOnPath(t, stubAsk("Product Q1", "Is that answer complete?"), stubAsk("Product Q2", "Is the second answer complete?"), stubDone)
			r := newWrittenRun(t, routedToClaude)
			withSubmodule(t, r)
			head := r.git.Git("-C", "sub", "rev-parse", "HEAD")
			body := map[string]string{
				".git/modules/sub/packed-refs":        head + " refs/heads/packed\n",
				".git/modules/sub/HEAD":               head + "\n",
				".git/modules/sub/refs/heads/planted": head + "\n",
			}[c.path]
			stubAlso(t, script, 2, map[string]string{c.path: body})
			res, err := r.w.Run(context.Background())
			var uc *UnexpectedChangesError
			if !errors.As(err, &uc) || !slices.Equal(uc.Paths, []string{c.path}) {
				t.Fatalf("err = %v, want the interview stopped on %s alone", err, c.path)
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

// TestALaterReadingSizesWhatGitReadsBeforeAskingGitWhereItsDirectoriesAre: a later
// reading sizes what git reads whole in the directories the first reading
// placed before it asks git where its directories are, since that question
// reads the configuration whole too; and it asks for the working tree with
// them, so a core.worktree pointed at another tree is refused before git
// status walks it.
func TestALaterReadingSizesWhatGitReadsBeforeAskingGitWhereItsDirectoriesAre(t *testing.T) {
	t.Run("a configuration grown past the bound is refused before git runs", func(t *testing.T) {
		r := newWrittenRun(t, "")
		pin := &gitPin{}
		if _, err := readTreePinned(r.repo, "", pin); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(r.repo, ".git", "config")
		f, err := os.OpenFile(p, os.O_RDWR|os.O_APPEND, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		// A comment git reads to its end: the zeros after it are not a line
		// git refuses at.
		if _, err := f.WriteString("#"); err != nil {
			t.Fatal(err)
		}
		if err := errors.Join(f.Truncate(128<<30), f.Close()); err != nil {
			t.Fatal(err)
		}
		err = readWithin(t, func() error { _, err := readTreePinned(r.repo, "", pin); return err })
		if err == nil || !strings.Contains(err.Error(), "so git is not run") || !strings.Contains(err.Error(), ".git/config") {
			t.Fatalf("err = %v, want the reading refused before git runs, naming .git/config", err)
		}
	})

	t.Run("a working tree moved by core.worktree is refused before status", func(t *testing.T) {
		r := newWrittenRun(t, "")
		pin := &gitPin{}
		if _, err := readTreePinned(r.repo, "", pin); err != nil {
			t.Fatal(err)
		}
		other, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		r.git.Git("config", "core.worktree", other)
		_, err = readTreePinned(r.repo, "", pin)
		if err == nil || !strings.Contains(err.Error(), "working tree") || !strings.Contains(err.Error(), other) {
			t.Fatalf("err = %v, want the reading refused, naming the working tree git now names", err)
		}
	})
}
