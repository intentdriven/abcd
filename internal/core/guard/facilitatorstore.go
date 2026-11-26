package guard

import (
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Setting the facilitator's mode is the person's act (itd-2611252049031692):
// `abcd facilitator auto|human` writes their own entry for one project, and
// with --everywhere the computer's file. Each act is refused through the
// registry entry that names abcd, and under any other program name through a
// person's act of its own (personacts.go). A shell line that writes, moves or
// removes either file is refused here, and a file tool through
// FacilitatorStorePath.

const (
	// facilitatorAutoAnyEntryID and facilitatorHumanAnyEntryID are the reserved
	// ids the setting is refused under when the program running it is not one
	// abcd-facilitator-auto or abcd-facilitator-human names.
	facilitatorAutoAnyEntryID  = "abcd-facilitator-auto-any-program"
	facilitatorHumanAnyEntryID = "abcd-facilitator-human-any-program"

	// facilitatorStoreEntryID is the reserved id a shell write of either file
	// is refused under.
	facilitatorStoreEntryID = "facilitator-store-write"
)

// facilitatorPersonFile is the person's own entries, and facilitatorMachineFile
// the computer's settings file, each directly in abcd's home folder. The
// landing package's resolver reads them; a test there holds the names together,
// since this package cannot import it.
const (
	facilitatorPersonFile  = "facilitator.json"
	facilitatorMachineFile = "config.json"
)

// facilitatorStoreReason and facilitatorStoreSuccessor are what a refused write
// of either file says, on a shell line and through a file tool alike.
const (
	facilitatorStoreReason    = "This writes, moves or removes the person's own facilitator entries or the computer's settings file, which decide whose approval a merge waits on: a mode an agent writes there is one no person chose."
	facilitatorStoreSuccessor = "Leave this one to the person: tell them which mode you believe this project should run under and why, and let them run `abcd facilitator auto` or `abcd facilitator human` from their own terminal."
)

// FacilitatorStorePath reports whether p is the person's facilitator entries or
// the computer's settings file, directly in abcd's home folder, with case folded
// since the volume may fold it.
func FacilitatorStorePath(p string) bool {
	p = fsutil.FoldPath(filepath.Clean(p), true)
	return facilitatorStoreLeaf(filepath.Base(p)) && isHomeFolder(filepath.Dir(p))
}

// FacilitatorStoreRefusal is what a file tool refused FacilitatorStorePath is told.
func FacilitatorStoreRefusal() string {
	return facilitatorStoreReason + " " + facilitatorStoreSuccessor
}

// facilitatorStoreLeaf reports whether a folded name is either file's.
func facilitatorStoreLeaf(name string) bool {
	return name == facilitatorPersonFile || name == facilitatorMachineFile
}

// facilitatorStoreSignal refuses a line that writes, moves or removes either
// file: by its path, by a relative name on a line that names abcd's home folder,
// by a glob there that can match either, or by removing the folder itself. A
// line that names neither the home folder nor the person's file, and holds no
// glob, is not read further. A read is not refused.
func (r Registry) facilitatorStoreSignal(command string, rc *readCtx, v verdicts) (payloadSignal, bool) {
	if rc == nil {
		return payloadSignal{}, false
	}
	folded := fsutil.FoldPath(command, true)
	homeNamed := strings.Contains(folded, abcdhome.Rel())
	named := strings.Contains(folded, facilitatorPersonFile) || (homeNamed && strings.Contains(folded, facilitatorMachineFile))
	if !named && !homeNamed && !strings.ContainsAny(command, "*?[") {
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

// facilitatorStoreUse reports whether one write or removal reaches either file.
// named says the line names one of them, and homeNamed that it names abcd's home
// folder; a target the line does not fix is refused on a line that names one.
func facilitatorStoreUse(u pathUse, named, homeNamed bool) bool {
	switch {
	case u.glob != "" && (u.rel || u.ok):
		glob := fsutil.FoldPath(u.glob, true)
		return ((u.rel && homeNamed) || (u.ok && isHomeFolder(u.path))) &&
			(shellGlobMatch(glob, facilitatorPersonFile) || shellGlobMatch(glob, facilitatorMachineFile))
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
	facilitatorSuccessor = "Leave this one to the person: tell them which mode you believe this project should run under and why, and let them run `abcd facilitator auto` or `abcd facilitator human` from their own terminal."
)
