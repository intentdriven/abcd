package lifeboat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/decide/judge"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
	corereflect "github.com/intentdriven/abcd/internal/core/reflect"
)

// retrospectives_test.go holds the lifeboat half of itd-24: disembark packs
// every retrospective a voyage produced (criterion 5), embark writes them back
// into the new voyage's durable record, and embark ranks their lessons against
// the new voyage's brief (criterion 6).

const retroRel = ".abcd/development/retrospectives/"

func retroDoc(tag string, lessons ...string) string {
	var b strings.Builder
	b.WriteString("---\nrelease: " + tag + "\n---\n\n# Retrospective for " + tag + "\n\n## What went well\n\nThings.\n\n## Lessons learned\n\n")
	for _, l := range lessons {
		b.WriteString("- " + l + "\n")
	}
	b.WriteString("\n## Metrics\n\n- Intents shipped: 1\n")
	return b.String()
}

// retroSourceFixture is the embarkable source plus two retrospectives, and three
// files in the store that are not retrospectives: a directory that is not a
// release tag, a second file beside a README, and a file at the store's root.
func retroSourceFixture(t *testing.T) string {
	t.Helper()
	source := embarkableSourceFixture(t)
	for rel, body := range map[string]string{
		retroRel + "v0.1.0/README.md":     retroDoc("v0.1.0", "Audit every intent before the cut, because the retrospective reads the verdicts."),
		retroRel + "v0.2.0/README.md":     retroDoc("v0.2.0", "Keep the lifeboat small, since embark reads it whole."),
		retroRel + "notatag/README.md":    retroDoc("notatag", "Not a release."),
		retroRel + "v0.2.0/scratch.md":    "a note beside the retrospective\n",
		retroRel + "README.md":            "the store's own readme\n",
		retroRel + "v0.3.0-rc1/README.md": retroDoc("v0.3.0-rc1", "A prerelease is not a release."),
	} {
		mustWrite(t, filepath.Join(source, filepath.FromSlash(rel)), []byte(body))
	}
	return source
}

// Criterion 5: every retrospective the voyage produced is in the lifeboat, and
// nothing else under the store is; the family is sealed by the record hash.
func TestPlanPacksEveryRetrospective(t *testing.T) {
	source := retroSourceFixture(t)
	lb, err := Plan(source)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, want := range []string{"retrospectives/v0.1.0/README.md", "retrospectives/v0.2.0/README.md"} {
		if !hasPlanFile(lb, want) {
			t.Errorf("the lifeboat does not carry %s:\n%s", want, planPaths(lb))
		}
		src, _ := os.ReadFile(filepath.Join(source, filepath.FromSlash(retroRel+strings.TrimPrefix(want, "retrospectives/"))))
		if got := planFile(t, lb, want); string(got.Content) != string(src) {
			t.Errorf("%s is not verbatim", want)
		}
		if !isRecordDerived(want) {
			t.Errorf("%s is not sealed by the record manifest hash", want)
		}
	}
	for _, not := range []string{"retrospectives/notatag/README.md", "retrospectives/v0.2.0/scratch.md", "retrospectives/README.md", "retrospectives/v0.3.0-rc1/README.md"} {
		if hasPlanFile(lb, not) {
			t.Errorf("the lifeboat carries %s, which is not a retrospective", not)
		}
	}
}

// The inverse mapping: only <release-tag>/README.md under the family maps back;
// a hostile or foreign name is unmapped and never written.
func TestResolveTargetMapsRetrospectives(t *testing.T) {
	fam, tgt, disp, _ := resolveTarget("retrospectives/v0.11.0/README.md")
	if disp != dispPlanned || fam != "retrospectives" || tgt != retroRel+"v0.11.0/README.md" {
		t.Fatalf("resolveTarget = (%q, %q, %v)", fam, tgt, disp)
	}
	for _, rel := range []string{
		"retrospectives/README.md",
		"retrospectives/v0.11.0/notes.md",
		"retrospectives/v0.11.0/sub/README.md",
		"retrospectives/notatag/README.md",
		"retrospectives/v0.11.0-rc1/README.md",
		"retrospectives/..%2f/README.md",
		"retrospectives/v01.2.3/README.md",
	} {
		if _, _, disp, _ := resolveTarget(rel); disp != dispUnmapped {
			t.Errorf("resolveTarget(%q) = %v, want unmapped", rel, disp)
		}
	}
	if !strings.HasSuffix(corereflect.RetrospectivesRelDir+"/", retroRel) {
		t.Errorf("the family's target %q is not the retrospective store %q", retroRel, corereflect.RetrospectivesRelDir)
	}
}

