package guard

import (
	"strings"
	"sync"
)

// The merge rule (itd-2610150602547271, spc-2610161609350616 step 5). The
// merge gate in internal/core/landing decides whether abcd itself arms a
// merge; this is the hard rule beside it, on the commands themselves, so a
// path that forgets to ask the gate is still stopped:
//
//   - while facilitator.mode is human, `gh pr merge --auto`, any other
//     `gh pr merge` that merges (into a queue where one gates the default
//     branch, at once where none does; only `--disable-auto`, which withdraws
//     an arming, is not one), and a `gh api graphql` whose query is the
//     enqueuePullRequest or enablePullRequestAutoMerge mutation are refused,
//     naming the setting that would allow each;
//   - in any mode, the technical facilitator's approval act
//     (`abcd land --approve`, through the binary or `go run`) is refused: an
//     agent that records it writes a false record of who approved.
//
// The mode is a fact about the repository, not about the command's words, so
// no Pattern can say it, and the guard cannot import the resolver
// (internal/core/landing reads it through the layered reader, and both sit
// above this package). A front door hands it in (WithMergeMode), reading it
// through landing.ReadDefaultFacilitator, the resolver the gate uses: the
// default branch's committed .abcd/config.json and the machine layer. A
// registry with no resolver handed in (the bundled defaults on their own)
// cannot tell the mode and judges no merge command; the approval act is
// refused whatever the registry knows. The `abcd` binary's own spelling of the
// approval act is the registry entry abcd-land-approve, which also teaches it;
// the `go run` spelling no Pattern can name is raised here.

const (
	// mergeAutoEntryID, mergeNowEntryID and mergeEnqueueEntryID are the
	// reserved ids the merge commands are refused under while the mode is
	// human.
	mergeAutoEntryID    = "merge-auto-human"
	mergeNowEntryID     = "merge-now-human"
	mergeEnqueueEntryID = "merge-enqueue-human"

	// approveGoRunEntryID is the reserved id the approval act run through
	// `go run` is refused under, in any mode.
	approveGoRunEntryID = "abcd-land-approve-go-run"

	familyMerge = "merge"

	// modeSetting names the setting that allows the merge commands.
	modeSetting = "facilitator.mode: auto in the default branch's committed .abcd/config.json, with no facilitator.mode: human in ~/.abcd.noindex/config.json"

	// mergeSuccessor is what to do instead of a merge command while the mode
	// is human.
	mergeSuccessor = "Leave the merge to the technical facilitator: say the pull request is ready for their approval. Once they approve its head (a forge review, or `abcd land --approve` run by them), `abcd land` arms it."
)

// MergeMode reports whether the repository's facilitator mode is human, and
// how it reads ("facilitator.mode is human (repo layer, …)"), for the
// refusal. A mode that cannot be read is reported human, with why.
type MergeMode func() (human bool, reading string)

// mergeGate is a registry's handed-in mode, read the first time a merge
// command asks, so a line with no merge command in it costs no read.
type mergeGate struct {
	read    MergeMode
	once    sync.Once
	human   bool
	reading string
}

func (g *mergeGate) mode() (bool, string) {
	g.once.Do(func() { g.human, g.reading = g.read() })
	return g.human, g.reading
}

// WithMergeMode returns r judging the merge commands by the mode m reports.
func (r Registry) WithMergeMode(m MergeMode) Registry {
	if m != nil {
		r.merge = &mergeGate{read: m}
	}
	return r
}

// literalWords are the words of tokens the line fixes, with "" for a word it
// does not (a variable, a substitution whose output is unknown).
func literalWords(s segment, from int) []string {
	var out []string
	for i := from; i < len(s.tokens); i++ {
		w, fixed, ok := forgeWord(s, i)
		if !ok || !fixed {
			w = ""
		}
		out = append(out, w)
	}
	return out
}

// boolFlagOn reports whether word sets the boolean flag name: `--name`, or
// `--name=` any value but false.
func boolFlagOn(word, name string) bool {
	if word == name {
		return true
	}
	v, ok := strings.CutPrefix(word, name+"=")
	return ok && !strings.EqualFold(v, "false")
}

