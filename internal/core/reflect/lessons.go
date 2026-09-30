package reflect

import (
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/core/record/match"
)

// TopLessons is how many predecessor lessons embark shows ranked; the rest are
// a list opened on request (itd-24 decision 2, spec scope 7).
const TopLessons = 3

// Lesson is one lesson a retrospective carries, with the release it came from.
type Lesson struct {
	Release string `json:"release"`
	Text    string `json:"text"`
}

var lessonsHeadingRe = regexp.MustCompile(`^##\s+` + regexp.QuoteMeta(Lessons.Heading()) + `\s*$`)

// ReadLessons reads the lessons out of a retrospective: one per top-level
// bullet of its lessons section, or one per paragraph when the section has no
// bullets. Fenced and commented lines are not lessons.
func ReadLessons(readme []byte) []Lesson {
	text := string(readme)
	lines := strings.Split(text, "\n")
	release := ""
	if v, ok := frontmatter.ScalarString(frontmatter.Fields(lines)["release"].Value); ok {
		release = strings.TrimSpace(v)
	}
	mask := mdrecord.Mask(lines)
	start, end, ok := mdrecord.SectionLineRangeIn(lines, mask, lessonsHeadingRe)
	if !ok {
		return nil
	}
	var out []Lesson
	add := func(parts []string) {
		if t := strings.Join(strings.Fields(strings.Join(parts, " ")), " "); t != "" {
			out = append(out, Lesson{Release: release, Text: t})
		}
	}
	if blocks := mdrecord.BulletBlocks(lines, mask, start, end); len(blocks) > 0 {
		for _, bl := range blocks {
			parts := []string{mdrecord.TrimBulletPrefix(lines[bl.Start])}
			parts = append(parts, lines[bl.Start+1:bl.End]...)
			add(parts)
		}
		return out
	}
	var para []string
	for i := start; i < end; i++ {
		if mask[i] != 0 || strings.TrimSpace(lines[i]) == "" {
			add(para)
			para = nil
			continue
		}
		para = append(para, lines[i])
	}
	add(para)
	return out
}

// RankedLesson is a lesson with its score against the brief and the terms the
// two share, rarest first.
type RankedLesson struct {
	Lesson
	Score  float64  `json:"score"`
	Shared []string `json:"shared"`
}

// Ranking is the embark view of predecessor lessons: the few most like the new
// voyage's brief, the rest as a list, and the method named as a heuristic.
type Ranking struct {
	Heuristic string         `json:"heuristic"`
	Top       []RankedLesson `json:"top"`
	Rest      []RankedLesson `json:"rest"`
}

// RankLessons ranks lessons against framing, the new voyage's brief framing
// chapter, with the canonical term-overlap primitive (record/match): a
// lesson's score is the weighted share of its terms the framing holds, the
// weights taken across the lessons and the framing together. The best
// TopLessons come first, the rest follow best first; ties keep the order the
// lessons were given in.
func RankLessons(framing string, lessons []Lesson) Ranking {
	framingTerms := sortedTerms(framing)
	sets := [][]string{framingTerms}
	terms := make([][]string, len(lessons))
	for i, l := range lessons {
		terms[i] = sortedTerms(l.Text)
		sets = append(sets, terms[i])
	}
	w := match.NewWeights(sets)
	ranked := make([]RankedLesson, len(lessons))
	for i, l := range lessons {
		fwd, _, shared := match.Overlap(terms[i], framingTerms, w)
		if shared == nil {
			shared = []string{}
		}
		ranked[i] = RankedLesson{Lesson: l, Score: round3(fwd), Shared: shared}
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })
	n := TopLessons
	if n > len(ranked) {
		n = len(ranked)
	}
	return Ranking{
		Heuristic: match.Heuristic,
		Top:       ranked[:n],
		Rest:      append([]RankedLesson{}, ranked[n:]...),
	}
}

func sortedTerms(s string) []string {
	t := match.Terms(s)
	sort.Strings(t)
	return t
}

func round3(f float64) float64 {
	return float64(int64(f*1000+0.5)) / 1000
}
