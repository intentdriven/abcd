package interview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
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
	linkedHooks, linkedFromTree := t.TempDir(), t.TempDir()
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