// Embark writes the retrospectives back, byte for byte, into the store the
// reflect verb reads and writes.
func TestEmbarkFromWritesRetrospectivesBack(t *testing.T) {
	source := retroSourceFixture(t)
	dest := packSource(t, source)
	target := t.TempDir()
	res, err := EmbarkFrom(dest, target)
	if err != nil {
		t.Fatalf("EmbarkFrom: %v", err)
	}
	if res.Families["retrospectives"] != 2 {
		t.Errorf("families = %v, want two retrospectives", res.Families)
	}
	for _, tag := range []string{"v0.1.0", "v0.2.0"} {
		rel := retroRel + tag + "/README.md"
		s, _ := os.ReadFile(filepath.Join(source, filepath.FromSlash(rel)))
		g, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(rel)))
		if err != nil || string(s) != string(g) {
			t.Errorf("%s did not land verbatim: %v", rel, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(target, filepath.FromSlash(retroRel+"notatag"))); err == nil {
		t.Error("embark wrote a directory that is not a release")
	}
}

// Criterion 6: the lessons the lifeboat carries, ranked against the new
// voyage's brief, the few first and the rest as a list.
func TestPredecessorLessonsRanksTheFewMostLikeTheBrief(t *testing.T) {
	source := embarkableSourceFixture(t)
	mustWrite(t, filepath.Join(source, filepath.FromSlash(retroRel+"v0.1.0/README.md")), []byte(retroDoc("v0.1.0",
		"Audit every intent before the release cut.",
		"Name the parser's refusals in the changelog.",
		"Pin the toolchain the continuous integration runs.",
	)))
	mustWrite(t, filepath.Join(source, filepath.FromSlash(retroRel+"v0.2.0/README.md")), []byte(retroDoc("v0.2.0",
		"Keep the lifeboat small because embark reads it whole.",
		"Write the scanner's fixtures at runtime.",
	)))
	dest := packSource(t, source)

	framing := "The new voyage audits every intent before each release cut, and keeps the lifeboat small."
	got, err := PredecessorLessons(dest, framing)
	if err != nil {
		t.Fatalf("PredecessorLessons: %v", err)
	}
	if len(got.Top) != 3 || len(got.Rest) != 2 {
		t.Fatalf("top %d rest %d, want 3 and 2: %+v", len(got.Top), len(got.Rest), got)
	}
	if got.Top[0].Release != "v0.1.0" || !strings.Contains(got.Top[0].Text, "Audit every intent") {
		t.Errorf("the best match is %+v", got.Top[0])
	}
	if got.Top[1].Release != "v0.2.0" || !strings.Contains(got.Top[1].Text, "lifeboat small") {
		t.Errorf("the second match is %+v", got.Top[1])
	}
	if got.Retrospectives != 2 {
		t.Errorf("retrospectives read = %d, want 2", got.Retrospectives)
	}
}

