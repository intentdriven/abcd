package ahoy

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/provenance"
)

// MachineIdentity (itd-2610240918547728) covers the person's own commit
// identity record on their machine, and moving the identity pin a repository
// still stores to its owner's machine. Its gaps are advisory, never written
// under --yes, and never behind a category approval: each is asked as its own
// question, only at a terminal, because it records who a person is.
const MachineIdentity GapCategory = "machine-identity"

// legacyPinQuestionPrefix opens the migration question, so the question's id
// (setupConfirmIDs) is matched on the words that ask it.
const legacyPinQuestionPrefix = "This repository stores your commit identity, "

// The two gaps the person's own identity record raises.
const (
	// UnrecordedGapID: no record on this machine applies to the repository.
	UnrecordedGapID = "git_identity.unrecorded"
	// LegacyPinGapID: the repository still stores the identity pin itd-131
	// committed (.abcd/config/identity.json).
	LegacyPinGapID = "git_identity.repo_pin"
)

// TerminalPrompter is a Prompter that can say whether a person is answering at
// a terminal. A prompter that does not implement it is treated as having no one
// at a terminal, which is the fail-closed reading.
type TerminalPrompter interface {
	Prompter
	AtTerminal() bool
}

// HelpedPrompter is a Prompter that can ask a value question with help built
// for this run: the question's current setting and an example of what an
// answer writes here, which the static help cannot hold (PromptHelp's Now,
// Example and ExampleNone). The README pair asks through it where the
// prompter implements it, and through Prompt, by the help's key, where it
// does not (spc-2610031704338417).
type HelpedPrompter interface {
	Prompter
	PromptHelped(h PromptHelp, choices []string, def string) string
}

// promptHelped asks the value question h through p's PromptHelped where p
// has one, else through Prompt by h's key.
func promptHelped(p Prompter, h PromptHelp, choices []string, def string) string {
	if hp, ok := p.(HelpedPrompter); ok {
		return hp.PromptHelped(h, choices, def)
	}
	return p.Prompt(h.Key, choices, def)
}

// atTerminal reports whether a person is answering p at a terminal.
func atTerminal(p Prompter) bool {
	tp, ok := p.(TerminalPrompter)
	return ok && tp.AtTerminal()
}

// stepIdentityMigrate moves the identity pin a repository still stores to its
// owner's machine and out of the repository (criterion 4 of
// itd-2610240918547728), keeping its production_mode in the repository's own
// committed file. It asks first, as its own question, and only at a
// terminal; under --yes or off a terminal it writes nothing, and
// optionalSkipped reports that. On anyone else's machine the gap is not
// resolvable and this step never runs: the pin is neither copied nor removed.
func (a *applyCtx) stepIdentityMigrate() {
	if !a.has(LegacyPinGapID) || a.autoYes || !atTerminal(a.prompter) {
		return
	}
	home := userHome()
	plan, found, err := identity.PlanMigration(a.cwd, home)
	if err != nil {
		a.refuse("did not move the identity this repository stores: " + errText(err))
		return
	}
	if !found || !plan.Owner || plan.Blocker != "" {
		return
	}
	who := plan.Pin.Name + " <" + plan.Pin.Email + ">"
	q := legacyPinQuestionPrefix + who + ", in " + identity.LegacyPinRelPath +
		", which tells every other contributor to commit as you. Move it to your own machine (" + plan.Target.Display +
		") and remove it from the repository?"
	if plan.Pin.ProductionMode != "" {
		q += " (its production_mode stays committed, in " + provenance.DeclaredModeRelPath + ")"
	}
	if !a.prompter.Confirm(q) {
		a.refuse("left " + identity.LegacyPinRelPath + " as it is: the move was declined. abcd no longer reads it to check commits; run abcd ahoy install at a terminal to be asked again")
		return
	}
	res, err := identity.ApplyMigration(a.cwd, home, plan)
	if res.ProductionModeMoved {
		a.note(writeProductionMode, provenance.DeclaredModeRelPath)
	}
	if res.Recorded {
		a.note(writeIdentityRecord, res.RecordAt.Display)
	}
	if err != nil {
		a.refuse("did not finish moving the identity this repository stores: " + errText(err))
		return
	}
	if res.KeptDifferent {
		a.refuse("kept the identity you had already recorded in " + res.RecordAt.Display + "; the one " + identity.LegacyPinRelPath + " stored was not copied over it")
	}
	if res.Removed {
		a.note(writeIdentityPinRemoved, identity.LegacyPinRelPath)
	}
}

