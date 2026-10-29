package site

// Authorship, as the repository itself records it.
//
// Two facts are published and they are different facts. WHO wrote the code is
// `git shortlog` folded through `.mailmap` — the authors of record, humans
// responsible for the work — published as a COUNT of people and of their
// commits, never as a list: the site names no person (adr-2610291037481588,
// superseding the naming part of adr-47). WHAT assisted is the `Assisted-by:`
// trailer, which
// this repository requires on every AI-assisted commit and requires as an
// explicit `None` on every human-only one, so that silence is never mistaken for
// a declaration; a commit abcd composes from record facts declares
// `abcd:<version>`, which names no model and is counted apart from both.
// Presenting the second as authorship would be exactly the claim the trailer
// convention exists to refuse.
//
// The bots-and-tools row is derived, not listed. An author is a tool when its
// name carries the forge's own `[bot]` suffix, or when the repository's own
// trailers name it as an assisting vendor AND it commits from an address that is
// structurally a machine's — which together are what a pre-policy commit authored
// by the tool looks like from here. Deriving it means a second tool that ever
// lands a commit appears in the right row without an edit, and means no vendor
// name is written into this file.
//
// The second signal is a CONJUNCTION and that is the whole of it. A vendor token
// is an ordinary word, so a name match alone says nothing: on its own it moved a
// human author's entire shortlog count into the bots row the moment somebody
// declared a trailer whose vendor happened to be that person's git name
// (iss-2609081940550352). Neither half is sufficient either way round — a person
// may well commit from a forge privacy address, and an unrelated tool name is not
// evidence of anything.

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/assistedby"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxShortlogBytes bounds the authorship reads.
const maxShortlogBytes = 4 << 20

var (
	shortlogRe = regexp.MustCompile(`^\s*(\d+)\s+(.*?)\s*<([^>]*)>\s*$`)
)

// The structural machine signals — the forge's `[bot]` name suffix and the
// machine mailboxes — are identity.IsMachineName and identity.IsMachineAddress,
// read from the one list the CI attribution gate reads too
// (internal/core/identity/tool-identities.txt), so this page and the gate cannot
// disagree about what a machine's address looks like. They read the LOCAL PART,
// never the host: a person routinely commits from
// `1234+name@users.noreply.github.com`, the forge's privacy address, and treating
// that host as a machine signal would demote exactly the contributors the page
// exists to credit.

// chartableModel is the model a trailer value names, or "" when it names none.
//
// The chart is an inventory of MODELS, so a value has to name one to earn a bar.
// A value that does not — a bare vendor with no version, which the convention
// deliberately refuses, or a sentence somebody typed into the trailer block — is
// still a disclosure and is still counted as one at the commit level; it simply
// has no model to draw. Publishing the free text instead puts whatever was typed
// onto the site as a label under a heading that reads as a list of models.
//
// A value whose HEAD conforms is CLIPPED to that head rather than dropped: the
// model was named, and the trailing prose is the part with no place on a chart.
//
// The grammar is the gate's TRAILER_RE, read through internal/core/assistedby,
// its one Go home and the place it is tied to the gate.
func chartableModel(v string) string { return assistedby.ModelHead(v) }

// Author is one row of the bots-and-tools column: a forge bot or a tool that
// authored commits, named because a tool's name is credit, not a person's.
//
// The address is deliberately absent. `git shortlog -sne` yields it and the
// mailmap folds it, so it is read — but the site is published, and republishing
// an address on a web page is a harvesting surface the record never asked for.
// No profile link is derived either: the row names the tool and stops there.
//
// Human authors never become an Author. They are counted as they are read
// (Authorship.HumanAuthors, HumanCommits) and nothing about any one of them is
// kept, so no later change to a page or to the JSON can publish a name that is
// no longer there to publish.
type Author struct {
	Name    string `json:"name"`
	Commits int    `json:"commits"`
	// email is read so the column sorts stably, and is never exported.
	email string
}

