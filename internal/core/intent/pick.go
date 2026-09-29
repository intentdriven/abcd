package intent

// pick.go is the readiness score and the pick order of `abcd build next`
// (itd-2609211116005482, spc-2609212015048113 scope 2 and 4): which planned
// intent a run takes next, computed from facts the records hold, and the
// grounds entry that writes the reason onto the chosen intent.
//
// The score has three parts at equal weight (decision 7, the bundled default,
// revisable after ten picks): how clear the acceptance criteria are, whether
// the spec names a test path, and how small the spec's expected footprint
// is. Each part is 0 to PartPoints. A part whose section is absent reads zero
// and says so. Age is not scored: it breaks ties only, oldest first (decision
// 2). The ordering is declared a heuristic (decision 8); PickLess is its one
// statement, so every reader that orders intents for a pick (the build's pick,
// the status board's "next up") orders them the same way.
//
// The reason is computed, never composed (decision 5): the candidates with
// their scores, the rule that placed the winner, the runner-up and why it
// lost, and a falsifier the lane's state can meet.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// PartPoints is the most one part of the readiness score carries.
const PartPoints = 100

// ReadinessWeights are the weights of the score's three parts.
type ReadinessWeights struct {
	Criteria  int `json:"criteria"`
	TestPath  int `json:"test_path"`
	Footprint int `json:"footprint"`
}

// BundledReadinessWeights are the equal weights the score is declared with
// (decision 7).
var BundledReadinessWeights = ReadinessWeights{Criteria: 1, TestPath: 1, Footprint: 1}

// ReadinessPart is one part of the score: its points and what earned them.
type ReadinessPart struct {
	Points int    `json:"points"`
	Detail string `json:"detail"`
}

// ReadinessScore is an intent's readiness, computed from its record and its
// spec's.
type ReadinessScore struct {
	Criteria  ReadinessPart `json:"criteria"`
	TestPath  ReadinessPart `json:"test_path"`
	Footprint ReadinessPart `json:"footprint"`
	// Total is the weighted sum of the three parts; it orders the pick.
	Total int `json:"total"`
	// NoFootprint is true when the spec carries no `## Footprint` section, so
	// its test-path and footprint parts read zero.
	NoFootprint bool `json:"no_footprint"`
}

// gwtRe finds the three clauses of a Given-When-Then criterion.
var (
	givenRe = regexp.MustCompile(`(?i)\bgiven\b`)
	whenRe  = regexp.MustCompile(`(?i)\bwhen\b`)
	thenRe  = regexp.MustCompile(`(?i)\bthen\b`)
)

// readiness scores an intent from its record and its spec's content, with the
// bundled weights.
func readiness(intentContent, specContent string) ReadinessScore {
	return readinessWith(BundledReadinessWeights, intentContent, specContent)
}

// readinessWith is readiness under the given weights.
func readinessWith(w ReadinessWeights, intentContent, specContent string) ReadinessScore {
	var s ReadinessScore

	lines := strings.Split(intentContent, "\n")
	mask := mdrecord.Mask(lines)
	total, clear := 0, 0
	if start, end, ok := mdrecord.SectionLineRangeIn(lines, mask, acHeadingRe); ok {
		for _, b := range mdrecord.BulletBlocks(lines, mask, start, end) {
			total++
			text := strings.Join(lines[b.Start:b.End], " ")
			if givenRe.MatchString(text) && whenRe.MatchString(text) && thenRe.MatchString(text) {
				clear++
			}
		}
	}
	if total == 0 {
		s.Criteria = ReadinessPart{Detail: "no acceptance criteria"}
	} else {
		s.Criteria = ReadinessPart{Points: PartPoints * clear / total,
			Detail: fmt.Sprintf("%d of %d criteria in Given-When-Then form", clear, total)}
	}

	fp := spec.ReadFootprint(specContent)
	switch {
	case !fp.Present:
		s.NoFootprint = true
		s.TestPath = ReadinessPart{Detail: "the spec carries no footprint"}
		s.Footprint = ReadinessPart{Detail: "the spec carries no footprint"}
	default:
		if fp.Tests == "" {
			s.TestPath = ReadinessPart{Detail: "the spec's footprint names no tests"}
		} else {
			s.TestPath = ReadinessPart{Points: PartPoints, Detail: "the spec's footprint names its tests"}
		}
		if n := len(fp.Packages); n == 0 {
			s.Footprint = ReadinessPart{Detail: "the spec's footprint names no packages"}
		} else {
			s.Footprint = ReadinessPart{Points: PartPoints / n, Detail: fmt.Sprintf("%d package(s)", n)}
		}
	}
	s.Total = w.Criteria*s.Criteria.Points + w.TestPath*s.TestPath.Points + w.Footprint*s.Footprint.Points
	return s
}

// ReadinessIn scores an intent the caller has already looked up, from its
// record and the record of specID read through a spec store the caller has
// already loaded. It is the one read both orderings score through, the build's
// pick and the status board's "next up", so the two cannot score an intent
// differently.
func ReadinessIn(repoRoot string, store spec.Store, it Intent, specID string) (ReadinessScore, error) {
	ic, err := readRepoFile(filepath.Join(repoRoot, filepath.FromSlash(it.Path)), it.Path)
	if err != nil {
		return ReadinessScore{}, err
	}
	sp, ok := store.Lookup(specID)
	if !ok {
		return ReadinessScore{}, fmt.Errorf("intent: %s's spec %s is not in the spec store", it.ID, specID)
	}
	sc, err := readRepoFile(filepath.Join(repoRoot, filepath.FromSlash(sp.Path)), sp.Path)
	if err != nil {
		return ReadinessScore{}, err
	}
	return readiness(string(ic), string(sc)), nil
}

