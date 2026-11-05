package guard

import "strings"

// The acts only the person takes are refused through the binary their
// registry entries name, and abcd also runs under other names: go run of the
// module, a cross-compiled build (bin/abcd-<os>-<arch>), the source-checkout
// wrapper, a copy renamed, a program a variable names. So each act is refused
// as well under any program whose words carry the act's verbs and flags, under
// a reserved id of its own, giving the reason and successor of the entry for
// the same act (TestEachPersonsActSaysWhatItsRegistryEntrySays holds the two
// together).

const (
	// approveAnyEntryID is the reserved id the approval act is refused under,
	// in any mode, when the program running it is not one the registry entry
	// abcd-land-approve names: `go run .`, or abcd built under another name.
	approveAnyEntryID = "abcd-land-approve-any-program"

	// guardAllowAnyEntryID is the reserved id `guard allow` is refused under
	// when the program running it is not one abcd-guard-allow names.
	guardAllowAnyEntryID = "abcd-guard-allow-any-program"

	// ledgerFlipAnyEntryID and ledgerFlipRetiredAnyEntryID are the reserved
	// ids the source-ledger flip is refused under, in the library spelling
	// and the retired source one, when the program is not one
	// abcd-source-ledger-flip or its retired twin names.
	ledgerFlipAnyEntryID        = "abcd-source-ledger-flip-any-program"
	ledgerFlipRetiredAnyEntryID = "abcd-source-ledger-flip-retired-any-program"

	// implementStopAnyEntryID and sharedFolderAnyEntryID are the reserved ids
	// a stop and the folder it is read from are refused under when the program
	// is not one abcd-implement-stop or abcd-ahoy-shared-folder names
	// (itd-2611041549471712).
	implementStopAnyEntryID = "abcd-implement-stop-any-program"
	sharedFolderAnyEntryID  = "abcd-ahoy-shared-folder-any-program"
)

// personAct is one act only the person takes: the verbs that name it, in
// order, and the flags it needs set, as cobra reads them wherever they stand.
// A boolean flag (flags) is set only by a word the line fixes, so a flag a
// substitution prints is not taken as set; a flag that takes a value
// (valueFlags) is read as the registry entry reads it, fail-closed, so
// `--flip=$n` and an unknown dash-word that can become the flag count as set.
type personAct struct {
	id, entry, family string
	verbs, flags      []string
	valueFlags        []string
	// adjacent holds the verbs to consecutive words: an act no flag names, whose
	// two common words would otherwise match a search that merely mentions them.
	adjacent       bool
	why, successor string
}

var personActs = []personAct{
	{
		id: approveAnyEntryID, entry: "abcd-land-approve", family: familyMerge,
		verbs: []string{"land"}, flags: []string{"--approve"},
		why:       "Recording the technical facilitator's approval of a head is the person's own act: the merge gate arms what it approves, so an approval an agent records is a false record of who approved, whatever facilitator.mode says.",
		successor: "Leave this one to the person: tell them the head you believe is ready and why, and let them run `abcd land --approve` themselves.",
	},
	{
		id: guardAllowAnyEntryID, entry: "abcd-guard-allow", family: "guard allow",
		verbs: []string{"guard", "allow"}, flags: []string{"--once"},
		why:       "Allowing a delete the guard could not prove safe is the person's own act: the allow lets one exact command through that may take work with it, so an allow an agent grants itself is one no person decided.",
		successor: "Leave this one to the person: tell them the exact command, what the guard could not prove and why you believe it is safe, and let them run `abcd guard allow --once --command '<the command>'` themselves.",
	},
	{
		id: ledgerFlipAnyEntryID, entry: "abcd-source-ledger-flip", family: "library ledger",
		verbs: []string{"library", "ledger"}, valueFlags: []string{"--flip"},
		why:       "Flipping a source-ledger line to public citation is the person's own act: the ledger records that a human chose to cite that source, so a flip an agent makes writes a false record of who decided.",
		successor: "Leave this one to the person: tell them which ledger line you believe is ready to cite publicly and why, and let them run `abcd library ledger --flip N` themselves.",
	},
	{
		id: ledgerFlipRetiredAnyEntryID, entry: "abcd-source-ledger-flip-retired", family: "source ledger",
		verbs: []string{"source", "ledger"}, valueFlags: []string{"--flip"},
		why:       "The retired spelling of the ledger flip: a binary older than the library verb still answers it, and such a binary is often reachable (the plugin root's copy, an older build). Flipping a source-ledger line to public citation is the person's own act, so a flip an agent makes writes a false record of who decided.",
		successor: "Leave this one to the person: tell them which ledger line you believe is ready to cite publicly and why, and let them run `abcd library ledger --flip N` themselves.",
	},
	{
		id: implementStopAnyEntryID, entry: "abcd-implement-stop", family: "implement stop",
		verbs: []string{"implement", "stop"}, adjacent: true,
		why:       "Setting or lifting a stop is the person's own act: a stop withdraws what the unattended runs that read it may do, and lifting it is their word to give, so one an agent sets or lifts is one no person decided.",
		successor: "Leave this one to the person: tell them why you believe the runs should stop or go on, and let them run `abcd implement stop`, or `abcd implement stop --lift`, from their own account.",
	},
	{
		id: sharedFolderAnyEntryID, entry: "abcd-ahoy-shared-folder", family: "ahoy shared-folder",
		verbs: []string{"ahoy", "shared-folder"}, adjacent: true,
		why:       "Choosing the shared folder is the person's own act: it decides where this account reads a stop from, so a choice an agent makes can put every run out of reach of the person's stop.",
		successor: "Leave this one to the person: tell them what the folder is for and let them run `abcd ahoy shared-folder <folder>` themselves, in each account.",
	},
}