// humanKey is the identity a human author is counted under, so one person who
// commits under two names or two spellings of one address counts once.
//
// It is the address, case-folded, because an address is what an identity IS in
// git and the case of a mailbox is not something a person chooses twice. A
// forge privacy address is keyed on the login it carries instead, so its two
// forms (`1234+name@…` and `name@…`) fold to one person, compared without
// regard to case as the forge compares logins. A name is not a key while
// there is an address: two people may share one, and a count that merged them
// would be wrong in the other direction. Anything finer is the repository's `.mailmap`, which
// shortlog has already applied.
//
// The one exception is an EMPTY address (`Name <>`), which identifies nobody:
// keying it as the empty address would fold every such author into one
// person. Such an author is keyed by the name, case-folded and trimmed, since
// it is the only thing the commit says about who wrote it.
func humanKey(name, email string) string {
	if u := claim.NoReplyUsername(email); u != "" {
		return "login:" + strings.ToLower(u)
	}
	if e := strings.ToLower(strings.TrimSpace(email)); e != "" {
		return "email:" + e
	}
	return "name:" + strings.ToLower(strings.TrimSpace(name))
}

// ModelTally is one distinct `Assisted-by:` value and how often it appears.
type ModelTally struct {
	Model   string `json:"model"`
	Commits int    `json:"commits"`
}

// Authorship is the whole picture: who authored, what assisted, the commits abcd
// composed, and the commits that declared no assistance at all.
type Authorship struct {
	// Commits is the total number of commits in the history walked.
	Commits int `json:"commits"`
	// Authored is Commits less the merges: the commits a person actually wrote,
	// and the only honest denominator for a disclosure rate.
	Authored int `json:"authored"`
	// Merges is what was set aside to get there. It is published rather than
	// quietly subtracted, because a denominator that changed without saying so
	// is how the rate went wrong in the first place.
	Merges int `json:"merges"`
	// HumanAuthors is how many people authored the history: the authors of
	// record, mailmap-folded and then folded once more by humanKey. It is a
	// count and never a list, so the export names no person.
	HumanAuthors int `json:"human_authors"`
	// HumanCommits is how many commits on HEAD those people authored, as
	// shortlog counts them (merges they made included).
	HumanCommits int `json:"human_commits"`
	// Bots are the forge bots and tool-authored commits, kept in a separate row
	// so a reader never has to guess which lines are people.
	Bots []Author `json:"bots"`
	// AssistedCommits is how many AUTHORED commits declare assistance — the
	// commit-level count, and the numerator of the disclosure rate.
	AssistedCommits int `json:"assisted_commits"`
	// Assisted is the number of `Assisted-by:` trailer occurrences that NAME A
	// MODEL, across authored commits. It is what the per-model tally sums to —
	// it is rendered as the chart's own total, so it can be nothing else; it is
	// not a number of commits, and a commit naming two models counts twice here
	// and once in AssistedCommits. Rendering this one as a count of commits is
	// the defect that published a disclosure rate well below the truth.
	//
	// A declaration that names no model (see chartableModel) is counted in
	// AssistedCommits and not here, so this can sit below that figure. The
	// disclosure rate is computed from the commit-level counts and is unaffected.
	Assisted int `json:"assisted"`
	// MultiTrailerCommits is how many AUTHORED commits declare more than one
	// model. It is counted per COMMIT rather than derived as Assisted minus
	// AssistedCommits: that subtraction is a surplus of occurrences, and a
	// single commit naming three models would report two — publishing an
	// occurrence figure under a label that says commits, which is the exact
	// conflation this type exists to keep apart.
	MultiTrailerCommits int `json:"multi_trailer_commits"`
	// DeclaredNone is the number of AUTHORED commits declaring
	// `Assisted-by: None` — work no tool touched, saying so.
	DeclaredNone int `json:"declared_none"`
	// ComposedByAbcd is the number of AUTHORED commits declaring
	// `Assisted-by: abcd:<version>` and no model: text abcd composed from record
	// facts (ruling PC1), which no model wrote and which a tool did touch. It is
	// neither assistance nor a declaration of none, so it is counted in neither
	// and never charted, and it is stated beside DeclaredNone. A commit naming a
	// model as well is assisted: a model's text is in it.
	ComposedByAbcd int `json:"composed_by_abcd"`
	// Undeclared is the number of AUTHORED commits carrying no trailer at all.
	// It is published because an absent trailer and a forgotten one are the
	// same bytes, and the honest number is the one that says how much of the
	// history predates the convention. Merges are excluded: nobody wrote them,
	// so nothing was forgotten.
	Undeclared int `json:"undeclared"`
	// ByModel tallies each distinct declared MODEL by OCCURRENCE, most first —
	// the trailer-shaped values alone, clipped to the grammar.
	ByModel []ModelTally `json:"by_model"`
}

// noneDeclaration is the trailer value a human-only commit carries. It is one of
// the two accepted non-vendor values, both closed shapes, so it can be compared
// for exactly; the other is the abcd label, read through internal/core/assistedby
// (the one Go home of its grammar, tied to the gate's ABCD_RE by test).
const noneDeclaration = "None"

