package guard

import (
	"regexp"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The merge rule (itd-2610150602547271, spc-2610161609350616 step 5). The
// merge gate in internal/core/landing decides whether abcd itself arms a
// merge; this is the hard rule beside it, on the commands themselves, so a
// path that forgets to ask the gate is still stopped:
//
//   - while facilitator.mode is human, `gh pr merge --auto`, any other
//     `gh pr merge` that merges (into a queue where one gates the default
//     branch, at once where none does; only `--disable-auto`, which withdraws
//     an arming, is not one), a `gh api` call carrying the enqueuePullRequest,
//     enablePullRequestAutoMerge or mergePullRequest mutation, a GraphQL call
//     whose query is not on the line, and a PUT to the REST merge route are
//     refused, naming the setting that would allow each; in auto, such a
//     command that names another repository (or runs after the line changed
//     directory or set GH_REPO) is refused too, since the mode read is this
//     checkout's;
//   - in any mode, the technical facilitator's approval acts are refused:
//     `abcd land --approve` under any program name, and an approving review
//     posted on the forge (`gh pr review --approve`, the REST and GraphQL
//     review routes with APPROVE). An agent that makes either writes a false
//     record of who approved.
//
// The mode is a fact about the repository, not about the command's words, so
// no Pattern can say it, and the guard cannot import the resolver
// (internal/core/landing reads it through the layered reader, and both sit
// above this package). A front door hands it in (WithMergeMode), reading it
// through landing.ReadFetchedFacilitator: the default branch's committed
// .abcd/config.json, as origin has it just fetched, and the machine layer. A
// registry with no resolver handed in (the bundled defaults on their own)
// cannot tell the mode and judges no merge command; the approval acts are
// refused whatever the registry knows. The `abcd` binary's own spelling of the
// land approval is the registry entry abcd-land-approve, which also teaches
// it; every other spelling is raised here.
//
// It guards against an agent merging or approving by mistake, not against a
// determined one: the agent holds the person's own credentials, and can point
// origin elsewhere or reach the forge by routes no command line shows.

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

	// approveForgeEntryID is the reserved id an approving review posted on
	// the forge is refused under, in any mode.
	approveForgeEntryID = "forge-review-approve"

	familyMerge = "merge"

	// mergeSuccessor is what to do instead of a merge command while the mode
	// is human.
	mergeSuccessor = "Leave the merge to the technical facilitator: say the pull request is ready for their approval. Once they approve its head (a forge review, or `abcd land --approve` run by them), `abcd land` arms it."
)

// modeSetting names the setting that allows the merge commands.
var modeSetting = "facilitator.mode: auto in the default branch's committed .abcd/config.json, with no facilitator.mode: human in " + abcdhome.Display("config.json")

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
		if forgeApproval(s) {
			add(payloadSignal{
				id: approveForgeEntryID, verdict: VerdictBlock, family: familyMerge,
				reason:    "An approving review on the forge is the technical facilitator's own act: the merge gate counts an approval by someone with write access, so one an agent posts under the person's account is a false record of who approved, whatever facilitator.mode says.",
				successor: "Leave this one to the person: tell them the pull request and head you believe are ready and why, and let them approve it themselves. A review that comments or requests changes is not refused.",
			})
		}
	}
	if r.merge == nil {
		return out
	}
	// elsewhere is a line that, before this segment, changed directory or
	// set GH_REPO, so gh may resolve a repository other than this checkout's.
	elsewhere := false
	for _, s := range segs {
		hits := mergeCommands(s)
		for _, hit := range hits {
			hit.moved = !hit.named && elsewhere
			hit.named = hit.named || elsewhere
			human, reading := r.merge.mode()
			if !human && !hit.named {
				continue
			}
			sig := payloadSignal{verdict: VerdictBlock, family: familyMerge, successor: mergeSuccessor}
			if !human {
				// Auto here, but the command merges in a repository it
				// names, whose mode this checkout's reading does not
				// answer for, and no setting here changes that.
				sig.id = hit.id
				sig.reason = "The command names the repository it merges in (-R, --repo, GH_REPO, or a REST route spelling owner and name), and the facilitator.mode read here is this checkout's, which does not answer for another repository, so it is refused whatever this checkout's mode says."
				if hit.moved {
					sig.reason = "The line changes directory or sets GH_REPO before this merge command, so gh may resolve a repository other than this checkout's, and the facilitator.mode read here, this checkout's, does not answer for another repository, so it is refused whatever this checkout's mode says."
				}
				sig.successor = "Run the merge from that repository's own checkout without naming the repository, where its mode is read, or leave it to the person."
				add(sig)
				continue
			}
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
		elsewhere = elsewhere || segChangesDir(s) || setsGHRepo(s)
	}
	return out
}