// actOf reports whether segment s runs act a under a program other than one
// called abcd, which the registry entry judges. What the program is called is
// not read, since a build is runnable under any name; the words it is handed
// are, the verbs in order and the flags anywhere. The verbs need not stand
// side by side, since a flag's value may sit between them; a line whose
// words carry them in order with other words between is refused too, the
// over-blocking direction the guard takes where the reading is a guess, and
// the flag each act needs keeps that to lines that name the act.
func (a personAct) actOf(s segment) bool {
	for _, site := range commandSites(s) {
		if commandNamed(s, site, "abcd") {
			continue
		}
		next := 0
		on := make([]bool, len(a.flags))
		valued := make([]bool, len(a.valueFlags))
		skip := false
		for j, w := range literalWords(s, site.idx+1) {
			if a.adjacent {
				// The words the command line's parser passes over on its way to a
				// sub-command: --x=y alone, and --x with the word after it, but for
				// the flags that take no word. A "--" ends its way.
				if skip {
					skip = false
					continue
				}
				if w == "--" {
					break
				}
				if strings.HasPrefix(w, "-") && w != "-" {
					skip = !strings.Contains(w, "=") && w != "--json" && w != "--help" && w != "-h"
					continue
				}
			}
			switch {
			case next < len(a.verbs) && w == a.verbs[next]:
				next++
			case a.adjacent && next > 0 && next < len(a.verbs):
				// The run is broken: the verbs must start again.
				next = 0
				if w == a.verbs[0] {
					next = 1
				}
			}
			for i, f := range a.flags {
				on[i] = on[i] || boolFlagOn(w, f)
			}
			at := site.idx + 1 + j
			tok := s.tokens[at]
			for i, f := range a.valueFlags {
				valued[i] = valued[i] || flagMatches(f, knownText(tok), s.globAt(at)) || unknownFlagCouldBe(tok, f)
			}
		}
		if next == len(a.verbs) && allTrue(on) && allTrue(valued) {
			return true
		}
	}
	return false
}

func allTrue(bs []bool) bool {
	for _, b := range bs {
		if !b {
			return false
		}
	}
	return true
}

// personActSignals are the refusals of the person's acts under any program,
// for the segments of one reading.
func personActSignals(segs []segment) []payloadSignal {
	var out []payloadSignal
	for _, s := range segs {
		for _, a := range personActs {
			if !a.actOf(s) {
				continue
			}
			sig := payloadSignal{id: a.id, verdict: VerdictBlock, family: a.family, reason: a.why, successor: a.successor}
			if !containsSignal(out, sig) {
				out = append(out, sig)
			}
		}
	}
	return out
}
