package guard

import (
	"regexp"
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

	// mergeRESTEntryID is the REST merge route (`gh api -X PUT
	// repos/{owner}/{repo}/pulls/N/merge`), and mergeUnreadEntryID a
	// `gh api graphql` call whose query is not on the line, so it could be a
	// merge mutation; both are refused while the mode is human.
	mergeRESTEntryID   = "merge-rest-human"
	mergeUnreadEntryID = "merge-graphql-unread-human"

	// approveAnyEntryID is the reserved id the approval act is refused under,
	// in any mode, when the program running it is not one the registry entry
	// abcd-land-approve names: `go run`, or abcd built under another name.
	approveAnyEntryID = "abcd-land-approve-any-program"

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
		if approveAct(s) {
			add(payloadSignal{
				id: approveAnyEntryID, verdict: VerdictBlock, family: familyMerge,
				reason:    "Recording the technical facilitator's approval of a head is the person's own act: the merge gate arms what it approves, so an approval an agent records is a false record of who approved, whatever facilitator.mode says.",
				successor: "Leave this one to the person: tell them the head you believe is ready and why, and let them run `abcd land --approve` themselves.",
			})
		}
	}
	if r.merge == nil {
		return out
	}
	for _, s := range segs {
		for _, hit := range mergeCommands(s) {
			human, reading := r.merge.mode()
			if hit.named && !human {
				human, reading = true, "the command names its repository, whose facilitator.mode is not read here (this checkout's is), so it counts as human"
			}
			if !human {
				continue
			}
			sig := payloadSignal{verdict: VerdictBlock, family: familyMerge, successor: mergeSuccessor}
			switch kind := hit.id; kind {
			case mergeAutoEntryID:
				sig.id = kind
				sig.reason = reading + ", so `gh pr merge --auto` would arm a merge the technical facilitator has not approved. The setting that allows it is " + modeSetting + "."
			case mergeNowEntryID:
				sig.id = kind
				sig.reason = reading + ", so `gh pr merge` would merge (into the merge queue where one gates the default branch, at once where none does) what the technical facilitator has not approved. The setting that allows it is " + modeSetting + "; only `gh pr merge --disable-auto` is not a merge."
			case mergeEnqueueEntryID:
				sig.id = kind
				sig.reason = reading + ", so the GraphQL enqueuePullRequest, enablePullRequestAutoMerge or mergePullRequest mutation would queue, arm or make a merge the technical facilitator has not approved. The setting that allows it is " + modeSetting + "."
			case mergeRESTEntryID:
				sig.id = kind
				sig.reason = reading + ", so a PUT to the pull request's REST merge route would merge what the technical facilitator has not approved. The setting that allows it is " + modeSetting + "."
			case mergeUnreadEntryID:
				sig.id = kind
				sig.reason = reading + ", and this GraphQL call's query is not on the line (read from a file, from --input, or printed by a substitution), so it could queue, arm or make a merge the technical facilitator has not approved. Put the query on the line to have it read. The setting that allows every merge is " + modeSetting + "."
			}
			add(sig)
		}
	}
	return out
}

// mergeHit is one merge command a segment runs: the reserved id it is refused
// under, and whether it names its own repository (-R, --repo, a GH_REPO
// prefix, or a REST path that spells owner and name), whose mode the one read
// here, this checkout's, does not answer for.
type mergeHit struct {
	id    string
	named bool
}

// mergePathRe is the REST route that merges a pull request,
// `repos/{owner}/{repo}/pulls/{number}/merge`, with or without its leading
// slash, a host before it or a query after it.
var mergePathRe = regexp.MustCompile(`(?i)(^|/)repos/([^/]+)/([^/]+)/pulls/[^/]+/merge/?([?#].*)?$`)

// mergeMutations are the GraphQL mutations that queue, arm or make a merge.
var mergeMutations = []string{"enqueuePullRequest", "enablePullRequestAutoMerge", "mergePullRequest"}

// ghFieldFlags are gh api's options that add a field, whose value word may
// carry the GraphQL query (`query=…`, or `query=@file`, which gh reads).
var ghFieldFlags = []string{"-f", "-F", "--field", "--raw-field"}

// mergeCommands are the merge commands segment s runs.
func mergeCommands(s segment) []mergeHit {
	var out []mergeHit
	for _, a := range sitesNamed(s, "gh") {
		args := s.tokens[a.idx+1:]
		readings, _ := operandReadings(args, ghRepoFlags, 2)
		envRepo := false
		for _, w := range literalWords(s, 0)[:a.idx] {
			envRepo = envRepo || strings.HasPrefix(w, "GH_REPO=")
		}
		for _, idx := range readings {
			if len(idx) < 2 {
				continue
			}
			group, _, ok1 := forgeWord(s, a.idx+1+idx[0])
			verb, _, ok2 := forgeWord(s, a.idx+1+idx[1])
			if !ok1 || !ok2 {
				continue
			}
			all := literalWords(s, a.idx+1)
			rest := literalWords(s, a.idx+1+idx[1]+1)
			named := envRepo
			for _, w := range all {
				named = named || w == "-R" || w == "--repo" || strings.HasPrefix(w, "--repo=") || (strings.HasPrefix(w, "-R") && len(w) > 2)
			}
			switch {
			case group == "pr" && verb == "merge":
				auto, disable := false, false
				for _, w := range rest {
					auto = auto || boolFlagOn(w, "--auto")
					disable = disable || boolFlagOn(w, "--disable-auto")
				}
				switch {
				case auto:
					out = append(out, mergeHit{mergeAutoEntryID, named})
				case !disable:
					out = append(out, mergeHit{mergeNowEntryID, named})
				}
			case group == "api" && verb == "graphql":
				if hit, ok := graphqlMerge(s, a.idx+1+idx[1]+1); ok {
					out = append(out, mergeHit{hit, named})
				}
			case group == "api":
				if restMerge(all) {
					out = append(out, mergeHit{mergeRESTEntryID, named || restNamesRepo(all)})
				}
			default:
				continue
			}
			break
		}
	}
	return out
}

