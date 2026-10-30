package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/decide/judge"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// embark_lessons_test.go is embark's lesson ranking as a host judgement routed
// through the decision adapter (dcn-2610300523005947, spc-2609221011151661
// step 2), at the front door: `embark lessons` lists one score question per
// lesson, `--scores` takes the host's scores back and ranks by them, and the
// route of embark-lesson-ranking decides who is asked.

// lessonsView is the JSON the lessons command prints.
type lessonsView struct {
	RankedBy string `json:"ranked_by"`
	Top      []struct {
		ID     string `json:"id"`
		Text   string `json:"text"`
		Judged *int   `json:"judged"`
	} `json:"top"`
	Rest      []struct{ ID string } `json:"rest"`
	Unranked  bool                  `json:"unranked"`
	Questions []struct {
		ID          string `json:"id"`
		Lesson      string `json:"lesson"`
		Type        string `json:"type"`
		Kind        string `json:"kind"`
		Min         int    `json:"min"`
		Max         int    `json:"max"`
		StateSHA256 string `json:"state_sha256"`
	} `json:"questions"`
	Judgement *struct {
		Mode     string `json:"mode"`
		RankedBy string `json:"ranked_by"`
		Fallback string `json:"fallback"`
		Pairs    int    `json:"pairs"`
	} `json:"judgement"`
	Diagnostics []string `json:"diagnostics"`
}

func readLessonsView(t *testing.T, raw []byte) lessonsView {
	t.Helper()
	var v lessonsView
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, raw)
	}
	return v
}

