package ahoy

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The clone-local commit-msg shim. The product thinker ruled on 2026-10-15
// that the commit-message check needs no manual step, within the rule that abcd
// changes nothing about how a repository works beyond what the person opted
// into (iss-2610152024267547). The committed hook in .githooks runs only in a
// clone whose core.hooksPath points there, and setting that key would choose
// every hook git runs in the clone. So, on the explicit --outbound alone, where
// no core.hooksPath is set (git then runs $GIT_COMMON_DIR/hooks) and nothing
// occupies that directory's commit-msg, install writes abcd's own embedded
// template there.
//
// The shim is the same embedded template as the committed copy, byte for byte,
// and finds abcd the same one way (abcd.binary). It never delegates to or
// executes the committed .githooks copy, so no repository-authored code runs
// through it. It lives in the common git directory, so every worktree of the
// clone shares the one hook. It never replaces a hook abcd did not write, never
// sets core.hooksPath and never touches global config: where git runs other
// hooks, or a commit-msg hook is already there, it writes nothing and one note
// says the check is not wired, why, and the line to chain it with.

// shimAction is what the explicit opt-in does to the clone's commit-msg.
type shimAction int

const (
	shimNone    shimAction = iota // nothing to write: wired, or not abcd's to wire (see note)
	shimCreate                    // nothing at the path: write the template
	shimRefresh                   // abcd's own shim, not the current template: replace it
)

// shimHookRel is the commit-msg hook's path inside the common git directory.
const shimHookRel = "hooks/commit-msg"

// outboundShimChainLine is the one line a person adds to a commit-msg hook of
// their own to run the check. It runs only the abcd this clone's abcd.binary
// names, and does nothing when none is named.
const outboundShimChainLine = `if bin=$(git config --local --get abcd.binary); then "$bin" lint outbound --label "commit message" "$1" || exit 1; fi`

// outboundShimPlan is what the explicit opt-in would do to the clone's
// commit-msg, read without writing anything.
type outboundShimPlan struct {
	gitDir string     // the common git directory; "" when git could not name it
	path   string     // the absolute commit-msg path git runs with no hooks path
	action shimAction // what to write there
	note   string     // why the check is not wired; "" when it is, or will be
}

// gitCommonDir names the clone's common git directory, absolute: the one every
// worktree shares, and so the one whose hooks git runs for each of them.
func gitCommonDir(cwd string) (string, error) {
	out, err := gitutil.Run(cwd, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", errors.New("git named no git directory")
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(cwd, out)
	}
	return filepath.Clean(out), nil
}

// notWired renders the note for a clone whose check this run did not wire.
func notWired(why, where string) string {
	return "the commit-message check is not wired into git in this clone: " + why +
		". abcd never replaces a hook, sets core.hooksPath or changes global git config, so to run the check add this line to " +
		where + ": " + outboundShimChainLine
}

// hookDirRunsCheck reports whether the commit-msg in dir is abcd's template or
// runs `abcd lint outbound` itself.
func hookDirRunsCheck(dir string) bool {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return false
	}
	defer root.Close()
	state, runsCheck := classifyOutboundHook(root, "commit-msg")
	return state == HookInstalled || runsCheck
}

