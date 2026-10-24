package ahoy

// The commit checks' own question (iss-2610241134315056). The hooks setup
// commits in .githooks run only in a clone whose core.hooksPath points there,
// and a fresh clone sets no hooks path, so a clone setup reported clean ran
// none of them. Setting the key chooses every hook git runs in the clone, so
// it is a change only the person approves, asked as its own question, on the
// pattern of the shell-settings offer (shell_path.go):
//
//   - It is asked only of a person at a terminal: --yes never gives the yes,
//     and off a terminal it is never put. Its category is never asked as an
//     approval: the question is its own consent.
//   - It is offered only where no hooks path is set at any scope. A hooks path
//     set outside the clone (a global or system dispatcher, banlist.hooks_path
//     "foreign") is left alone, never shadowed by a local one; a local value
//     naming another directory is the person's, and is reported, not replaced.
//   - Until the hooks run, the install is never reported clean: the gap stays,
//     and the report lists the exact command.

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// HooksPathCategory covers pointing git at the committed hooks in this clone.
const HooksPathCategory GapCategory = "hooks-path"

// HooksArmGapID is the offer to point git at the committed hooks.
const HooksArmGapID = "banlist.hooks_unarmed"

// HooksElsewhereGapID is a clone whose own config points git at another
// hooks directory, so the committed hooks do not run; abcd does not replace
// the person's value.
const HooksElsewhereGapID = "banlist.hooks_path_elsewhere"

// ArmHooksCommand is the one command that points this clone's git at the
// committed hooks, which the report lists when the question is not answered.
const ArmHooksCommand = "git config --local core.hooksPath " + guardHooksDirRelPath

// armHooksAsk is the question the offer ends on.
const armHooksAsk = "Switch on this repository's commit checks, setting core.hooksPath to " + guardHooksDirRelPath +
	" in this clone's .git/config?"

// localHooksPathUnset reports whether this clone's own config sets no
// core.hooksPath. known is false when git did not answer (not a repository,
// a git that failed), which offers nothing.
func localHooksPathUnset(cwd string) (unset, known bool) {
	out, err := gitutil.Run(cwd, "config", "--local", "--get", "core.hooksPath")
	if err == nil {
		return strings.TrimSpace(out) == "", true
	}
	// Exit status 1 is git's "no such key" in a repository it answers for.
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return true, true
	}
	return false, false
}

// detectHooksArm raises the offer (or, for a local value naming another
// directory, the diagnosis) while abcd's committed guard is installed and git
// does not run it in this clone.
func detectHooksArm(cwd string, h BanlistHealth) []Gap {
	if h.Hook != HookInstalled || h.HooksPath != HooksPathStateUnarmed {
		return nil
	}
	unset, known := localHooksPathUnset(cwd)
	switch {
	case !known:
		return nil
	case unset:
		return []Gap{hooksArmGap()}
	}
	return []Gap{{
		ID: HooksElsewhereGapID, Category: ConfigChange, Scope: "repo",
		Title: "git runs another hooks directory in this clone, not " + guardHooksDirRelPath,
		Detail: "This clone's .git/config sets core.hooksPath to another directory (`git config --local --get core.hooksPath` prints it), " +
			"so the commit checks committed in " + guardHooksDirRelPath + " never run here.",
		FixHint: "Call the hooks in " + guardHooksDirRelPath + "/ from the hooks in that directory, or point git at them yourself: " +
			ArmHooksCommand + " (abcd does not replace a value you set).",
		Required: false, Resolvable: false,
	}}
}

func hooksArmGap() Gap {
	return Gap{
		ID: HooksArmGapID, Category: HooksPathCategory, Scope: "repo",
		Title: "git does not run this repository's commit checks in this clone",
		Detail: GuardHookRelPath + " and the other hooks in " + guardHooksDirRelPath + " are committed, but core.hooksPath is not " +
			"set in this clone, so git never runs them and no commit is checked.",
		FixHint:  "ahoy install asks at a terminal and sets it only on a yes; --yes never sets it. Or run it yourself: " + ArmHooksCommand,
		Required: false, Resolvable: true,
	}
}

// hooksArmStep is the report's step for a clone whose committed hooks git
// does not run.
func hooksArmStep() ReportStep {
	return ReportStep{Step: hooksArmAction,
		Why: "git runs the hooks committed in " + guardHooksDirRelPath + " only once core.hooksPath points there, and this clone sets none."}
}