// mergeSignals are the merge rule's refusals for the segments of one reading.
func (r Registry) mergeSignals(segs []segment) []payloadSignal {
	var out []payloadSignal
	add := func(sig payloadSignal) {
		if !containsSignal(out, sig) {
			out = append(out, sig)
		}
	}
	for _, s := range segs {
		if approveGoRun(s) {
			add(payloadSignal{
				id: approveGoRunEntryID, verdict: VerdictBlock, family: familyMerge,
				reason:    "Recording the technical facilitator's approval of a head is the person's own act: the merge gate arms what it approves, so an approval an agent records is a false record of who approved, whatever facilitator.mode says.",
				successor: "Leave this one to the person: tell them the head you believe is ready and why, and let them run `abcd land --approve` themselves.",
			})
		}
	}
	if r.merge == nil {
		return out
	}
	for _, s := range segs {
		for _, kind := range mergeCommands(s) {
			human, reading := r.merge.mode()
			if !human {
				break
			}
			sig := payloadSignal{verdict: VerdictBlock, family: familyMerge, successor: mergeSuccessor}
			switch kind {
			case mergeAutoEntryID:
				sig.id = kind
				sig.reason = reading + ", so `gh pr merge --auto` would arm a merge the technical facilitator has not approved. The setting that allows it is " + modeSetting + "."
			case mergeNowEntryID:
				sig.id = kind
				sig.reason = reading + ", so `gh pr merge` would merge (into the merge queue where one gates the default branch, at once where none does) what the technical facilitator has not approved. The setting that allows it is " + modeSetting + "; only `gh pr merge --disable-auto` is not a merge."
			case mergeEnqueueEntryID:
				sig.id = kind
				sig.reason = reading + ", so the GraphQL enqueuePullRequest or enablePullRequestAutoMerge mutation would queue or arm a merge the technical facilitator has not approved. The setting that allows it is " + modeSetting + "."
			}
			add(sig)
		}
	}
	return out
}

// ghValueFlags are gh's global options that take a value, so the word after
// `-R` is not read as the command group.
var ghValueFlags = []string{"-R", "--repo"}

// mergeCommands are the merge commands segment s runs, each by the reserved
// id it is refused under.
func mergeCommands(s segment) []string {
	var out []string
	for _, a := range sitesNamed(s, "gh") {
		args := s.tokens[a.idx+1:]
		readings, _ := operandReadings(args, ghValueFlags, 2)
		for _, idx := range readings {
			if len(idx) < 2 {
				continue
			}
			group, _, ok1 := forgeWord(s, a.idx+1+idx[0])
			verb, _, ok2 := forgeWord(s, a.idx+1+idx[1])
			if !ok1 || !ok2 {
				continue
			}
			rest := literalWords(s, a.idx+1+idx[1]+1)
			switch {
			case group == "pr" && verb == "merge":
				auto, disable := false, false
				for _, w := range rest {
					auto = auto || boolFlagOn(w, "--auto")
					disable = disable || boolFlagOn(w, "--disable-auto")
				}
				switch {
				case auto:
					out = append(out, mergeAutoEntryID)
				case !disable:
					out = append(out, mergeNowEntryID)
				}
			case group == "api" && verb == "graphql":
				for _, w := range rest {
					if strings.Contains(w, "enqueuePullRequest") || strings.Contains(w, "enablePullRequestAutoMerge") {
						out = append(out, mergeEnqueueEntryID)
						break
					}
				}
			default:
				continue
			}
			break
		}
	}
	return out
}

// approveGoRun reports whether segment s runs `abcd land --approve` through
// `go run`: a go command whose first operand is run, whose package word names
// abcd's command (cmd/abcd), and whose arguments after it name the land verb
// and set --approve.
func approveGoRun(s segment) bool {
	for _, a := range sitesNamed(s, "go") {
		words := literalWords(s, a.idx+1)
		i := 0
		for i < len(words) && strings.HasPrefix(words[i], "-") {
			i++
		}
		if i >= len(words) || words[i] != "run" {
			continue
		}
		pkg := -1
		for j := i + 1; j < len(words); j++ {
			w := strings.TrimSuffix(words[j], "/")
			if w == "cmd/abcd" || strings.HasSuffix(w, "/cmd/abcd") || strings.Contains(w, "/cmd/abcd@") {
				pkg = j
				break
			}
		}
		if pkg < 0 {
			continue
		}
		land, approve := false, false
		for _, w := range words[pkg+1:] {
			land = land || w == "land"
			approve = approve || boolFlagOn(w, "--approve")
		}
		if land && approve {
			return true
		}
	}
	return false
}
