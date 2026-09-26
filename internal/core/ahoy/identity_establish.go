package ahoy

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/identity"
)

// TerminalPrompter is a Prompter that can say whether a person is answering at
// a terminal. A prompter that does not implement it is treated as having no one
// at a terminal, which is the fail-closed reading.
type TerminalPrompter interface {
	Prompter
	AtTerminal() bool
}

// atTerminal reports whether a person is answering p at a terminal.
func atTerminal(p Prompter) bool {
	tp, ok := p.(TerminalPrompter)
	return ok && tp.AtTerminal()
}

// establishGapIDs are the gaps a repo-local user.name/user.email can mend.
var establishGapIDs = []string{MismatchGapID, UnsetGapID, CommitterGapID, ToolIdentityGapID}

// stepGitIdentity is the itd-131 propose-and-confirm step. When the effective
// author or committer diverges, it proposes the human identity — the pin, else
// the global git identity, read from disk only — and writes repo-local
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
		a.refuse(did + ": there is no terminal to confirm at, so nothing was written (fail-closed); run abcd ahoy install at a terminal to be offered the pinned identity, or set user.name/user.email by hand. An autonomous routine's runner sets the human identity before its first commit (" + routineRunnerRecord + ")")
		return
	}
	prop, ok, err := identity.Propose(a.cwd)
	if err != nil {
		a.refuse(did + ": the identity to propose could not be read: " + errText(err))
		return
	}
	if !ok {
		a.refuse(did + ": there is no human identity to propose — neither a pin in " + identity.PinRelPath + " nor a global git user.name/user.email that is not a machine's; set them and run abcd ahoy install again")
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