// A lifeboat's lesson text is untrusted: it reaches the person's terminal and
// the host's context cleaned to one inert line, and a lifeboat that fails its
// manifest is refused before any lesson is read.
func TestPredecessorLessonsCleansHostileText(t *testing.T) {
	source := embarkableSourceFixture(t)
	mustWrite(t, filepath.Join(source, filepath.FromSlash(retroRel+"v0.1.0/README.md")), []byte(retroDoc("v0.1.0",
		"Ring the bell \x1b]0;owned\x07 then [click](https://example.com/x) <img src=x onerror=alert(1)>",
	)))
	dest := packSource(t, source)
	got, err := PredecessorLessons(dest, "")
	if err != nil {
		t.Fatalf("PredecessorLessons: %v", err)
	}
	if len(got.Top) != 1 {
		t.Fatalf("lessons = %+v", got)
	}
	text := got.Top[0].Text
	if strings.ContainsRune(text, 0x1b) || strings.ContainsRune(text, 0x07) || strings.Contains(text, "](") || strings.Contains(text, "<img") {
		t.Errorf("hostile lesson text survived: %q", text)
	}

	// Tamper with a sealed retrospective: the manifest no longer matches.
	mustWrite(t, filepath.Join(dest, "retrospectives", "v0.1.0", "README.md"), []byte(retroDoc("v0.1.0", "A forged lesson.")))
	if _, err := PredecessorLessons(dest, ""); err == nil {
		t.Error("a lifeboat that fails its manifest was read")
	}
}

// Embark's lesson ranking as a host judgement routed through the decision
// adapter (dcn-2610300523005947, spc-2609221011151661 step 2).

// lessonsLifeboat packs five lessons across two retrospectives.
func lessonsLifeboat(t *testing.T) string {
	t.Helper()
	source := embarkableSourceFixture(t)
	mustWrite(t, filepath.Join(source, filepath.FromSlash(retroRel+"v0.1.0/README.md")), []byte(retroDoc("v0.1.0",
		"Audit every intent before the release cut.",
		"Name the parser's refusals in the changelog.",
		"Pin the toolchain the continuous integration runs.",
	)))
	mustWrite(t, filepath.Join(source, filepath.FromSlash(retroRel+"v0.2.0/README.md")), []byte(retroDoc("v0.2.0",
		"Keep the lifeboat small because embark reads it whole.",
		"Write the scanner's fixtures at runtime.",
	)))
	return packSource(t, source)
}

const lessonsFraming = "The new voyage audits every intent before each release cut, and keeps the lifeboat small."