// setsGHRepo reports whether segment s sets GH_REPO, as a command's prefix
// or through export, declare or env, as far as the line fixes the word's
// start.
func setsGHRepo(s segment) bool {
	for _, tok := range s.tokens {
		if strings.HasPrefix(fixedLead(tok), "GH_REPO=") {
			return true
		}
	}
	return false
}

// fixedLead is tok's text before the first part the line does not fix.
func fixedLead(tok string) string {
	if i := strings.IndexAny(tok, unknownText+varText); i >= 0 {
		return tok[:i]
	}
	return tok
}

// mergeHit is one merge command a segment runs: the reserved id it is refused
// under, and whether it names its own repository (-R, --repo, a GH_REPO
// prefix, or a REST path that spells owner and name), whose mode the one read
// here, this checkout's, does not answer for.
type mergeHit struct {
	id    string
	named bool
	// moved is a hit named only because the line, before it, changed
	// directory or set GH_REPO.
	moved bool
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
		for _, tok := range s.tokens[:a.idx] {
			envRepo = envRepo || strings.HasPrefix(fixedLead(tok), "GH_REPO=")
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
			rest := literalWords(s, a.idx+1+idx[1]+1)
			named := envRepo
			for _, tok := range s.tokens[a.idx+1:] {
				w := fixedLead(tok)
				named = named || w == "-R" || w == "--repo" || strings.HasPrefix(w, "--repo=") || (strings.HasPrefix(w, "-R") && (len(w) > 2 || len(tok) > 2))
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
					out = append(out, mergeHit{id: mergeAutoEntryID, named: named})
				case !disable:
					out = append(out, mergeHit{id: mergeNowEntryID, named: named})
				}
			case group == "api":
				from := a.idx + 1 + idx[0] + 1
				eps := apiEndpoints(s, from)
				// An endpoint the line does not fix at all may be graphql,
				// so its call is read as GraphQL as well as REST.
				if graphqlCall(s, eps) || unknownEndpoint(s, eps) {
					if hit, ok := graphqlMerge(s, from); ok {
						out = append(out, mergeHit{id: hit, named: named})
						break
					}
				}
				if put, route, spelled := restMerge(s, from, eps); put && route {
					out = append(out, mergeHit{id: mergeRESTEntryID, named: named || spelled})
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
	if mentionsMutation(s, from) {
		return mergeEnqueueEntryID, true
	}
	// unread is a query the line points away from; unknown is a field the
	// line does not fix, which may be the query unless a fixed query= stood.
	unread, unknown, query := false, false, false
	for i := from; i < len(s.tokens); i++ {
		w, fixed, ok := forgeWord(s, i)
		if flagCouldBe(s.tokens[i], "--input") || (ok && fixed && strings.HasPrefix(w, "--input=")) {
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
			if !strings.HasPrefix(f, "--") {
				// pflag reads `-F=query=…` as `-F query=…`.
				v = strings.TrimPrefix(v, "=")
			}
			return v, true, true, true
		}
	}
	return "", false, false, false
}

// mentionsMutation reports whether a word from `from` on, as far as the line
// fixes it, names one of mergeMutations.
func mentionsMutation(s segment, from int) bool {
	for i := from; i < len(s.tokens); i++ {
		w, _, ok := forgeWord(s, i)
		for _, m := range mergeMutations {
			if ok && strings.Contains(w, m) {
				return true
			}
		}
	}
	return false
}

// ghAPIValueFlags are gh api's options that take the next word as their
// value, so the endpoint reading never takes a method, a header or a field
// for the endpoint.
var ghAPIValueFlags = []string{
	"-R", "--repo", "-X", "--method", "-H", "--header", "-f", "-F", "--field", "--raw-field",
	"--input", "--hostname", "-q", "--jq", "-t", "--template", "--cache", "-p", "--preview",
}

// apiEndpoints are the indexes of the words a `gh api` call's endpoint can
// be, its first operand under each reading of the words from `from` on.
func apiEndpoints(s segment, from int) []int {
	readings, _ := operandReadings(s.tokens[from:], ghAPIValueFlags, 1)
	var out []int
	for _, idx := range readings {
		if len(idx) == 1 {
			out = append(out, from+idx[0])
		}
	}
	return out
}

// graphqlCall reports whether one of a `gh api` call's endpoint words, as the
// line fixes it, is the GraphQL endpoint: `graphql`, `/graphql`, or an API
// URL ending in `/graphql`.
func graphqlCall(s segment, eps []int) bool {
	for _, i := range eps {
		w, fixed, ok := forgeWord(s, i)
		if !ok || !fixed {
			continue
		}
		lw := strings.ToLower(w)
		if lw == "graphql" || lw == "/graphql" || ((strings.HasPrefix(lw, "https://") || strings.HasPrefix(lw, "http://")) && strings.HasSuffix(lw, "/graphql")) {
			return true
		}
	}
	return false
}

// unknownEndpoint reports whether one of a `gh api` call's endpoint words is
// wholly text the line does not fix (`"$EP"`), which may be any endpoint.
func unknownEndpoint(s segment, eps []int) bool {
	for _, i := range eps {
		if tok := s.tokens[i]; unfixed(tok) && strings.Trim(tok, unknownText+varText) == "" {
			return true
		}
	}
	return false
}

// looseText is tok with each part the line does not fix read as one
// segment, for the loose route patterns.
func looseText(tok string) string {
	return strings.NewReplacer(unknownText, "X", varText, "X").Replace(tok)
}

// mergeRouteLoose is the REST merge route as a word the line fixes only in
// part may spell it, its unknown parts read as one segment each:
// `"repos/$REPO/pulls/$N/merge"`.
var mergeRouteLoose = regexp.MustCompile(`(?i)(^|/)pulls/[^/]+/merge/?([?#].*)?$`)

// restMerge reads a `gh api` call's words from `from` on: put is a PUT
// method (`-X PUT`, `-XPUT`, `-X=PUT`, `--method PUT`, `--method=PUT`, any
// case) or a method word the line does not fix; route is an endpoint word
// (eps) that is the REST merge route, or one the line does not fix that could
// be it (wholly unknown, or ending in the route's tail); spelled is a route whose repository is not gh's
// `{owner}/{repo}`, which an unknown route counts as.
func restMerge(s segment, from int, eps []int) (put, route, spelled bool) {
	for _, i := range eps {
		tok := s.tokens[i]
		if !isUnknown(tok) && !strings.ContainsRune(tok, varMark) {
			if m := mergePathRe.FindStringSubmatch(tok); m != nil {
				route = true
				spelled = spelled || !placeholder(m[2], "owner") || !placeholder(m[3], "repo")
			}
		} else if strings.Trim(tok, unknownText+varText) == "" || mergeRouteLoose.MatchString(looseText(tok)) {
			route, spelled = true, true
		}
	}
	for i := from; i < len(s.tokens); i++ {
		tok := s.tokens[i]
		var m string
		switch {
		case tok == "-X" || tok == "--method":
			if i+1 >= len(s.tokens) {
				continue
			}
			m = s.tokens[i+1]
		case strings.HasPrefix(tok, "--method="):
			m = strings.TrimPrefix(tok, "--method=")
		case strings.HasPrefix(tok, "-X") && len(tok) > 2:
			m = strings.TrimPrefix(strings.TrimPrefix(tok, "-X"), "=")
		default:
			continue
		}
		put = put || unfixed(m) || strings.EqualFold(m, "PUT")
	}
	return put, route, spelled
}

// placeholder reports whether w is gh api's placeholder for name, in either
// spelling gh fills (`{owner}`, `:owner`).
func placeholder(w, name string) bool { return w == "{"+name+"}" || w == ":"+name }

// unfixed reports whether tok holds text the line does not fix: a
// substitution's output, or a variable's value in a re-read payload.
func unfixed(tok string) bool { return isUnknown(tok) || strings.ContainsRune(tok, varMark) }

// approveAct reports whether segment s records the technical facilitator's
// approval under a name the registry entry abcd-land-approve does not read:
// any program, abcd's command built under another name, `go run` of the
// module or of `.`, given the land verb and --approve in either order. What the
// program is called is not read, since a build is runnable under any name;
// the words it is handed are. A word the line does not fix reads as empty,
// so a flag a substitution prints is not taken as --approve.
func approveAct(s segment) bool {
	for _, a := range commandSites(s) {
		if commandNamed(s, a, "abcd") {
			continue
		}
		// cobra parses the found command's flags wherever they stand, so
		// `--approve land` sets land's --approve as `land --approve` does.
		land, approve := false, false
		for _, w := range literalWords(s, a.idx+1) {
			land = land || w == "land"
			approve = approve || boolFlagOn(w, "--approve")
		}
		if land && approve {
			return true
		}
	}
	return false
}

// reviewRouteLoose is the REST route that posts a pull request review or
// submits a pending one, unknown parts read as one segment each.
var reviewRouteLoose = regexp.MustCompile(`(?i)(^|/)pulls/[^/]+/reviews(/[^/]+/events)?/?([?#].*)?$`)

// reviewMutations are the GraphQL mutations that post or submit a review.
var reviewMutations = []string{"addPullRequestReview", "submitPullRequestReview"}

// approveEventRe is an APPROVE event spelled inside a review mutation.
var approveEventRe = regexp.MustCompile(`event\s*:\s*APPROVE\b`)

// forgeApproval reports whether segment s posts an approving review on the
// forge: `gh pr review` with --approve (-a); a `gh api` call to the REST
// review route whose event field is APPROVE, or whose event or body the line
// does not show (an unfixed field, --input); or one carrying a review
// mutation whose event is APPROVE, in the query or in a field (a GraphQL
// variable) set to APPROVE; a GraphQL call whose query is not on the line is
// read as one that may carry such a mutation. A
// review that comments or requests changes is not one, and neither is a
// read of the reviews, whatever its --jq filter says.
func forgeApproval(s segment) bool {
	for _, a := range sitesNamed(s, "gh") {
		readings, _ := operandReadings(s.tokens[a.idx+1:], ghRepoFlags, 2)
		for _, idx := range readings {
			if len(idx) < 1 {
				continue
			}
			group, _, ok := forgeWord(s, a.idx+1+idx[0])
			if !ok {
				continue
			}
			from := a.idx + 1 + idx[0] + 1
			if group == "pr" && len(idx) >= 2 {
				if verb, _, ok := forgeWord(s, a.idx+1+idx[1]); ok && verb == "review" {
					for _, w := range literalWords(s, from) {
						if w == "-a" || boolFlagOn(w, "--approve") {
							return true
						}
					}
				}
			}
			if group != "api" {
				continue
			}
			rest := false
			for _, i := range apiEndpoints(s, from) {
				rest = rest || reviewRouteLoose.MatchString(looseText(s.tokens[i]))
			}
			mutation := false
			for i := from; i < len(s.tokens); i++ {
				w, _, ok := forgeWord(s, i)
				for _, m := range reviewMutations {
					mutation = mutation || (ok && strings.Contains(w, m))
				}
			}
			// A GraphQL call whose query is not on the line may carry a
			// review mutation; its event is then read from its fields.
			eps := apiEndpoints(s, from)
			gql := graphqlCall(s, eps) || unknownEndpoint(s, eps)
			queryShown := false
			for i := from; i < len(s.tokens); i++ {
				if v, vfixed, vok, isField := fieldValue(s, i); isField && vok && vfixed && strings.HasPrefix(v, "query=") && !strings.HasPrefix(v, "query=@") {
					queryShown = true
				}
			}
			mutation = mutation || (gql && !queryShown)
			if !rest && !mutation {
				continue
			}
			for i := from; i < len(s.tokens); i++ {
				w, fixed, ok := forgeWord(s, i)
				if mutation && ok && approveEventRe.MatchString(w) {
					return true
				}
				if rest && ok && fixed && (w == "--input" || strings.HasPrefix(w, "--input=")) {
					return true
				}
				v, vfixed, vok, isField := fieldValue(s, i)
				switch {
				case !isField:
				case vok && vfixed && (strings.EqualFold(v, "event=APPROVE") || (mutation && strings.HasSuffix(v, "=APPROVE"))):
					return true
				case rest && !vok:
					// The value word is not fixed at all: it is the event
					// unless the line fixes another key (per_page="$N").
					if key := fixedLead(s.tokens[i+1]); key == "" || strings.HasPrefix("event=", key) || strings.HasPrefix(key, "event=") {
						return true
					}
				case rest && !vfixed && (v == "" || strings.HasPrefix(v, "event=")):
					return true
				}
			}
		}
	}
	return false
}
