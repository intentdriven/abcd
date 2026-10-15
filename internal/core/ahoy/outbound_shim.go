package ahoy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"

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
// sets core.hooksPath and never touches global config. Where git runs other
// hooks, or a commit-msg hook is already there, it saves the same template as
// hooks/abcd-commit-msg instead, a name git never runs, and one note says the
// check is not wired, why, and the line that calls that copy from the person's
// own hook: the copy then fails open exactly as the shim does, and blocks only
// on abcd's own refusal.

// shimAction is what the explicit opt-in does to the file a plan names.
type shimAction int

const (
	shimNone    shimAction = iota // nothing to write: current, or not abcd's to write (see note)
	shimCreate                    // nothing at the path: write the template
	shimRefresh                   // a template abcd wrote earlier: replace it with the current one
)

// shimHookRel is the commit-msg hook's path inside the common git directory.
// shimChainRel is where abcd saves its copy of the check for a person's own
// hook to call, under a name git never runs as a hook.
const (
	shimHookRel  = "hooks/commit-msg"
	shimChainRel = "hooks/abcd-commit-msg"
)

// outboundShimChainLine is the one line a person adds to a commit-msg hook of
// their own to run the check. It runs abcd's own copy of the template, saved at
// shimChainRel, so the checks are the template's (an absolute abcd.binary
// outside the working tree, the scissors cut, the verdict line) and it fails
// open as the template does; where the copy is missing it does nothing.
const outboundShimChainLine = `h="$(git rev-parse --git-common-dir 2>/dev/null || true)/` + shimChainRel + `"; if [ -x "$h" ]; then "$h" "$1" || exit 1; fi`

// earlierShimSHA256 is the SHA-256, hex, of every template an earlier abcd
// wrote as the shim or its chained copy. A file byte for byte one of them, or
// the current template, is abcd's to refresh; any other file carrying the
// marker is the person's edit and is left alone. None yet: the shim first
// shipped with the current template. A change to defaults/commit-msg adds the
// hash of the template it replaces. A var so a test can stand in an earlier
// template.
var earlierShimSHA256 = []string{}

// shimKind is what a file carrying abcd's marker is.
type shimKind int

const (
	shimCurrent shimKind = iota // the current template, byte for byte
	shimEarlier                 // a template an earlier abcd wrote, byte for byte
	shimEdited                  // anything else: the person's edit
)

// kindOfShim classifies the marked file at rel by its bytes.
func kindOfShim(root *os.Root, rel string) shimKind {
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxAhoyFileBytes)
	if err != nil {
		return shimEdited
	}
	if bytes.Equal(data, outboundHookTemplate) {
		return shimCurrent
	}
	sum := sha256.Sum256(data)
	if slices.Contains(earlierShimSHA256, hex.EncodeToString(sum[:])) {
		return shimEarlier
	}
	return shimEdited
}