// scoresFor is a scores file answering every question of v, lesson i scored
// score(i, lesson text).
func scoresFor(t *testing.T, v LessonsView, score func(q LessonQuestion) int) []byte {
	t.Helper()
	doc := LessonScores{SchemaVersion: LessonScoresSchemaVersion}
	for _, q := range v.Questions {
		s := score(q)
		doc.Scores = append(doc.Scores, LessonScore{ID: q.ID, StateSHA256: q.StateSHA256, Score: &s})
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func rankedIDs(v LessonsView) string {
	var ids []string
	for _, l := range append(append([]corereflect.RankedLesson{}, v.Top...), v.Rest...) {
		ids = append(ids, l.ID)
	}
	return strings.Join(ids, " ")
}

// scoreFake is the adapter: it scores a lesson by a word in it, or fails.
type scoreFake struct {
	err   error
	calls int
	seen  []judge.Question
	// slow is how long each answer takes, past any deadline the caller set.
	slow time.Duration
}

func (f *scoreFake) Judge(_ context.Context, s judge.State, q judge.Question) (judge.Answer, error) {
	f.calls++
	f.seen = append(f.seen, q)
	time.Sleep(f.slow)
	if f.err != nil {
		return judge.Answer{}, f.err
	}
	v := "0"
	if strings.Contains(s.Text, "scanner") {
		v = "4"
	}
	return judge.Answer{Value: v, Probability: 0.75, Model: "typesafe/jev-1.13-0901"}, nil
}

func lessonsRouter(mode string, adapter *scoreFake, pairs *[]lab.Pair) *judge.Router {
	routes := map[judge.Type]oracle.Target{}
	if mode != oracle.JudgementHost {
		routes[judge.EmbarkLessonRanking] = oracle.Target{Provider: "openrouter", Model: "typesafe/jev-1.13", Mode: mode}
	}
	return judge.NewForTest(routes, func(oracle.Target) judge.Judge { return adapter }, func(p lab.Pair) (string, string, error) {
		*pairs = append(*pairs, p)
		return "lab-261030000000-abcdef0", abcdhome.Display("lab/x/lab-261030000000-abcdef0/pairs.jsonl"), nil
	})
}

// One score question per lesson, on a scale of 0 to 4, its state the brief's
// framing and the one lesson; the lessons stay in today's term-overlap order,
// named a heuristic. With no brief nothing is asked.
func TestPredecessorLessonsAsksOneScoreQuestionPerLesson(t *testing.T) {
	dest := lessonsLifeboat(t)
	v, err := PredecessorLessons(dest, lessonsFraming)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Questions) != 5 || v.RankedBy != corereflect.RankedByTermOverlap || !strings.Contains(v.Heuristic, "heuristic") {
		t.Fatalf("questions %d, ranked by %q: %+v", len(v.Questions), v.RankedBy, v)
	}
	seen := map[string]bool{}
	for i, q := range v.Questions {
		if q.ID != fmt.Sprintf("lesson-%d", i+1) || q.Type != string(judge.EmbarkLessonRanking) || q.Kind != string(judge.KindScore) ||
			q.Min != 0 || q.Max != 4 || len(q.StateSHA256) != 64 || q.Lesson == "" || q.Release == "" ||
			!strings.Contains(q.Ask, "apply to the new voyage") || seen[q.StateSHA256] {
			t.Fatalf("question %d = %+v", i, q)
		}
		seen[q.StateSHA256] = true
	}
	if v.Top[0].ID != "lesson-1" || v.Top[1].ID != "lesson-4" {
		t.Fatalf("today's order: %s", rankedIDs(v))
	}
	// Ranked without scores, on the host, nothing changes and nothing is asked of an adapter.
	adapter, pairs := &scoreFake{}, []lab.Pair{}
	for _, mode := range []string{oracle.JudgementHost, oracle.JudgementShadow} {
		got, err := RankPredecessorLessons(context.Background(), v, lessonsRouter(mode, adapter, &pairs), nil)
		if err != nil || rankedIDs(got) != rankedIDs(v) || len(got.Questions) != 5 || got.Judgement == nil ||
			got.Judgement.RankedBy != corereflect.RankedByTermOverlap || got.Judgement.Mode != mode {
			t.Fatalf("%s without scores: %+v %v", mode, got, err)
		}
	}
	if adapter.calls != 0 || len(pairs) != 0 {
		t.Fatalf("no adapter asked and no pair kept without the host's scores: %d %d", adapter.calls, len(pairs))
	}

	none, err := PredecessorLessons(dest, "")
	if err != nil {
		t.Fatal(err)
	}
	if !none.Unranked || len(none.Questions) != 0 {
		t.Fatalf("no brief, nothing asked: %+v", none)
	}
	got, err := RankPredecessorLessons(context.Background(), none, lessonsRouter(oracle.JudgementOn, adapter, &pairs), nil)
	if err != nil || adapter.calls != 0 || !got.Unranked || len(got.Questions) != 0 || got.Judgement != nil {
		t.Fatalf("no brief: nothing is asked of the adapter either: %+v %v calls %d", got, err, adapter.calls)
	}
	if _, err := RankPredecessorLessons(context.Background(), none, nil, []byte(`{"schema_version":1,"scores":[]}`)); err == nil ||
		!strings.Contains(err.Error(), "nothing was asked") {
		t.Fatalf("scores with no brief are refused: %v", err)
	}
}

// The host's scores rank the lessons, term overlap breaking ties, and the
// questions are answered.
func TestRankPredecessorLessonsByTheHostsScores(t *testing.T) {
	v, err := PredecessorLessons(lessonsLifeboat(t), lessonsFraming)
	if err != nil {
		t.Fatal(err)
	}
	raw := scoresFor(t, v, func(q LessonQuestion) int {
		if strings.Contains(q.Lesson, "toolchain") || strings.Contains(q.Lesson, "scanner") {
			return 3
		}
		return 1
	})
	got, err := RankPredecessorLessons(context.Background(), v, nil, raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Top[0].ID != "lesson-3" || got.Top[1].ID != "lesson-5" || got.RankedBy != corereflect.RankedByHost ||
		len(got.Questions) != 0 || got.Judgement == nil || got.Judgement.RankedBy != corereflect.RankedByHost {
		t.Fatalf("ranked by the host's scores: %s %+v", rankedIDs(got), got.Judgement)
	}
	if got.Top[2].ID != "lesson-1" {
		t.Fatalf("overlap breaks the tie among the 1s: %s", rankedIDs(got))
	}
	if j := got.Top[0].Judged; j == nil || *j != 3 {
		t.Fatalf("the lesson carries its score: %+v", got.Top[0])
	}
}

// A scores file is held to the questions: every lesson scored exactly once,
// over the state it was asked over, on the scale; anything else is refused
// with nothing ranked.
func TestRankPredecessorLessonsRefusesAScoresFileOutOfShape(t *testing.T) {
	v, err := PredecessorLessons(lessonsLifeboat(t), lessonsFraming)
	if err != nil {
		t.Fatal(err)
	}
	q := v.Questions
	entry := func(id, sha string, score string) string {
		return `{"id":"` + id + `","state_sha256":"` + sha + `","score":` + score + `}`
	}
	all := func(override func(i int) string) string {
		var parts []string
		for i, x := range q {
			if e := override(i); e != "" {
				parts = append(parts, e)
				continue
			}
			parts = append(parts, entry(x.ID, x.StateSHA256, "2"))
		}
		return `{"schema_version":1,"scores":[` + strings.Join(parts, ",") + `]}`
	}
	keep := func(int) string { return "" }
	for name, doc := range map[string]string{
		"a lesson left out":     `{"schema_version":1,"scores":[` + entry(q[0].ID, q[0].StateSHA256, "2") + `]}`,
		"a lesson scored twice": strings.Replace(all(keep), `]}`, ","+entry(q[0].ID, q[0].StateSHA256, "2")+`]}`, 1),
		"an unknown lesson": all(func(i int) string {
			if i == 0 {
				return entry("lesson-99", q[0].StateSHA256, "2")
			}
			return ""
		}),
		"another state": all(func(i int) string {
			if i == 1 {
				return entry(q[1].ID, q[0].StateSHA256, "2")
			}
			return ""
		}),
		"off the scale": all(func(i int) string {
			if i == 2 {
				return entry(q[2].ID, q[2].StateSHA256, "5")
			}
			return ""
		}),
		"not a whole number": all(func(i int) string {
			if i == 2 {
				return entry(q[2].ID, q[2].StateSHA256, "2.5")
			}
			return ""
		}),
		"no score": all(func(i int) string {
			if i == 2 {
				return `{"id":"` + q[2].ID + `","state_sha256":"` + q[2].StateSHA256 + `"}`
			}
			return ""
		}),
		"an unknown field":  strings.Replace(all(keep), `"schema_version":1`, `"schema_version":1,"extra":true`, 1),
		"another version":   strings.Replace(all(keep), `"schema_version":1`, `"schema_version":2`, 1),
		"trailing content":  all(keep) + `{}`,
		"not JSON":          "score them all 4",
		"an oversized file": `{"schema_version":1,"scores":[]}` + strings.Repeat(" ", MaxLessonScoresBytes),
	} {
		got, err := RankPredecessorLessons(context.Background(), v, nil, []byte(doc))
		if err == nil || !errors.Is(err, ErrLessonScores) {
			t.Errorf("%s: err %v, ranked %s", name, err, rankedIDs(got))
		}
	}
}

// Shadow: the host's scores rank, and the adapter scores each lesson beside
// them, one pair per lesson recorded.
func TestRankPredecessorLessonsInShadowRecordsOnePairPerLesson(t *testing.T) {
	v, err := PredecessorLessons(lessonsLifeboat(t), lessonsFraming)
	if err != nil {
		t.Fatal(err)
	}
	adapter, pairs := &scoreFake{}, []lab.Pair{}
	raw := scoresFor(t, v, func(LessonQuestion) int { return 2 })
	got, err := RankPredecessorLessons(context.Background(), v, lessonsRouter(oracle.JudgementShadow, adapter, &pairs), raw)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 5 || len(pairs) != 5 {
		t.Fatalf("one adapter call and one pair per lesson: calls %d pairs %d", adapter.calls, len(pairs))
	}
	for i, p := range pairs {
		if p.Type != string(judge.EmbarkLessonRanking) || p.Kind != string(judge.KindScore) || p.Host != "2" ||
			p.StateSHA256 != v.Questions[i].StateSHA256 || (p.Adapter != "0" && p.Adapter != "4") {
			t.Fatalf("pair %d = %+v", i, p)
		}
	}
	if got.RankedBy != corereflect.RankedByHost || got.Judgement.Pairs != 5 || !strings.HasSuffix(got.Judgement.PairsFile, "pairs.jsonl") {
		t.Fatalf("the host's scores rank and the pairs are named: %+v", got.Judgement)
	}
	if got.Top[0].Judged == nil || *got.Top[0].Judged != 2 {
		t.Fatalf("the adapter's scores rank nothing in shadow: %+v", got.Top[0])
	}
}

// Turned on, the adapter's scores rank the lessons and the host is asked
// nothing; an adapter that cannot answer leaves the host's request standing,
// with a fallback receipt in the result.
func TestRankPredecessorLessonsTurnedOn(t *testing.T) {
	v, err := PredecessorLessons(lessonsLifeboat(t), lessonsFraming)
	if err != nil {
		t.Fatal(err)
	}
	adapter, pairs := &scoreFake{}, []lab.Pair{}
	got, err := RankPredecessorLessons(context.Background(), v, lessonsRouter(oracle.JudgementOn, adapter, &pairs), nil)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 5 || len(pairs) != 0 || len(got.Questions) != 0 || got.Top[0].ID != "lesson-5" ||
		got.RankedBy != corereflect.RankedByAdapter || got.Judgement.RankedBy != corereflect.RankedByAdapter ||
		got.Judgement.Target != "openrouter/typesafe/jev-1.13" || got.Judgement.ModelReported != "typesafe/jev-1.13-0901" {
		t.Fatalf("the adapter's scores rank and nothing is asked: calls %d, %s, %+v", adapter.calls, rankedIDs(got), got.Judgement)
	}
	if p := got.Top[0].Probability; p == nil || *p != 0.75 {
		t.Fatalf("the adapter's probability is carried: %+v", got.Top[0])
	}
	for _, q := range adapter.seen {
		if q.Type != judge.EmbarkLessonRanking || q.Kind != judge.KindScore || q.Min != 0 || q.Max != 4 {
			t.Fatalf("question = %+v", q)
		}
	}

	failing := &scoreFake{err: errors.New("connection refused")}
	got, err = RankPredecessorLessons(context.Background(), v, lessonsRouter(oracle.JudgementOn, failing, &pairs), nil)
	if err != nil {
		t.Fatal(err)
	}
	if failing.calls != 1 || len(got.Questions) != 5 || got.RankedBy != corereflect.RankedByTermOverlap || got.Judgement == nil ||
		!strings.Contains(got.Judgement.Fallback, "connection refused") || !strings.Contains(got.Judgement.Fallback, "the host's request stands") {
		t.Fatalf("the host's request stands with a receipt: calls %d %+v", failing.calls, got.Judgement)
	}
	// The host's scores answer the request that stood.
	got, err = RankPredecessorLessons(context.Background(), v, lessonsRouter(oracle.JudgementOn, failing, &pairs),
		scoresFor(t, v, func(LessonQuestion) int { return 4 }))
	if err != nil || got.RankedBy != corereflect.RankedByHost || failing.calls != 1 {
		t.Fatalf("the host's scores are taken: %+v %v calls %d", got.Judgement, err, failing.calls)
	}
}

// withLessonBudgets shortens the ranking's overall time for the adapter, on
// and in shadow, for the length of one test.
func withLessonBudgets(t *testing.T, on, shadow time.Duration) {
	t.Helper()
	oldOn, oldShadow := lessonAdapterBudget, lessonShadowBudget
	lessonAdapterBudget, lessonShadowBudget = on, shadow
	t.Cleanup(func() { lessonAdapterBudget, lessonShadowBudget = oldOn, oldShadow })
}

// Shadow stops asking at the first lesson whose pair is recorded as failed: a
// provider that hangs costs one deadline, not one per lesson. The lessons not
// reached keep no pair, and the note names them and says why; the host's
// scores still rank.
func TestRankPredecessorLessonsInShadowStopsAtTheFirstFailedPair(t *testing.T) {
	v, err := PredecessorLessons(lessonsLifeboat(t), lessonsFraming)
	if err != nil {
		t.Fatal(err)
	}
	adapter, pairs := &scoreFake{err: context.DeadlineExceeded}, []lab.Pair{}
	got, err := RankPredecessorLessons(context.Background(), v, lessonsRouter(oracle.JudgementShadow, adapter, &pairs),
		scoresFor(t, v, func(LessonQuestion) int { return 2 }))
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 1 || len(pairs) != 1 || pairs[0].Skipped != lab.SkippedFailed || pairs[0].StateSHA256 != v.Questions[0].StateSHA256 {
		t.Fatalf("asked once, one failed pair: calls %d pairs %+v", adapter.calls, pairs)
	}
	jd := got.Judgement
	if jd == nil || jd.Pairs != 1 || got.RankedBy != corereflect.RankedByHost || len(got.Questions) != 0 {
		t.Fatalf("the host's scores rank, one pair kept: %+v", jd)
	}
	for _, want := range []string{"4 lesson(s) not asked", "lesson-2 through lesson-5", "keep no pair", "lesson-1", "failed"} {
		if !strings.Contains(jd.PairsNote, want) {
			t.Fatalf("the note names the lessons not asked and why (%q): %q", want, jd.PairsNote)
		}
	}
}

// Shadow has one deadline for the whole pass: once it is spent, no further
// lesson is asked, even of an adapter that answers.
func TestRankPredecessorLessonsInShadowStopsWhenItsTimeRunsOut(t *testing.T) {
	withLessonBudgets(t, time.Minute, 20*time.Millisecond)
	v, err := PredecessorLessons(lessonsLifeboat(t), lessonsFraming)
	if err != nil {
		t.Fatal(err)
	}
	adapter, pairs := &scoreFake{slow: 60 * time.Millisecond}, []lab.Pair{}
	got, err := RankPredecessorLessons(context.Background(), v, lessonsRouter(oracle.JudgementShadow, adapter, &pairs),
		scoresFor(t, v, func(LessonQuestion) int { return 2 }))
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 1 || len(pairs) != 1 || pairs[0].Skipped != "" || got.Judgement.Pairs != 1 {
		t.Fatalf("one lesson asked before the time ran out: calls %d pairs %+v", adapter.calls, pairs)
	}
	for _, want := range []string{"4 lesson(s) not asked", "lesson-2 through lesson-5", "ran out"} {
		if !strings.Contains(got.Judgement.PairsNote, want) {
			t.Fatalf("the note says the time ran out (%q): %q", want, got.Judgement.PairsNote)
		}
	}
}

// Turned on, the ranking has one deadline for the whole pass too: once it is
// spent the host's request stands, with a receipt saying so.
func TestRankPredecessorLessonsTurnedOnStopsWhenItsTimeRunsOut(t *testing.T) {
	withLessonBudgets(t, 20*time.Millisecond, time.Minute)
	v, err := PredecessorLessons(lessonsLifeboat(t), lessonsFraming)
	if err != nil {
		t.Fatal(err)
	}
	adapter, pairs := &scoreFake{slow: 60 * time.Millisecond}, []lab.Pair{}
	got, err := RankPredecessorLessons(context.Background(), v, lessonsRouter(oracle.JudgementOn, adapter, &pairs), nil)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.calls != 1 || len(got.Questions) != 5 || got.RankedBy != corereflect.RankedByTermOverlap || got.Judgement == nil ||
		!strings.Contains(got.Judgement.Fallback, "at lesson-2") || !strings.Contains(got.Judgement.Fallback, "ran out") ||
		!strings.Contains(got.Judgement.Fallback, "the host's request stands") {
		t.Fatalf("the host's request stands once the time is spent: calls %d %+v", adapter.calls, got.Judgement)
	}
}