// PickCandidate is one intent the pick may take, with its score.
type PickCandidate struct {
	ID    string         `json:"id"`
	Score ReadinessScore `json:"score"`
}

// IDOlder reports whether intent id a is older than b: an ordinal id predates
// every timestamp id (adr-45), and ids of one kind order by their number.
func IDOlder(a, b string) bool {
	na := strings.TrimLeft(strings.TrimPrefix(a, "itd-"), "0")
	nb := strings.TrimLeft(strings.TrimPrefix(b, "itd-"), "0")
	if len(na) != len(nb) {
		return len(na) < len(nb)
	}
	return na < nb
}

// PickLess is the pick order, the one statement of it: the readiest first,
// the oldest among equals.
func PickLess(a, b PickCandidate) bool {
	if a.Score.Total != b.Score.Total {
		return a.Score.Total > b.Score.Total
	}
	return IDOlder(a.ID, b.ID)
}

// PickOrder sorts candidates into the pick order, in place.
func PickOrder(c []PickCandidate) {
	sort.SliceStable(c, func(i, j int) bool { return PickLess(c[i], c[j]) })
}

// PickRule is the rule that places the winner, as the reason states it.
const PickRule = "readiest first by the total of three equal-weight parts (criteria clarity, test path, footprint), the oldest first among equals"

// PickFalsifier is the lane outcome that shows a pick wrong, as the reason
// states it: an outcome the lane's state file can meet.
const PickFalsifier = "the pick is shown wrong if the lane's state shows fix rounds past the pace rule's count, or the lane hands back as unachievable"

// Pick is a choice among candidates.
type Pick struct {
	// Chosen is the candidate taken.
	Chosen PickCandidate `json:"chosen"`
	// RunnerUp is the next in the pick order; nil when there was one candidate.
	RunnerUp *PickCandidate `json:"runner_up"`
	// TieBrokenByAge is true when the runner-up scored the same as the chosen
	// one, so age placed the winner.
	TieBrokenByAge bool `json:"tie_broken_by_age"`
	// Candidates are every candidate, in the pick order.
	Candidates []PickCandidate `json:"candidates"`
	Rule       string          `json:"rule"`
	Falsifier  string          `json:"falsifier"`
}

// Choose orders the candidates and takes the first. ok is false when there is
// none.
func Choose(cands []PickCandidate) (Pick, bool) {
	if len(cands) == 0 {
		return Pick{}, false
	}
	ordered := append([]PickCandidate(nil), cands...)
	PickOrder(ordered)
	p := Pick{Chosen: ordered[0], Candidates: ordered, Rule: PickRule, Falsifier: PickFalsifier}
	if len(ordered) > 1 {
		ru := ordered[1]
		p.RunnerUp = &ru
		p.TieBrokenByAge = ru.Score.Total == p.Chosen.Score.Total
	}
	return p, true
}

// RunPickMarker opens the text of every grounds entry a run's pick writes:
// "picked by run <run-id> on <date>". The token vocabulary is closed, so the
// marker lives in the text after the `pursued:` token (the intent's open
// question keeps the token's shape open).
const RunPickMarker = "picked by run "

// runPickRe is the marker as the gate recognises it.
var runPickRe = regexp.MustCompile(`^picked by run run-[0-9]+ on [0-9]{4}-[0-9]{2}-[0-9]{2}\b`)

// isRunPick reports whether a grounds entry is one a run's pick wrote.
func isRunPick(g grounds.Grounds) bool { return runPickRe.MatchString(g.Text) }

// PickEntryText is the reason the pick writes onto the chosen intent: the
// marker, every candidate with its score, the rule, the runner-up and why it
// lost, and the falsifier.
func PickEntryText(runID string, date time.Time, p Pick) string {
	var b strings.Builder
	b.WriteString(RunPickMarker + runID + " on " + date.UTC().Format("2006-01-02") + "; candidates: ")
	for i, c := range p.Candidates {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(candidateText(c))
	}
	b.WriteString("; rule: " + p.Rule)
	switch {
	case p.RunnerUp == nil:
		b.WriteString("; runner-up: none, " + p.Chosen.ID + " was the only candidate")
	case p.TieBrokenByAge:
		b.WriteString(fmt.Sprintf("; runner-up: %s, tied at %d and younger, so the tie was broken by age", p.RunnerUp.ID, p.RunnerUp.Score.Total))
	default:
		b.WriteString(fmt.Sprintf("; runner-up: %s, which lost on score (%d against %d)", p.RunnerUp.ID, p.RunnerUp.Score.Total, p.Chosen.Score.Total))
	}
	b.WriteString("; falsifier: " + p.Falsifier)
	return b.String()
}

// candidateText is one candidate as the reason names it.
func candidateText(c PickCandidate) string {
	s := c.Score
	t := c.ID + " " + strconv.Itoa(s.Total) + " (criteria " + strconv.Itoa(s.Criteria.Points) +
		", test path " + strconv.Itoa(s.TestPath.Points) + ", footprint " + strconv.Itoa(s.Footprint.Points)
	if s.NoFootprint {
		t += ", its spec carries no footprint"
	}
	return t + ")"
}