// outboundShimPlan is what the explicit opt-in would do to the clone's
// commit-msg, read without writing anything.
type outboundShimPlan struct {
	gitDir string     // the common git directory; "" when git could not name it
	rel    string     // the file under gitDir the action writes: shimHookRel or shimChainRel
	shown  string     // how the receipt names that file
	action shimAction // what to write there
	note   string     // why the check is not wired, or a copy was left; "" when neither
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

// shimShown names rel, a file under the common git directory, for a note or
// the receipt: repo-relative where the directory is inside the working tree,
// and otherwise (a linked worktree's, the main checkout's) by what it is, never
// by its path, which can carry a private name.
func shimShown(cwd, gitDir, rel string) string {
	if p := filepath.Join(gitDir, filepath.FromSlash(rel)); under(cwd, p) {
		return filepath.ToSlash(fsutil.RepoRel(cwd, p))
	}
	return rel + " in this clone's shared git directory (git rev-parse --git-common-dir names it)"
}

// notWired renders the note for a clone whose check this run did not wire,
// where abcd's copy of the check is at chain for the person's hook to call.
func notWired(why, where, chain string) string {
	return "the commit-message check is not wired into git in this clone: " + why +
		". abcd never replaces a hook, sets core.hooksPath or changes global git config, so it keeps its copy of the check at " +
		chain + ", a name git never runs; to run the check, add this line to " + where + ": " + outboundShimChainLine
}

// notWiredNoCopy renders the note for a clone where abcd could keep no copy
// of the check either, so there is no line to give.
func notWiredNoCopy(why string) string {
	return "the commit-message check is not wired into git in this clone: " + why +
		", so abcd wrote nothing; run abcd ahoy install --outbound again once that is resolved"
}

// editedNote says a marked file abcd would refresh was edited, so it was left.
func editedNote(shown string) string {
	return shown + " is abcd's copy of the commit-message check but was edited since abcd wrote it, so abcd left it as it is and did not refresh it"
}

// readHook classifies the commit-msg at rel under root: whether it is abcd's
// (marked), and, for one abcd did not write, whether it runs the check and
// whether it does so by calling abcd's copy at shimChainRel.
func readHook(root *os.Root, rel string) (state HookState, runsCheck, chains bool) {
	state, runsCheck = classifyOutboundHook(root, rel)
	if runsCheck {
		data, err := fsutil.ReadGuardedInRoot(root, rel, maxAhoyFileBytes)
		chains = err == nil && outboundHookChainsCopy(data)
	}
	return state, runsCheck, chains
}

// hookDirRunsCheck is readHook for the commit-msg in dir.
func hookDirRunsCheck(dir string) (state HookState, runsCheck, chains bool) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return HookUnreadable, false, false
	}
	defer root.Close()
	return readHook(root, "commit-msg")
}

