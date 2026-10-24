package ahoy

// The shell-settings offer (itd-2610240919030303, adr-2610241022330872): when
// the folder abcd installs into is not on the PATH, setup asks a person at a
// terminal whether to add the one line that puts it there to the startup file
// their login shell reads, naming the file and the line, and adds it only on a
// yes. internal/core/shellpath decides the file, holds the safety bar and
// makes the edit; this file asks.
//
//   - It is asked only of a person at a terminal: --yes never gives the yes,
//     and off a terminal (a pipe, a routine, CI) it is never put, so a piped
//     answer stream keeps the order it had before the offer existed. Its
//     category is never asked as an approval: the offer is its own consent.
//   - Settings that already add the folder are left as they are, and a file
//     abcd will not edit (a link kept elsewhere, another account's) is never
//     offered; in both cases, and on a no, the line is listed for the person.
//   - `abcd ahoy uninstall` takes the line out again (uninstallShellPath).

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/shellpath"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// ShellPath covers adding the PATH line to the person's shell settings.
const ShellPath GapCategory = "shell-path"

// ShellPathOfferGapID is the offer of the PATH line.
const ShellPathOfferGapID = "path.shell_settings_offered"

// shellEnv is the environment the plan reads. A variable so the package's
// tests never plan against the machine's own shell settings (TestMain).
var shellEnv = shellpath.CurrentEnv

// shellApply makes the edit on a yes. A variable so a test can make it fail.
var shellApply = shellpath.Apply

// shellPathAskPrefix opens the question the offer ends on, which names the
// line (shellpath.ExportLine) and the file.
const shellPathAskPrefix = "Add `export PATH="

// detectShellPath raises the offer beside path.bin_dir_not_on_path while the
// plan says the line may be added.
func detectShellPath(dir string) []Gap {
	plan := shellpath.PlanFor(shellEnv(), dir)
	if plan.State != shellpath.Offer {
		return nil
	}
	return []Gap{{
		ID: ShellPathOfferGapID, Category: ShellPath, Scope: "machine",
		Title:  "abcd can add " + displayPath(dir) + " to the PATH in " + displayPath(plan.File),
		Detail: "abcd can add one line to " + displayPath(plan.File) + " that puts " + displayPath(dir) + " on the PATH: " + plan.Line,
		FixHint: "ahoy install asks at a terminal and adds the line only on a yes; --yes never adds it. " +
			"Or add it yourself: " + plan.Export,
		Required: false, Resolvable: true,
	}}
}

// installedBinDir is the folder of the entry this run left, when that folder
// is not on the PATH.
func (a *applyCtx) installedBinDir() (string, bool) {
	if a.binTarget == "" {
		return "", false
	}
	if present, err := fsutil.ExistsNoFollow(a.binTarget); err != nil || !present {
		return "", false
	}
	dir := filepath.Dir(a.binTarget)
	if dirOnPath(dir) {
		return "", false
	}
	return dir, true
}

// stepShellPath makes the offer. It runs after stepPathEntry, so it judges the
// entry this run actually left, the first install's included, and it never
// runs under --yes or off a terminal. What it found is kept for
// noteReachability, which says what the person still has to do.
func (a *applyCtx) stepShellPath() {
	dir, ok := a.installedBinDir()
	if !ok {
		return
	}
	plan := shellpath.PlanFor(shellEnv(), dir)
	a.shellPlan = &plan
	if plan.State != shellpath.Offer || a.autoYes || !atTerminal(a.prompter) {
		return
	}
	if !a.prompter.Confirm(shellPathQuestion(dir, plan)) {
		return
	}
	applied, err := shellApply(shellEnv(), plan)
	if err != nil {
		a.refuse("the PATH line was not added to " + displayPath(plan.File) + ": " + displayText(err.Error()) + ".")
		// The person said yes, so what is left is theirs to do, in the
		// default report: put the file back by hand where abcd could not, and
		// otherwise add the line themselves.
		var re *shellpath.RestoreError
		if errors.As(err, &re) {
			a.needYou("Copy "+displayText(re.Backup)+" over "+displayText(re.File)+" by hand",
				displayText(re.File)+" did not read back as abcd wrote it, and putting it back from the copy failed too ("+displayText(re.Cause)+").")
			return
		}
		a.needYou("Add `"+plan.Export+"` to "+displayPath(plan.File)+" yourself",
			"abcd's edit of "+displayPath(plan.File)+" after your yes wrote nothing: "+strings.TrimSuffix(displayText(err.Error()), ".")+".")
		return
	}
	a.shellAdded = true
	if applied.Backup != "" {
		a.note(writeShellPathBackup, applied.Backup)
	}
	a.note(writeShellPath, applied.File)
	a.inform("Added `" + applied.Line + "` to " + displayPath(applied.File) + "; open a new terminal for `abcd` to run by name there.")
	for _, n := range []string{applied.PruneNote, applied.RecordNote} {
		if n != "" {
			a.inform(n + ".")
		}
	}
}

// shellPathQuestion is the reason, the file, the line and the question: core
// never prints, so what the person consents to travels as the confirm's text.
// The one line names the line and the file, so the question shown without ?
// still says what a yes writes and where (itd-2610240919030303).
func shellPathQuestion(dir string, plan shellpath.Plan) string {
	file := displayPath(plan.File)
	where := "the end of " + file
	if plan.Create {
		where = "a new " + file
	}
	return displayPath(dir) + " is not on your PATH. abcd can add this line to " + where +
		", keeping a copy first. It puts " + displayPath(dir) + " ahead of the rest of your PATH, so this abcd answers " +
		"before any older copy, and abcd ahoy uninstall removes it:\n" +
		"  " + plan.Line + "\n" +
		"Add `" + plan.Export + "` to " + file + "?"
}

// isShellPathQuestion reports whether text is the shell-settings offer.
func isShellPathQuestion(text string) bool {
	i := strings.LastIndex(text, "\n")
	return i >= 0 && strings.HasPrefix(text[i+1:], shellPathAskPrefix) && strings.HasSuffix(text, "?") &&
		strings.Contains(text, shellpath.Marker)
}

// shellPathNote is what the run says about a folder still off the PATH once
// the steps are done: nothing when this run added the line (the step said
// so), that a new terminal picks it up when the settings already add it, and
// otherwise the line to add, with why abcd did not offer to add it itself.
func (a *applyCtx) shellPathNote(dir string) string {
	if a.shellAdded {
		return ""
	}
	plan := a.shellPlan
	if plan == nil {
		p := shellpath.PlanFor(shellEnv(), dir)
		plan = &p
	}
	switch plan.State {
	case shellpath.AlreadyAdded:
		return displayPath(dir) + " is not on this terminal's PATH, but " + displayPath(plan.File) +
			" already adds it; open a new terminal for `abcd` to run by name."
	case shellpath.Checklist:
		return pathReachMessage(dir) + " abcd did not offer to add it: " + displayText(plan.Reason) + "."
	}
	return pathReachMessage(dir)
}

// uninstallShellPath takes out the PATH line abcd added, and nothing else.
func uninstallShellPath(dir string) ShellPathReceipt {
	r, err := shellpath.Remove(shellEnv(), dir)
	if err != nil {
		return ShellPathReceipt{Note: "the PATH line was not removed: " + displayText(err.Error())}
	}
	return ShellPathReceipt{Removed: r.Removed, Note: displayText(r.Note)}
}