// planOutboundShim reads what the explicit opt-in would do to the clone's
// commit-msg hook. It writes nothing; stepOutboundShim acts on it, and
// outboundWouldChange counts it.
func planOutboundShim(cwd string) outboundShimPlan {
	paths, err := gitutil.HooksPaths(cwd)
	if err != nil {
		return outboundShimPlan{note: notWired("git could not say whether core.hooksPath is set ("+errText(err)+"), so abcd wrote no hook", "the commit-msg hook git runs in this clone")}
	}
	if len(paths) > 0 {
		// The last value is the one git runs. A relative value is relative to
		// the working tree's root. Its path is never printed: a directory name
		// on it can be a private name.
		dir := paths[len(paths)-1]
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(cwd, dir)
		}
		if hookDirRunsCheck(dir) {
			return outboundShimPlan{}
		}
		return outboundShimPlan{note: notWired("core.hooksPath is set, so git runs the hooks in the directory it names (git config --get core.hooksPath prints it) and never .git/hooks",
			"the commit-msg hook in that directory")}
	}
	gitDir, err := gitCommonDir(cwd)
	if err != nil {
		return outboundShimPlan{note: notWired("git could not name this clone's git directory ("+errText(err)+"), so abcd wrote no hook", "the commit-msg hook git runs in this clone")}
	}
	plan := outboundShimPlan{gitDir: gitDir, path: filepath.Join(gitDir, filepath.FromSlash(shimHookRel))}
	shown := receiptPath(cwd, plan.path)
	root, err := os.OpenRoot(gitDir)
	if err != nil {
		plan.note = notWired("this clone's git directory could not be opened ("+errText(err)+"), so abcd wrote no hook", shown)
		return plan
	}
	defer root.Close()
	state, runsCheck := classifyOutboundHook(root, shimHookRel)
	switch state {
	case HookAbsent:
		plan.action = shimCreate
	case HookInstalled:
		data, err := fsutil.ReadGuardedInRoot(root, shimHookRel, maxAhoyFileBytes)
		if err != nil || !bytes.Equal(data, outboundHookTemplate) {
			plan.action = shimRefresh
		}
	case HookForeign:
		if !runsCheck {
			plan.note = notWired("a commit-msg hook abcd did not write is already at "+shown, "it")
		}
	default:
		plan.note = notWired(shown+" exists but cannot be read as a hook (a symlink, a directory or an unreadable file)", "it once it is a regular file")
	}
	return plan
}

// outboundStandingNotes is the shim's note for a run that has nothing to
// write, so a rerun of the explicit opt-in still says the check is not wired.
func outboundStandingNotes(cwd string, opts InstallOptions) []string {
	if !opts.Outbound {
		return nil
	}
	if note := planOutboundShim(cwd).note; note != "" {
		return []string{note}
	}
	return nil
}

// stepOutboundShim wires the commit-message check into git for this clone, on
// the explicit --outbound alone. It writes abcd's embedded template where
// nothing is, refreshes abcd's own shim when it is not the current template,
// and otherwise writes nothing, giving the plan's note when the check is not
// wired. A refresh re-checks the marker just before it replaces the file, so a
// hook that changed hands since the plan was read is left alone.
func (a *applyCtx) stepOutboundShim() {
	if !a.outbound {
		return
	}
	plan := planOutboundShim(a.cwd)
	if plan.note != "" {
		a.refuse(plan.note)
		return
	}
	if plan.action == shimNone {
		return
	}
	shown := receiptPath(a.cwd, plan.path)
	failed := func(err error) {
		a.refuse(notWired(shown+" could not be written ("+errText(err)+")", shown))
	}
	root, err := os.OpenRoot(plan.gitDir)
	if err != nil {
		failed(err)
		return
	}
	defer root.Close()
	switch plan.action {
	case shimCreate:
		dir := filepath.Dir(filepath.FromSlash(shimHookRel))
		_, statErr := root.Lstat(dir)
		if err := root.MkdirAll(dir, 0o755); err != nil {
			failed(err)
			return
		}
		if statErr != nil {
			if err := root.Chmod(dir, 0o755); err != nil {
				failed(err)
				return
			}
		}
		if err := fsutil.CreateExclusiveIn(root, shimHookRel, outboundHookTemplate, 0o755); err != nil {
			failed(err)
			return
		}
		// A umask-stripped exec bit would leave a hook git never runs.
		if err := root.Chmod(shimHookRel, 0o755); err != nil {
			failed(err)
			return
		}
	case shimRefresh:
		if classifyHook(root, shimHookRel, outboundHookMarkerRe) != HookInstalled {
			a.refuse(notWired(shown+" changed while install ran and is no longer abcd's, so abcd left it", shown))
			return
		}
		if err := fsutil.WriteFileAtomicInRoot(root, shimHookRel, outboundHookTemplate, 0o755); err != nil {
			failed(err)
			return
		}
	}
	a.note(writeOutboundShim, plan.path)
}