// planOutboundShim reads what the explicit opt-in would do to the clone's
// commit-msg hook. It writes nothing; stepOutboundShim acts on it, and
// outboundWouldChange counts it.
func planOutboundShim(cwd string) outboundShimPlan {
	gitDir, err := gitCommonDir(cwd)
	if err != nil {
		return outboundShimPlan{note: notWiredNoCopy("git could not name this clone's git directory (" + errText(err) + ")")}
	}
	root, err := os.OpenRoot(gitDir)
	if err != nil {
		return outboundShimPlan{note: notWiredNoCopy("this clone's git directory could not be opened (" + errText(err) + ")")}
	}
	defer root.Close()
	plan := outboundShimPlan{gitDir: gitDir}
	hook := shimShown(cwd, gitDir, shimHookRel)

	// why and where say, for a clone where git runs a hook other than the
	// shim, why the check is not wired and which hook the line goes in.
	var why, where string
	var runsCheck, chains bool
	paths, err := gitutil.HooksPaths(cwd)
	switch {
	case err != nil:
		why = "git could not say whether core.hooksPath is set (" + errText(err) + "), so abcd wrote no hook git runs"
		where = "the commit-msg hook git runs in this clone"
	case len(paths) > 0 && paths[len(paths)-1] == "":
		// git reads an empty value as set, and runs no hook from hooks/.
		why = "core.hooksPath is set to the empty value, so git runs no commit-msg hook in this clone, not even " + hook
		where = "a commit-msg hook in a directory you point core.hooksPath at"
	case len(paths) > 0:
		// The last value is the one git runs. A relative value is relative to
		// the working tree's root. Its path is never printed: a directory name
		// on it can be a private name.
		dir := paths[len(paths)-1]
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(cwd, dir)
		}
		state, runs, ch := hookDirRunsCheck(dir)
		if state == HookInstalled {
			return outboundShimPlan{}
		}
		runsCheck, chains = runs, ch
		why = "core.hooksPath is set, so git runs the hooks in the directory it names (git config --get core.hooksPath prints it) and never " + hook
		where = "the commit-msg hook in that directory"
	default:
		state, runs, ch := readHook(root, shimHookRel)
		switch state {
		case HookAbsent:
			plan.rel, plan.shown, plan.action = shimHookRel, hook, shimCreate
			return plan
		case HookInstalled:
			switch kindOfShim(root, shimHookRel) {
			case shimEarlier:
				plan.rel, plan.shown, plan.action = shimHookRel, hook, shimRefresh
			case shimEdited:
				plan.note = editedNote(hook)
			}
			return plan
		case HookForeign:
			runsCheck, chains = runs, ch
			why = "a commit-msg hook abcd did not write is already at " + hook
			where = "it"
		default:
			why = hook + " could not be read as a regular file (it is a symlink, a directory or a file this account cannot read, or its path leads outside the git directory)"
			where = "the commit-msg hook git runs there"
		}
	}

	// git runs a hook abcd did not write: keep abcd's copy of the check
	// beside it for that hook to call. A hook that runs the check already
	// draws no note, and needs no copy unless it calls one, which is then
	// written and kept current.
	chain := shimShown(cwd, gitDir, shimChainRel)
	switch classifyHook(root, shimChainRel, outboundHookMarkerRe) {
	case HookAbsent:
		if runsCheck && !chains {
			return plan
		}
		plan.rel, plan.shown, plan.action = shimChainRel, chain, shimCreate
	case HookInstalled:
		switch kindOfShim(root, shimChainRel) {
		case shimEarlier:
			plan.rel, plan.shown, plan.action = shimChainRel, chain, shimRefresh
		case shimEdited:
			if runsCheck {
				plan.note = editedNote(chain)
				return plan
			}
			plan.note = notWired(why, where, chain) + " (" + editedNote(chain) + ")"
			return plan
		}
	default:
		if runsCheck && !chains {
			return plan
		}
		plan.note = notWiredNoCopy(why + "; and " + chain + " holds a file abcd did not write or could not read, so abcd left it and has no copy of the check for that hook to call (move it aside to let abcd write one)")
		return plan
	}
	if !runsCheck {
		plan.note = notWired(why, where, chain)
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
// nothing is, refreshes a template an earlier abcd wrote, and otherwise writes
// nothing, giving the plan's note when the check is not wired or a copy was
// left. A refresh re-reads the file just before it replaces it, so a file
// that changed since the plan was read is left alone.
func (a *applyCtx) stepOutboundShim() {
	if !a.outbound {
		return
	}
	plan := planOutboundShim(a.cwd)
	if plan.action != shimNone && !a.writeShim(plan) {
		return
	}
	if plan.note != "" {
		a.refuse(plan.note)
	}
}

// writeShim writes the plan's file, and reports whether it did.
func (a *applyCtx) writeShim(plan outboundShimPlan) bool {
	failed := func(err error) bool {
		a.refuse(notWiredNoCopy(plan.shown + " could not be written (" + errText(err) + ")"))
		return false
	}
	root, err := os.OpenRoot(plan.gitDir)
	if err != nil {
		return failed(err)
	}
	defer root.Close()
	switch plan.action {
	case shimCreate:
		dir := filepath.Dir(filepath.FromSlash(plan.rel))
		_, statErr := root.Lstat(dir)
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return failed(err)
		}
		if statErr != nil {
			if err := root.Chmod(dir, 0o755); err != nil {
				return failed(err)
			}
		}
		if err := fsutil.CreateExclusiveIn(root, plan.rel, outboundHookTemplate, 0o755); err != nil {
			return failed(err)
		}
		// A umask-stripped exec bit would leave a hook git never runs.
		if err := root.Chmod(plan.rel, 0o755); err != nil {
			return failed(err)
		}
	case shimRefresh:
		if classifyHook(root, plan.rel, outboundHookMarkerRe) != HookInstalled || kindOfShim(root, plan.rel) != shimEarlier {
			a.refuse(plan.shown + " changed while install ran, so abcd left it as it is")
			return false
		}
		if err := fsutil.WriteFileAtomicInRoot(root, plan.rel, outboundHookTemplate, 0o755); err != nil {
			return failed(err)
		}
	}
	kind := writeOutboundShim
	if plan.rel == shimChainRel {
		kind = writeOutboundChain
	}
	a.note(kind, plan.shown)
	return true
}