// stepIdentityRecord offers a person who has not recorded who they commit as
// on this machine their usual git identity, and records it in the
// machine-wide record on a yes (criterion 1 of itd-2610240918547728). It is
// its own question, asked only at a terminal. Under --yes or off a terminal
// it records nothing, and optionalSkipped reports that: a blanket approval or
// an unattended run never decides who a person is.
func (a *applyCtx) stepIdentityRecord() {
	if !a.has(UnrecordedGapID) || a.autoYes || !atTerminal(a.prompter) {
		return
	}
	const did = "did not record who you commit as on this machine"
	// The move above may have just recorded one.
	if res, err := identity.Check(a.cwd); err == nil && res.Recorded {
		return
	}
	prop, ok, err := identity.ProposeRecord(a.cwd)
	if err != nil {
		a.refuse(did + ": your git identity could not be read: " + errText(err))
		return
	}
	if !ok {
		a.refuse(did + ": git holds no human identity to propose; set git config --global user.name and user.email to your own and run abcd ahoy install again")
		return
	}
	ref := identity.MachineRecordRef()
	who := prop.Identity.Name + " <" + prop.Identity.Email + ">"
	if !a.prompter.Confirm("Record " + who + ", " + prop.From + ", as who you commit as on this machine? (writes " + ref.Display +
		"; a commit made under any other identity is then refused until you change it)") {
		a.refuse(did + ": " + who + " was declined; run abcd ahoy install at a terminal to be asked again")
		return
	}
	if err := identity.WriteRecord(userHome(), ref, identity.Record{Name: prop.Identity.Name, Email: prop.Identity.Email}); err != nil {
		a.refuse(did + ": " + errText(err))
		return
	}
	a.note(writeIdentityRecord, ref.Display)
}

// establishGapIDs are the gaps a repo-local user.name/user.email can mend.
var establishGapIDs = []string{MismatchGapID, UnsetGapID, CommitterGapID, ToolIdentityGapID}

// stepGitIdentity is the itd-131 propose-and-confirm step. When the effective
// author or committer diverges, it proposes the human identity — the person's
// own record, else their global git identity, read from disk only and never
// from the repository — and writes repo-local
// user.name/user.email only when the person confirms that proposal at a
// terminal.
//
// Every other path writes nothing and says why:
//   - under --yes: a blanket approval never changes who commits;
//   - with no terminal (a pipe, a routine, CI): the run is fail-closed and never
//     asks, so it can neither write unprompted nor block on a question no one
//     can answer — establishing a routine's identity is its runner's job
//     (iss-2608210932052003);
//   - when an environment override or an author.*/committer.* key outranks
//     user.*: writing user.* would change nothing, so it names what to unset;
//   - when there is no human identity on disk to propose.
//
// The write is a plain `git config --local`, as the person running it; the step
// never escalates privileges and never touches global or system config.
func (a *applyCtx) stepGitIdentity() {
	if !a.anyGap(establishGapIDs) || !a.approved[ConfigChange] {
		return
	}
	const did = "did not change who commits to this repository"
	if a.autoYes {
		a.refuse(did + ": --yes never rewrites the git identity; run abcd ahoy install at a terminal to be asked")
		return
	}
	if !atTerminal(a.prompter) {
		a.refuse(did + ": there is no terminal to confirm at, so nothing was written (fail-closed); run abcd ahoy install at a terminal to be offered your own identity, or set user.name/user.email by hand. An autonomous routine's runner sets the human identity before its first commit (" + routineRunnerRecord + ")")
		return
	}
	prop, ok, err := identity.Propose(a.cwd)
	if err != nil {
		a.refuse(did + ": the identity to propose could not be read: " + errText(err))
		return
	}
	if !ok {
		a.refuse(did + ": there is no human identity to propose — neither a record of your own in " + identity.MachineRecordRef().Display + " nor a global git user.name/user.email that is not a machine's; set them and run abcd ahoy install again")
		return
	}
	over, err := identity.Outranking(a.cwd, prop.Identity)
	if err != nil {
		a.refuse(did + ": git config could not be read: " + errText(err))
		return
	}
	if len(over) > 0 {
		a.refuse(did + ": " + strings.Join(over, ", ") + " outranks user.name/user.email, so writing them would change nothing; unset it and run abcd ahoy install again")
		return
	}
	who := prop.Identity.Name + " <" + prop.Identity.Email + ">"
	if !a.prompter.Confirm("Commit to this repository as " + who + ", " + prop.From + "? (sets user.name and user.email in this repository's .git/config only)") {
		a.refuse(did + ": the proposed identity " + who + " was declined; git config is unchanged")
		return
	}
	if err := identity.WriteLocal(a.cwd, prop.Identity); err != nil {
		a.refuse(did + ": " + errText(err))
		return
	}
	a.note(writeGitIdentity, ".git/config")
}

// anyGap reports whether any of ids is a present gap.
func (a *applyCtx) anyGap(ids []string) bool {
	for _, id := range ids {
		if a.has(id) {
			return true
		}
	}
	return false
}