// LoadAuthorship reads the authorship facts out of git. A directory that is not
// a repository yields a zero Authorship and no error.
func LoadAuthorship(repoRoot string) (Authorship, error) {
	// Empty rather than absent: a page that renders "no bots" from an empty list
	// and nothing at all from a null is a page with two ways to say one thing.
	a := Authorship{Bots: []Author{}, ByModel: []ModelTally{}}
	if !gitutil.InRepo(repoRoot) {
		return a, nil
	}

	// The parent list comes back with the trailers because a MERGE is not a
	// commit anyone wrote: the forge creates it, no convention asks it to
	// declare anything, and counting merges as undeclared work buries the real
	// number under them.
	trailers, err := gitutil.RunCapped(repoRoot, maxShortlogBytes,
		"log", "--pretty=format:%x00%p%x1e%(trailers:key=Assisted-by,valueonly,separator=%x1f)")
	if err != nil {
		return Authorship{}, err
	}
	vendors := map[string]bool{}
	tally := map[string]int{}
	// One record per commit, each opened by the NUL the format writes; the first
	// field of the split is the text before the first record and is not a commit.
	records := strings.Split(trailers, "\x00")
	if len(records) > 0 {
		records = records[1:]
	}
	for _, rec := range records {
		a.Commits++
		parents, rest, _ := strings.Cut(rec, "\x1e")
		if len(strings.Fields(parents)) > 1 {
			a.Merges++
			continue
		}
		a.Authored++
		declared, assisted, composed, models := false, false, false, 0
		for _, v := range strings.Split(rest, "\x1f") {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			declared = true
			if v == noneDeclaration {
				// Counted, never charted: a declaration of NO assistance in a
				// tally of what assisted would make the bars sum past their own
				// total. It is stated separately, beneath the chart.
				continue
			}
			if assistedby.NamesAbcd(v) {
				// abcd composed the text and no model wrote it: never a model
				// bar and never a vendor, whose name would otherwise take part
				// in the bots-row conjunction below. A label in a shape the gate
				// refuses (`ABCD:latest`) still claims abcd's provenance, not a
				// model's, so it is read the same way.
				composed = true
				continue
			}
			// The commit declares that something assisted, whatever shape the
			// value is in; that fact is the disclosure and is counted first.
			assisted = true
			model := chartableModel(v)
			if model == "" {
				continue
			}
			tally[model]++
			a.Assisted++
			models++
			vendor, _, _ := strings.Cut(model, ":")
			vendors[vendor] = true
		}
		if models > 1 {
			a.MultiTrailerCommits++
		}
		switch {
		case assisted:
			a.AssistedCommits++
		case composed:
			a.ComposedByAbcd++
		case declared:
			a.DeclaredNone++
		default:
			a.Undeclared++
		}
	}

	for model, n := range tally {
		a.ByModel = append(a.ByModel, ModelTally{Model: model, Commits: n})
	}
	sort.Slice(a.ByModel, func(i, j int) bool {
		if a.ByModel[i].Commits != a.ByModel[j].Commits {
			return a.ByModel[i].Commits > a.ByModel[j].Commits
		}
		return a.ByModel[i].Model < a.ByModel[j].Model
	})

	shortlog, err := gitutil.RunCapped(repoRoot, maxShortlogBytes, "shortlog", "-sne", "HEAD")
	if err != nil {
		return Authorship{}, err
	}
	people := map[string]bool{}
	for _, line := range strings.Split(shortlog, "\n") {
		m := shortlogRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		name, email := m[2], m[3]
		if identity.IsMachineName(name) ||
			(vendors[name] && identity.IsMachineAddress(identity.RoleAuthor, email)) {
			a.Bots = append(a.Bots, Author{Name: name, Commits: n, email: email})
			continue
		}
		people[humanKey(name, email)] = true
		a.HumanCommits += n
	}
	a.HumanAuthors = len(people)
	sortAuthors(a.Bots)
	return a, nil
}

// sortAuthors orders an authorship column: most commits first, then by name, so
// the page is a function of the history and not of git's output order.
func sortAuthors(as []Author) {
	sort.Slice(as, func(i, j int) bool {
		if as[i].Commits != as[j].Commits {
			return as[i].Commits > as[j].Commits
		}
		if as[i].Name != as[j].Name {
			return as[i].Name < as[j].Name
		}
		return as[i].email < as[j].email
	})
}
