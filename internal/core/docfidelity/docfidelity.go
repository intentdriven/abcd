// Package docfidelity is the doc-fidelity gate over the brief (itd-60,
// spc-2609020903498198): the brief describes every surface that ships, or the
// intent does not reach shipped.
package docfidelity

import (
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// RunReviewFirst is the words every missing, stale or unusable review refusal
// carries (ruling DR3).
const RunReviewFirst = "run the docs review first"

// Doc kinds a reviewed sentence stands in.
const (
	DocBrief  = "brief"
	DocPublic = "public"
)

// Kind is the kind of shipped surface.
type Kind string

const (
	KindVerb    Kind = "verb"
	KindSubVerb Kind = "sub-verb"
	KindAgent   Kind = "agent"
)

// Surface is one shipped surface.
type Surface struct {
	Kind Kind   `json:"kind"`
	Name string `json:"name"`
}

// Row is one coverage row.
type Row struct {
	Surface
	Chapter string `json:"chapter,omitempty"`
}

// Sentence is one sentence a reviewer judged false.
type Sentence struct {
	Doc      string `json:"doc"`
	Chapter  string `json:"chapter"`
	Sentence string `json:"sentence"`
	Evidence string `json:"evidence"`
	// Replacement is the reviewer's drafted sentence, "" when none was drafted.
	Replacement string `json:"replacement,omitempty"`
}

// Edit is a proposed brief edit: replace Sentence in Chapter by Replacement.
type Edit struct {
	Chapter     string `json:"chapter"`
	Sentence    string `json:"sentence"`
	Replacement string `json:"replacement"`
	Evidence    string `json:"evidence"`
}

// Flag is the review flag an applied edit records: the brief carries a
// sentence nobody but the reviewer wrote until the product thinker reads it.
type Flag struct {
	Chapter     string `json:"chapter"`
	Sentence    string `json:"sentence"`
	Replacement string `json:"replacement"`
	Evidence    string `json:"evidence,omitempty"`
	Commit      string `json:"commit"`
	Applied     string `json:"applied"`
}

// ReviewStatus is what the saved review says about the code under judgement.
type ReviewStatus string

const (
	ReviewMatch        ReviewStatus = "match"
	ReviewNone         ReviewStatus = "none"
	ReviewStale        ReviewStatus = "stale"
	ReviewInvalid      ReviewStatus = "invalid"
	ReviewHold         ReviewStatus = "hold"
	ReviewInconclusive ReviewStatus = "inconclusive"
)

// Review is layer 2's answer.
type Review struct {
	Status   ReviewStatus `json:"status"`
	Commit   string       `json:"commit"`
	Named    string       `json:"named,omitempty"`
	Verdict  string       `json:"verdict,omitempty"`
	Problems []string     `json:"problems,omitempty"`
	Findings []Sentence   `json:"findings,omitempty"`
}

// Reviewer is the delegated layer.
type Reviewer interface{ Review() Review }

// Inputs is everything the judgement reads.
type Inputs struct {
	Commands   []surface.Command
	Agents     []string
	Chapters   map[string]string
	Population []string
	Flags      []Flag
}

// Verdict is the judgement.
type Verdict struct {
	Report     bool       `json:"report"`
	Population []string   `json:"population"`
	Coverage   []Row      `json:"coverage"`
	Uncovered  []Surface  `json:"uncovered"`
	Review     *Review    `json:"review,omitempty"`
	False      []Sentence `json:"false_sentences"`
	Public     []Sentence `json:"public_findings"`
	Proposed   []Edit     `json:"proposed_edits"`
	Applied    []Sentence `json:"applied_edits"`
	Refuse     bool       `json:"refuse"`
	Reasons    []string   `json:"reasons"`
}

// Key is the surface's sort key: the command path for a verb or sub-verb,
// "agent:<name>" for an agent.
func (s Surface) Key() string {
	if s.Kind == KindAgent {
		return "agent:" + s.Name
	}
	return s.Name
}

// Shipped derives the shipped surfaces from the command tree and the agent
// set: every command that is neither hidden (itself or through an ancestor)
// nor a moved spelling, below the bare root, and every agent. The tree is the
// one commands.md and surface.json are generated from, so the three agree.
func Shipped(commands []surface.Command, agents []string) []Surface {
	hidden := map[string]bool{}
	for _, c := range commands {
		if c.Hidden {
			hidden[c.Path] = true
		}
	}
	under := func(path string) bool {
		for p := path; ; {
			if hidden[p] {
				return true
			}
			i := strings.LastIndexByte(p, ' ')
			if i < 0 {
				return false
			}
			p = p[:i]
		}
	}
	var out []Surface
	for _, c := range commands {
		words := strings.Fields(c.Path)
		if len(words) < 2 || c.MovedTo != "" || under(c.Path) {
			continue
		}
		k := KindVerb
		if len(words) > 2 {
			k = KindSubVerb
		}
		out = append(out, Surface{Kind: k, Name: c.Path})
	}
	for _, a := range agents {
		out = append(out, Surface{Kind: KindAgent, Name: a})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

// Names reports whether a chapter's text names the surface. A command is named
// by a code span that is its path, or that begins with its path and a space (an
// invocation with operands); an agent by a code span that is its name, or by
// its prompt's path agents/<name>.md. A bare word is never a naming: a surface
// named only in passing prose has no chapter describing it.
func Names(text string, s Surface) bool {
	if s.Kind == KindAgent {
		return strings.Contains(text, "`"+s.Name+"`") || strings.Contains(text, "agents/"+s.Name+".md")
	}
	return strings.Contains(text, "`"+s.Name+"`") || strings.Contains(text, "`"+s.Name+" ")
}

// Judge composes the verdict: layer 1 first, and only when it holds, layer 2.
// It reads its inputs and writes nothing. In report mode (the per-task pass)
// both layers run whatever layer 1 found, every finding is stated, and nothing
// refuses.
func Judge(in Inputs, r Reviewer, report bool) Verdict {
	v := Verdict{Report: report, Population: append([]string(nil), in.Population...),
		Uncovered: []Surface{}, False: []Sentence{}, Public: []Sentence{},
		Proposed: []Edit{}, Applied: []Sentence{}, Reasons: []string{}}
	who := ""
	if len(in.Population) > 0 {
		who = strings.Join(in.Population, ", ") + ": "
	}
	chapters := make([]string, 0, len(in.Chapters))
	for name := range in.Chapters {
		chapters = append(chapters, name)
	}
	sort.Strings(chapters)
	for _, s := range Shipped(in.Commands, in.Agents) {
		row := Row{Surface: s}
		for _, name := range chapters {
			if Names(in.Chapters[name], s) {
				row.Chapter = name
				break
			}
		}
		v.Coverage = append(v.Coverage, row)
		if row.Chapter == "" {
			v.Uncovered = append(v.Uncovered, s)
			v.Reasons = append(v.Reasons, who+"no brief chapter under 04-surfaces/ names the "+string(s.Kind)+" `"+s.Name+"`")
		}
	}
	layerOne := len(v.Reasons) > 0
	if layerOne && !report {
		// An undocumented surface needs no reviewer to be judged, and paying
		// for one would make the cheap half hostage to the expensive half.
		v.Refuse = true
		return v
	}
	rev := r.Review()
	v.Review = &rev
	for _, f := range rev.Findings {
		if f.Doc == DocPublic {
			v.Public = append(v.Public, f)
		}
	}
	at := short(rev.Commit)
	switch rev.Status {
	case ReviewMatch:
		// Record refuses a PROMOTE naming a false brief sentence; a receipt
		// edited to carry one after it was saved is still not a pass.
		for _, f := range rev.Findings {
			if f.Doc != DocPublic {
				v.Reasons = append(v.Reasons, who+"the doc-fidelity review for "+at+" is PROMOTE yet names a false sentence in "+
					f.Chapter+": \""+f.Sentence+"\", so it is not a usable verdict: "+RunReviewFirst)
			}
		}
	case ReviewNone:
		v.Reasons = append(v.Reasons, who+"no saved doc-fidelity review names "+at+": "+RunReviewFirst)
	case ReviewStale:
		v.Reasons = append(v.Reasons, who+"the saved doc-fidelity review names "+short(rev.Named)+", not "+at+
			", so the code changed since it was reviewed: "+RunReviewFirst)
	case ReviewHold:
		n := 0
		for _, f := range rev.Findings {
			if f.Doc == DocPublic {
				continue
			}
			n++
			if applied(in, f) {
				v.Applied = append(v.Applied, f)
				continue
			}
			v.False = append(v.False, f)
			reason := who + "the docs review confirmed a false sentence in " + f.Chapter + ": \"" + f.Sentence + "\" (" + f.Evidence + ")"
			if f.Replacement != "" {
				v.Proposed = append(v.Proposed, Edit{Chapter: f.Chapter, Sentence: f.Sentence, Replacement: f.Replacement, Evidence: f.Evidence})
				reason += "; the reviewer drafted its correction — `abcd docs fidelity --apply` applies it and flags it for review"
			}
			v.Reasons = append(v.Reasons, reason)
		}
		if n == 0 && len(v.Public) == 0 {
			v.Reasons = append(v.Reasons, who+"the doc-fidelity review for "+at+" is HOLD but names no sentence, so there is nothing to correct: "+RunReviewFirst)
		}
	case ReviewInconclusive:
		v.Reasons = append(v.Reasons, who+"the doc-fidelity review for "+at+" returned no usable verdict ("+rev.Verdict+"): "+RunReviewFirst)
	default:
		msg := strings.Join(rev.Problems, "; ")
		if msg == "" {
			msg = "status " + string(rev.Status)
		}
		v.Reasons = append(v.Reasons, who+"the doc-fidelity review for "+at+" is refused ("+msg+"): "+RunReviewFirst)
	}
	v.Refuse = !report && len(v.Reasons) > 0
	return v
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	if sha == "" {
		return "HEAD"
	}
	return sha
}

// applied reports that a confirmed false sentence has been corrected by a
// drafted edit and flagged for review: its chapter no longer carries the
// sentence, carries the drafted replacement, and a flag names the edit. All
// three, so neither a silent hand edit nor a flag over an unedited chapter
// completes the change with the brief lagging and no flag recorded.
func applied(in Inputs, f Sentence) bool {
	if f.Replacement == "" {
		return false
	}
	text, ok := in.Chapters[f.Chapter]
	if !ok || strings.Contains(text, f.Sentence) || !strings.Contains(text, f.Replacement) {
		return false
	}
	for _, fl := range in.Flags {
		if fl.Chapter == f.Chapter && fl.Sentence == f.Sentence && fl.Replacement == f.Replacement {
			return true
		}
	}
	return false
}
