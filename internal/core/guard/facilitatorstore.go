package guard

import (
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Setting the facilitator's mode is the person's act (itd-2611252049031692):
// `abcd facilitator auto|human` writes the person's global setting, in the
// settings file in abcd's home, which decides wherever a project commits no
// mode; there is no setting of a person's for one project (the product
// owner's ruling of 2026-11-26). Each act is refused through the registry
// entry that names abcd, and under any other program name through a person's
// act of its own (personacts.go). A shell line that writes, moves or removes
// the file is refused here, and a file tool through FacilitatorStorePath.

const (
	// facilitatorAutoAnyEntryID and facilitatorHumanAnyEntryID are the reserved
	// ids the setting is refused under when the program running it is not one
	// abcd-facilitator-auto or abcd-facilitator-human names.
	facilitatorAutoAnyEntryID  = "abcd-facilitator-auto-any-program"
	facilitatorHumanAnyEntryID = "abcd-facilitator-human-any-program"

	// facilitatorStoreEntryID is the reserved id a shell write of the file is
	// refused under.
	facilitatorStoreEntryID = "facilitator-store-write"
)

// facilitatorSettingsFile is the person's settings file, directly in abcd's
// home folder, which holds their global facilitator.mode. The landing
// package's resolver reads it; a test there holds the name together, since
// this package cannot import it.
const facilitatorSettingsFile = "config.json"

// facilitatorStoreReason and facilitatorStoreSuccessor are what a refused write
// of the file says, on a shell line and through a file tool alike.
const (
	facilitatorStoreReason    = "This writes, moves or removes the person's own settings file, whose facilitator.mode decides whose approval a merge waits on wherever a project commits no mode: a mode an agent writes there is one no person chose."
	facilitatorStoreSuccessor = facilitatorSuccessor
)

// FacilitatorStorePath reports whether p is the person's settings file,
// directly in abcd's home folder, with case folded since the volume may fold
// it.
func FacilitatorStorePath(p string) bool {
	p = fsutil.FoldPath(filepath.Clean(p), true)
	return facilitatorStoreLeaf(filepath.Base(p)) && isHomeFolder(filepath.Dir(p))
}

// FacilitatorStoreRefusal is what a file tool refused FacilitatorStorePath is told.
func FacilitatorStoreRefusal() string {
	return facilitatorStoreReason + " " + facilitatorStoreSuccessor
}

// facilitatorStoreLeaf reports whether a folded name is the file's.
func facilitatorStoreLeaf(name string) bool {
	return name == facilitatorSettingsFile
}

// facilitatorStoreSignal refuses a line that writes, moves or removes the
// file: by its path, by a relative name on a line that names abcd's home
// folder, by a glob there that can match it, or by removing the folder
// itself. A line that does not name the home folder, and holds no glob, is not
// read further. A read is not refused.
func (r Registry) facilitatorStoreSignal(command string, rc *readCtx, v verdicts) (payloadSignal, bool) {
	if rc == nil {
		return payloadSignal{}, false
	}
	folded := fsutil.FoldPath(command, true)
	homeNamed := strings.Contains(folded, abcdhome.Rel())
	named := homeNamed && strings.Contains(folded, facilitatorSettingsFile)
	if !homeNamed && !strings.ContainsAny(command, "*?[") {
		return payloadSignal{}, false
	}
	for _, p := range placedLine(rc.line, v.segs) {
		for _, st := range p.states {
			for _, u := range pathUses(rc, p.seg, st) {
				if facilitatorStoreUse(u, named, homeNamed) {
					return payloadSignal{
						id:        facilitatorStoreEntryID,
						verdict:   VerdictBlock,
						family:    "facilitator",
						reason:    facilitatorStoreReason,
						successor: facilitatorStoreSuccessor,
					}, true
				}
			}
		}
	}
	return payloadSignal{}, false
}

// facilitatorStoreUse reports whether one write or removal reaches the file.
// named says the line names it, and homeNamed that it names abcd's home
// folder; a target the line does not fix is refused on a line that names it.
func facilitatorStoreUse(u pathUse, named, homeNamed bool) bool {
	switch {
	case u.glob != "" && (u.rel || u.ok):
		glob := fsutil.FoldPath(u.glob, true)
		return ((u.rel && homeNamed) || (u.ok && isHomeFolder(u.path))) &&
			shellGlobMatch(glob, facilitatorSettingsFile)
	case u.ok:
		return ((u.remove || u.bound) && isHomeFolder(u.path)) || FacilitatorStorePath(u.path)
	case u.rel:
		return homeNamed && facilitatorStoreLeaf(fsutil.FoldPath(filepath.Clean(u.written), true))
	}
	return named
}

// The setting's refusals, under abcd's name and any other, say one thing.
const (
	facilitatorAutoWhy   = "Setting the facilitator's mode to auto is the person's own act: under auto the merge gate lands a head no person approved, so an auto an agent sets hands it an approval no person gave."
	facilitatorHumanWhy  = "Setting the facilitator's mode is the person's own act: the setting decides whose approval a merge waits on, so a mode an agent sets is one no person chose."
	facilitatorSuccessor = "Leave this one to the person: tell them which mode you believe they should work under and why, and let them run `abcd facilitator auto` or `abcd facilitator human` from their own terminal; it sets their own mode wherever a project commits none."
)