// graphqlMerge reads a `gh api graphql` call's words from `from` on: the
// enqueue id where its query is a merge mutation, and the unread id where the
// query is not on the line (a file gh reads, `--input`, a value a
// substitution prints), since that query could be one.
func graphqlMerge(s segment, from int) (string, bool) {
	// unread is a query the line points away from; unknown is a field the
	// line does not fix, which may be the query unless a fixed query= stood.
	unread, unknown, query := false, false, false
	for i := from; i < len(s.tokens); i++ {
		w, fixed, ok := forgeWord(s, i)
		for _, m := range mergeMutations {
			if ok && strings.Contains(w, m) {
				return mergeEnqueueEntryID, true
			}
		}
		if ok && fixed && (w == "--input" || strings.HasPrefix(w, "--input=")) {
			unread = true
		}
		v, vfixed, vok, isField := fieldValue(s, i)
		switch {
		case !isField:
		case vok && strings.HasPrefix(v, "query=@"):
			unread = true
		case vok && vfixed && strings.HasPrefix(v, "query="):
			query = true
		case !vok || !vfixed:
			unknown = true
		}
	}
	if unread || (unknown && !query) {
		return mergeUnreadEntryID, true
	}
	return "", false
}

// fieldValue reports whether the word at i sets one of ghFieldFlags, and
// the field it sets: the next word's text, or this word's own after the flag
// for the joined forms (`-fquery=…`, `--field=query=…`). vok is false for
// a value the line does not fix at all; vfixed is false for one it fixes only
// in part.
func fieldValue(s segment, i int) (text string, vfixed, vok, isField bool) {
	w, fixed, ok := forgeWord(s, i)
	if !ok || !fixed {
		return "", false, false, false
	}
	for _, f := range ghFieldFlags {
		if w == f {
			if i+1 >= len(s.tokens) {
				return "", false, false, false
			}
			text, vfixed, vok = forgeWord(s, i+1)
			return text, vfixed, vok, true
		}
		joined := f
		if strings.HasPrefix(f, "--") {
			joined += "="
		}
		if v, cut := strings.CutPrefix(w, joined); cut {
			return v, true, true, true
		}
	}
	return "", false, false, false
}

// restMerge reports whether a `gh api` call's words name the REST merge route
// with the PUT method: `-X PUT`, `-XPUT`, `--method PUT` or `--method=PUT`,
// any case. A method word the line does not fix counts as PUT.
func restMerge(words []string) bool {
	route := false
	for _, w := range words {
		route = route || mergePathRe.MatchString(w)
	}
	if !route {
		return false
	}
	for i, w := range words {
		var m string
		switch {
		case w == "-X" || w == "--method":
			if i+1 >= len(words) {
				continue
			}
			m = words[i+1]
			if m == "" {
				return true
			}
		case strings.HasPrefix(w, "--method="):
			m = strings.TrimPrefix(w, "--method=")
		case strings.HasPrefix(w, "-X") && len(w) > 2:
			m = strings.TrimPrefix(w, "-X")
		default:
			continue
		}
		if strings.EqualFold(m, "PUT") {
			return true
		}
	}
	return false
}

// restNamesRepo reports whether the REST merge route among words spells its
// repository rather than gh's `{owner}/{repo}` placeholders, which gh fills
// with this checkout's.
func restNamesRepo(words []string) bool {
	for _, w := range words {
		if m := mergePathRe.FindStringSubmatch(w); m != nil {
			return m[2] != "{owner}" || m[3] != "{repo}"
		}
	}
	return false
}

// approveAct reports whether segment s records the technical facilitator's
// approval under a name the registry entry abcd-land-approve does not read:
// any program, abcd's command built under another name, `go run` of the
// module or of `.`, given the land verb and then --approve. What the
// program is called is not read, since a build is runnable under any name;
// the words it is handed are. A word the line does not fix reads as empty,
// so a flag a substitution prints is not taken as --approve.
func approveAct(s segment) bool {
	for _, a := range commandSites(s) {
		if commandNamed(s, a, "abcd") {
			continue
		}
		land, approve := false, false
		for _, w := range literalWords(s, a.idx+1) {
			if !land {
				land = w == "land"
				continue
			}
			approve = approve || boolFlagOn(w, "--approve")
		}
		if land && approve {
			return true
		}
	}
	return false
}
