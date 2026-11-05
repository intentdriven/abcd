package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// TestAReturnMustStateTheRoundsHead: the hand-in a host's reviewer goes through
// refuses a return with no such first line, naming the line, and one naming another
// head; the same reviewer's return with the line is recorded.
func TestAReturnMustStateTheRoundsHead(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := startedOnce(t, repo, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implementedOnce(t, repo, id, "one.txt")
	res, err := advance(repo.Root(), id, stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("the ruthless reviewer is handed out: %+v %v", res, err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))
	head := currentLane(t, repo, id).HeadSHA
	hand := func(body string) error {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Receipt(repo.Root(), id, path, stages, Options{})
		return err
	}
	want := "Reviewed head: " + head[:9]
	if err := hand(reviewerReturn("SHIP")); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("a return with no such line is refused, naming it: %v", err)
	}
	other := strings.Repeat("a", len(head))
	if err := hand("Reviewed head: " + other + "\n\n" + reviewerReturn("SHIP")); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("a return naming another head is refused, naming the right one: %v", err)
	}
	if err := hand("Reviewed head: " + head + "\n\n" + reviewerReturn("SHIP")); err != nil {
		t.Fatalf("the line lets it in: %v", err)
	}
	if v := currentLane(t, repo, id).Validation[0].Validators[0]; v.Verdict != "SHIP" {
		t.Fatalf("the verdict is recorded: %+v", v)
	}
}

// TestARunnersReviewerIsRetriedWithTheLineNamed: a runner's reviewer that leaves the
// line out pauses; the next step starts it again with a brief saying exactly what is
// missing, and with the line it passes.
func TestARunnersReviewerIsRetriedWithTheLineNamed(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := startedOnce(t, repo, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	head := strings.Repeat("a", 40)
	steps := driveSteps(t.TempDir())
	const briefRel, returnRel = "review/brief.md", "review/return.md"
	if err := os.MkdirAll(filepath.Join(repo.Root(), "review"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root(), "review", "brief.md"), []byte("# review\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	steps[2] = StageDef{Name: StageImplement, Piece: 7,
		Run: func(c Context, l *Lane) (Outcome, error) {
			if len(l.Validation) == 0 {
				l.Validation = []ValidationRound{{Round: 1, HeadSHA: head,
					Validators: []ValidatorRun{{Role: RoleRuthless, Brief: briefRel, Return: returnRel}}}}
			}
			return Outcome{Await: &Await{Role: RoleRuthless, Brief: briefRel, Receipt: returnRel}}, nil
		},
		Verify: func(c Context, l *Lane, receipt string) error {
			c.Await.Role = RoleRuthless
			return verifyValidation(c, l, receipt)
		}}
	o := Options{Now: func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) }}
	for range 2 {
		if _, err := advance(repo.Root(), start.RunID, steps, o); err != nil {
			t.Fatal(err)
		}
	}
	env := newDriveEnv(t, "headonretry", "opencode")
	cfg := runnerConfig(t, `{"roles":{"ruthless-reviewer":{"runner":"opencode"}},"runner":{"opencode":{}}}`, "")
	rs := Runners{Config: cfg, Transcripts: &memTranscripts{}}
	res, err := Drive(context.Background(), repo.Root(), start.RunID, steps, o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused == nil || res.Paused.Paused == nil || res.Paused.Paused.Reason != runner.ReasonInvalid {
		t.Fatalf("the answer with no line pauses the role: %+v", res)
	}
	if !strings.Contains(res.Paused.Paused.Detail, "Reviewed head: "+head[:9]) {
		t.Fatalf("the pause names the line: %q", res.Paused.Paused.Detail)
	}
	if _, err := Drive(context.Background(), repo.Root(), start.RunID, steps, o, rs); err != nil {
		t.Fatal(err)
	}
	if env.launches(t, "opencode") != 2 {
		t.Fatalf("one retry: %d launches", env.launches(t, "opencode"))
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "SHIP" {
		t.Fatalf("with the line it passes: %+v", v)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), "review", "brief.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(brief), retryHeading) || !strings.Contains(string(brief), "`Reviewed head: "+head+"`") {
		t.Fatalf("the retry's brief says what is missing:\n%s", brief)
	}
}

// TestTheBriefsOpenTheReturnWithTheLine: every reviewer's brief, whoever runs it, and a
// routed request put the line first under what it hands back.
func TestTheBriefsOpenTheReturnWithTheLine(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := startedOnce(t, repo, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implementedOnce(t, repo, id, "one.txt")
	res, err := advance(repo.Root(), id, stages, Options{})
	if err != nil || res.Awaiting == nil {
		t.Fatalf("a reviewer is handed out: %+v %v", res, err)
	}
	raw, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	brief := string(raw)
	head := currentLane(t, repo, id).HeadSHA
	_, after, ok := strings.Cut(brief, "## What you hand back\n\n")
	if !ok || !strings.HasPrefix(after, "Its first line reads exactly `Reviewed head: "+head+"`") {
		t.Fatalf("the line opens what the reviewer hands back:\n%s", brief)
	}
}