// hooksArmAction is the step, the same words wherever it is listed, so the
// report lists it once.
const hooksArmAction = "Switch on the commit checks: run abcd ahoy install at a terminal and answer yes to that question, or run " +
	ArmHooksCommand

// hooksUnrun reports whether gaps say committed hooks git does not run, which
// keeps the install from being reported clean.
func hooksUnrun(gaps []Gap) bool {
	return hasGapID(gaps, HooksArmGapID) || hasGapID(gaps, HooksElsewhereGapID)
}

// armOffered reports whether this run puts the question: abcd's guard is, or
// is about to be, committed, and no hooks path is set at any scope. The plan
// lists it from the same judgement (personOnlyAsks).
func armOffered(cwd string, det DetectionResult) bool {
	present := gapIDSet(det.Gaps)
	if !present[HooksArmGapID] && !present["banlist.hook_missing"] {
		return false
	}
	if hooksPathState(cwd) != HooksPathStateUnarmed {
		return false
	}
	unset, known := localHooksPathUnset(cwd)
	return known && unset
}

// stepArmHooks makes the offer. It runs after stepBanlist, so it judges the
// guard this run committed, a first install's included, and before the
// outbound shim, which plans against the hooks path this step leaves. It
// never runs under --yes or off a terminal.
func (a *applyCtx) stepArmHooks() {
	if a.autoYes || !atTerminal(a.prompter) {
		return
	}
	root, err := os.OpenRoot(a.cwd)
	if err != nil {
		return
	}
	installed := classifyGuardHook(root, GuardHookRelPath) == HookInstalled
	root.Close()
	if !installed || hooksPathState(a.cwd) != HooksPathStateUnarmed {
		return
	}
	if unset, known := localHooksPathUnset(a.cwd); !known || !unset {
		return
	}
	if !a.prompter.Confirm(armHooksQuestion(a.cwd)) {
		return
	}
	if _, err := gitutil.Run(a.cwd, "config", "--local", "core.hooksPath", guardHooksDirRelPath); err != nil {
		a.refuse("core.hooksPath was not set: " + errText(err) + ".")
		a.needYou("Switch on the commit checks yourself: "+ArmHooksCommand,
			"you said yes, and git refused to set it: "+strings.TrimSuffix(errText(err), ".")+".")
		return
	}
	// Read back: the write is the key git now runs hooks from.
	if hooksPathState(a.cwd) != HooksPathStateArmed {
		a.refuse("core.hooksPath was set, but git does not read it back as " + guardHooksDirRelPath + ".")
		a.needYou("Check what git runs here with `git config --get core.hooksPath`, and set it with "+ArmHooksCommand,
			"you said yes, and git does not read the value back as "+guardHooksDirRelPath+".")
		return
	}
	a.changes = append(a.changes, "switched on the commit checks: core.hooksPath = "+guardHooksDirRelPath+" in this clone's .git/config")
}

// armHooksQuestion is what the person consents to: which hooks start
// running, what stops running, how to undo it, and the question.
func armHooksQuestion(cwd string) string {
	text := "git does not run the commit checks committed in " + guardHooksDirRelPath
	if names := committedHooks(cwd); len(names) > 0 {
		text += " (" + strings.Join(names, ", ") + ")"
	}
	text += " in this clone: core.hooksPath is not set. Setting it makes git run the hooks in " +
		guardHooksDirRelPath + " instead of those in .git/hooks"
	if own := ownGitHooks(cwd); len(own) > 0 {
		text += ", so " + strings.Join(own, ", ") + " in .git/hooks stop running"
	}
	return text + "; `git config --local --unset core.hooksPath` undoes it.\n" + armHooksAsk
}

// isArmHooksQuestion reports whether text is the commit-checks offer.
func isArmHooksQuestion(text string) bool {
	_, ask := splitConfirm(text)
	return ask == armHooksAsk
}

// committedHooks names the hooks in the committed directory.
func committedHooks(cwd string) []string {
	return hookNames(filepath.Join(cwd, guardHooksDirRelPath))
}

// ownGitHooks names the hooks in this clone's .git/hooks that git runs now,
// the samples git ships left out.
func ownGitHooks(cwd string) []string {
	dir, err := gitCommonDir(cwd)
	if err != nil {
		return nil
	}
	return hookNames(filepath.Join(dir, "hooks"))
}

// hookNames is the executable regular files in dir, sorted.
func hookNames(dir string) []string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".sample") || !e.Type().IsRegular() {
			continue
		}
		if fi, err := e.Info(); err == nil && fi.Mode().Perm()&0o111 != 0 {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}