// lessonsTarget is a lifeboat with five lessons and a target whose framing
// chapter is a brief.
func lessonsTarget(t *testing.T) (lb, target string) {
	t.Helper()
	lb = packEmbarkLifeboat(t, retroSourceRepo(t))
	target = t.TempDir()
	framing := filepath.Join(target, ".abcd", "development", "brief", "01-product", "06-framing.md")
	if err := os.MkdirAll(filepath.Dir(framing), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(framing, []byte("# Framing\n\nThis voyage audits every intent before each release cut and keeps the lifeboat small.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return lb, target
}

// writeScores writes a scores file answering every question of v, the
// lesson about the scanner scored 4 and every other 1.
func writeScores(t *testing.T, v lessonsView) string {
	t.Helper()
	type entry struct {
		ID          string `json:"id"`
		StateSHA256 string `json:"state_sha256"`
		Score       int    `json:"score"`
	}
	doc := struct {
		SchemaVersion int     `json:"schema_version"`
		Scores        []entry `json:"scores"`
	}{SchemaVersion: 1}
	for _, q := range v.Questions {
		s := 1
		if strings.Contains(q.Lesson, "scanner") {
			s = 4
		}
		doc.Scores = append(doc.Scores, entry{ID: q.ID, StateSHA256: q.StateSHA256, Score: s})
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "scores.json")
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// lessonsFake is the decision adapter: it scores the lesson about the
// changelog 4 and every other 0, or fails.
type lessonsFake struct {
	err   error
	calls int
}

func (f *lessonsFake) Judge(_ context.Context, s judge.State, _ judge.Question) (judge.Answer, error) {
	f.calls++
	if f.err != nil {
		return judge.Answer{}, f.err
	}
	if strings.Contains(s.Text, "changelog") {
		return judge.Answer{Value: "4", Probability: 0.8}, nil
	}
	return judge.Answer{Value: "0", Probability: 0.6}, nil
}

// withLessonsRouter routes embark-lesson-ranking in mode through adapter,
// pairs kept in got.
func withLessonsRouter(t *testing.T, mode string, adapter *lessonsFake, got *[]lab.Pair) {
	t.Helper()
	routes := map[judge.Type]oracle.Target{}
	if mode != oracle.JudgementHost {
		routes[judge.EmbarkLessonRanking] = oracle.Target{Provider: "openrouter", Model: "typesafe/jev-1.13", Mode: mode}
	}
	prev := loadLessonsRouter
	loadLessonsRouter = func(io.Writer, string) (*judge.Router, error) {
		return judge.NewForTest(routes, func(oracle.Target) judge.Judge { return adapter }, func(p lab.Pair) (string, string, error) {
			*got = append(*got, p)
			return "lab-261030000000-abcdef0", abcdhome.Display("lab/x/lab-261030000000-abcdef0/pairs.jsonl"), nil
		}), nil
	}
	t.Cleanup(func() { loadLessonsRouter = prev })
}

// The host is asked: one score question per lesson in the JSON, named in
// text; without scores the order is today's, named a heuristic; the host's
// scores rank, term overlap breaking ties.
func TestEmbarkLessonsAsksTheHostAndRanksByItsScores(t *testing.T) {
	lb, target := lessonsTarget(t)
	v := readLessonsView(t, runCLI(t, "embark", "lessons", lb, target, "--json"))
	if len(v.Questions) != 5 || v.RankedBy != "term-overlap" || v.Judgement == nil || v.Judgement.Mode != "host" {
		t.Fatalf("five questions, today's order: %+v", v)
	}
	for _, q := range v.Questions {
		if q.Type != "embark-lesson-ranking" || q.Kind != "score" || q.Min != 0 || q.Max != 4 || len(q.StateSHA256) != 64 {
			t.Fatalf("question = %+v", q)
		}
	}
	if v.Top[0].Text != "Audit every intent before the release cut." {
		t.Fatalf("today's term-overlap order: %+v", v.Top)
	}
	out := string(runCLI(t, "embark", "lessons", lb, target))
	for _, want := range []string{"heuristic", "embark-lesson-ranking", "--scores", v.Questions[0].ID, v.Questions[4].ID} {
		if !strings.Contains(out, want) {
			t.Errorf("text mode does not name %q:\n%s", want, out)
		}
	}

	scores := writeScores(t, v)
	got := readLessonsView(t, runCLI(t, "embark", "lessons", lb, target, "--scores", scores, "--json"))
	if got.RankedBy != "host" || len(got.Questions) != 0 || got.Top[0].Text != "Write the scanner's fixtures at runtime." ||
		got.Top[0].Judged == nil || *got.Top[0].Judged != 4 || got.Top[1].Text != "Audit every intent before the release cut." {
		t.Fatalf("ranked by the host's scores, overlap breaking ties: %+v", got)
	}
	out = string(runCLI(t, "embark", "lessons", lb, target, "--scores", scores))
	if !strings.Contains(out, "the host's scores") || !strings.Contains(out, "1. lesson-5 [v0.2.0] Write the scanner's fixtures at runtime.") {
		t.Fatalf("text mode names the host's ranking:\n%s", out)
	}
}

// A scores file out of shape, or one that is a link, is refused, exit 2,
// nothing ranked; with no brief nothing is asked and scores are refused.
func TestEmbarkLessonsRefusesAScoresFileOutOfShape(t *testing.T) {
	lb, target := lessonsTarget(t)
	v := readLessonsView(t, runCLI(t, "embark", "lessons", lb, target, "--json"))
	dir := t.TempDir()
	short := filepath.Join(dir, "short.json")
	if err := os.WriteFile(short, []byte(`{"schema_version":1,"scores":[{"id":"`+v.Questions[0].ID+`","state_sha256":"`+
		v.Questions[0].StateSHA256+`","score":2}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(writeScores(t, v), link); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"a lesson left out": short, "a link": link, "no file": filepath.Join(dir, "absent.json")} {
		out, err := runCLIErr(t, "embark", "lessons", lb, target, "--scores", path, "--json")
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != 2 || strings.Contains(string(out), `"top"`) {
			t.Errorf("%s: err %v, out %s", name, err, out)
		}
	}
	none := readLessonsView(t, runCLI(t, "embark", "lessons", lb, t.TempDir(), "--json"))
	if !none.Unranked || len(none.Questions) != 0 {
		t.Fatalf("no brief: unranked and nothing asked: %+v", none)
	}
	if _, err := runCLIErr(t, "embark", "lessons", lb, t.TempDir(), "--scores", writeScores(t, v)); err == nil ||
		!strings.Contains(err.Error(), "nothing was asked") {
		t.Fatalf("scores with no brief are refused: %v", err)
	}
}

// Shadow records one pair per lesson beside the host's scores; turned on, the
// adapter's scores rank and nothing is asked; an adapter that cannot answer
// leaves the host's questions standing with a fallback receipt.
func TestEmbarkLessonsRoutesThroughTheDecisionAdapter(t *testing.T) {
	lb, target := lessonsTarget(t)
	base := readLessonsView(t, runCLI(t, "embark", "lessons", lb, target, "--json"))
	scores := writeScores(t, base)

	adapter, pairs := &lessonsFake{}, []lab.Pair{}
	withLessonsRouter(t, oracle.JudgementShadow, adapter, &pairs)
	v := readLessonsView(t, runCLI(t, "embark", "lessons", lb, target, "--json"))
	if adapter.calls != 0 || len(v.Questions) != 5 {
		t.Fatalf("shadow without scores asks the host alone: calls %d %+v", adapter.calls, v)
	}
	v = readLessonsView(t, runCLI(t, "embark", "lessons", lb, target, "--scores", scores, "--json"))
	if adapter.calls != 5 || len(pairs) != 5 || v.RankedBy != "host" || v.Judgement.Pairs != 5 ||
		v.Top[0].Text != "Write the scanner's fixtures at runtime." {
		t.Fatalf("shadow pairs one per lesson, the host's scores rank: calls %d pairs %d %+v", adapter.calls, len(pairs), v)
	}

	on := &lessonsFake{}
	withLessonsRouter(t, oracle.JudgementOn, on, &pairs)
	v = readLessonsView(t, runCLI(t, "embark", "lessons", lb, target, "--json"))
	if on.calls != 5 || len(v.Questions) != 0 || v.RankedBy != "decision-adapter" || v.Top[0].Text != "Name the parser's refusals in the changelog." {
		t.Fatalf("turned on, the adapter ranks and nothing is asked: calls %d %+v", on.calls, v)
	}

	down := &lessonsFake{err: errors.New("connection refused")}
	withLessonsRouter(t, oracle.JudgementOn, down, &pairs)
	v = readLessonsView(t, runCLI(t, "embark", "lessons", lb, target, "--json"))
	if len(v.Questions) != 5 || v.RankedBy != "term-overlap" || v.Judgement == nil || !strings.Contains(v.Judgement.Fallback, "connection refused") {
		t.Fatalf("the host's request stands with a fallback receipt: %+v", v)
	}
	out := string(runCLI(t, "embark", "lessons", lb, target))
	if !strings.Contains(out, "connection refused") || !strings.Contains(out, "--scores") {
		t.Fatalf("text mode names the fallback and the host's request:\n%s", out)
	}
}

// A route naming a judgement type outside the registry is reported, not left
// silently on the host.
func TestEmbarkLessonsReportsARouteToAnUnknownType(t *testing.T) {
	lb, target := lessonsTarget(t)
	cfg := abcdhome.Path(os.Getenv("HOME"), "config.json")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter",
		"models":["typesafe/jev-1.13"]}},"judgements":{"embark-lesson-rankng":"shadow:openrouter/typesafe/jev-1.13"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runCLISplit(t, "embark", "lessons", lb, target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "oracle.judgements.embark-lesson-rankng") || !strings.Contains(stderr, "not a judgement type") {
		t.Fatalf("stderr names the unknown type:\n%s\nstdout:\n%s", stderr, stdout)
	}
	js, _, err := runCLISplit(t, "embark", "lessons", lb, target, "--json")
	if err != nil {
		t.Fatal(err)
	}
	v := readLessonsView(t, []byte(js))
	if len(v.Diagnostics) != 1 || !strings.Contains(v.Diagnostics[0], "embark-lesson-rankng") {
		t.Fatalf("the JSON carries the diagnostic: %+v", v.Diagnostics)
	}
}

// The configuration's layers are read as every front door reads them, and
// what they refuse is said: a HOME inside a git working tree is not read as
// the machine's layer, and one stderr line says so, with stdout still one
// JSON document.
func TestEmbarkLessonsSaysWhenTheHomeIsNotRead(t *testing.T) {
	lb, target := lessonsTarget(t)
	plantHome(t)
	stdout, stderr, err := runCLISplit(t, "embark", "lessons", lb, target, "--json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(stderr, "HOME lies inside the git working tree") != 1 || !strings.Contains(stderr, "abcd: HOME lies inside the git working tree") {
		t.Fatalf("stderr does not say the home was not read:\n%s", stderr)
	}
	if v := readLessonsView(t, []byte(stdout)); v.Judgement == nil || v.Judgement.Mode != "host" || len(v.Questions) != 5 {
		t.Fatalf("the refused home routes nothing: %+v", v)
	}
}

// The target's framing chapter is read inside the target: a directory on its
// path that leads out of the target is refused, nothing ranked, so no file
// outside it becomes the state the decision adapter is sent.
func TestEmbarkLessonsRefusesAFramingChapterOutsideTheTarget(t *testing.T) {
	lb, target := lessonsTarget(t)
	outside := t.TempDir()
	dev := filepath.Join(target, ".abcd", "development")
	if err := os.Rename(dev, filepath.Join(outside, "development")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "development"), dev); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runCLISplit(t, "embark", "lessons", lb, target, "--json")
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 2 || !strings.Contains(ee.Msg, "framing chapter") {
		t.Fatalf("a framing chapter outside the target is read: err %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
}
