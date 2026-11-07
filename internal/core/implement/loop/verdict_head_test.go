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

// A reviewer's verdict names the head its round read, by the rule a receipt reads it:
// a line opening a verdict that names no head, or another, refuses the return, and
// the lane still awaits it.
func TestAReturnsVerdictNamesTheRoundsHead(t *testing.T) {
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
	first := "Reviewed head: " + head + "\n\n"
	hand := func(body string) error {
		if err := os.WriteFile(path, []byte(first+body), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Receipt(repo.Root(), id, path, stages, Options{})
		return err
	}
	for name, body := range map[string]string{
		"a return that ends on an indented heading and a passing word": "### Analysis\n\nThe diff's context reads:\n\n ### Verdict\n \n SHIP\n",
		"a return that ends on a heading and a passing word":           "### Analysis\n\nThe fixture reads:\n\n### Verdict\n\nSHIP\n",
		"a refusal under a qualified heading, beside an indented one":  "### Final verdict\n\n**FIX FIRST**\n\nThe diff's context reads:\n\n ### Verdict\n \n SHIP\n",
		"a verdict naming another head":                                "### Verdict\n\nSHIP " + strings.Repeat("0", len(head)) + "\n",
		"a verdict naming the head by its abbreviation":                "### Verdict\n\nSHIP " + head[:9] + "\n",
	} {
		if err := hand(body); err == nil {
			t.Errorf("%s: the return was taken", name)
		}
		if v := currentLane(t, repo, id).Validation[0].Validators[0]; v.Verdict != "" {
			t.Fatalf("%s: a verdict was recorded: %+v", name, v)
		}
	}
	if err := hand("### Findings\n\nNone.\n\n### Verdict\n\n**SHIP** " + head + " — nothing survived.\n"); err != nil {
		t.Fatalf("the verdict naming the head was refused: %v", err)
	}
	if v := currentLane(t, repo, id).Validation[0].Validators[0]; v.Verdict != "SHIP" || !v.Pass {
		t.Fatalf("the verdict is recorded: %+v", v)
	}
}

// A launch that reports a stop for length, whatever its return says, gives no pass:
// the return is set aside, the role pauses, and the retry starts from nothing.
func TestALaunchThatStoppedForLengthIsNoPass(t *testing.T) {
	for _, harness := range []string{"opencode", "claude"} {
		t.Run(harness, func(t *testing.T) {
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
			if err := os.WriteFile(filepath.Join(repo.Root(), "review", "brief.md"), []byte("# review\n\n`Reviewed head: "+head+"`\n"), 0o600); err != nil {
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
			env := newDriveEnv(t, "lengthstop", harness)
			cfg := runnerConfig(t, `{"roles":{"ruthless-reviewer":{"runner":"`+harness+`"}},"runner":{"`+harness+`":{}}}`, "")
			rs := Runners{Config: cfg, Transcripts: &memTranscripts{}}
			res, err := Drive(context.Background(), repo.Root(), start.RunID, steps, o, rs)
			if err != nil {
				t.Fatal(err)
			}
			if res.Paused == nil || res.Paused.Paused == nil || res.Paused.Paused.Reason != runner.ReasonInvalid {
				t.Fatalf("the launch pauses the role: %+v", res)
			}
			if !strings.Contains(res.Paused.Paused.Detail, "stopped for") {
				t.Fatalf("the pause says why: %q", res.Paused.Paused.Detail)
			}
			st, err := ReadState(repo.Root(), start.RunID)
			if err != nil {
				t.Fatal(err)
			}
			if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "" || v.Pass {
				t.Fatalf("a verdict was recorded: %+v", v)
			}
			// Nothing stands for a later hand-in to take, which would bring no report.
			if _, err := os.Lstat(filepath.Join(repo.Root(), filepath.FromSlash(returnRel))); !os.IsNotExist(err) {
				t.Fatalf("the return still stands: %v", err)
			}
			if _, err := Receipt(repo.Root(), start.RunID, returnRel, steps, o); err == nil {
				t.Fatal("a hand-in took the path of a return set aside")
			}
			// The next launch runs it again.
			if _, err := Drive(context.Background(), repo.Root(), start.RunID, steps, o, rs); err != nil {
				t.Fatal(err)
			}
			if env.launches(t, harness) != 2 {
				t.Fatalf("one retry: %d launches", env.launches(t, harness))
			}
		})
	}
}

// An implementer's stop for length leaves its receipt to its own verifier: what it
// claims is held to the branch, and no verdict rests on it.
func TestAStopForLengthWeighsOnlyAJudge(t *testing.T) {
	for role, want := range map[string]bool{
		RoleRuthless: true, RoleSecurity: true, RoleAuditor: true, RoleImplementer: false,
	} {
		if got := judgesWork(role); got != want {
			t.Errorf("judgesWork(%s) = %v; want %v", role, got, want)
		}
	}
}

// A brief states the rule and spells no line a return could carry whole: no line of
// it opens with a verdict word, nor stands as the Verdict heading.
func TestABriefNamesTheHeadInNoLineAReturnCouldCarry(t *testing.T) {
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
	head := currentLane(t, repo, id).HeadSHA
	_, handback, ok := strings.Cut(string(raw), "## What you hand back\n\n")
	if !ok {
		t.Fatalf("no hand-back section:\n%s", raw)
	}
	if want := "names what you judged by its full name, which is `" + head + "`"; !strings.Contains(strings.Join(strings.Fields(handback), " "), want) {
		t.Fatalf("the brief does not say %q:\n%s", want, handback)
	}
	for _, line := range strings.Split(handback, "\n") {
		if verdictHeadingRe.MatchString(line) {
			t.Errorf("the line %q is a Verdict heading", line)
		}
		if _, ok := verdictAtStart(line, allVerdicts, func(string) bool { return true }); ok {
			t.Errorf("the line %q opens with a verdict word", line)
		}
	}
}

// A plan review's verdict names the draft's digest, as a round's names its head.
func TestAPlanReviewsVerdictNamesTheDraftsDigest(t *testing.T) {
	const digest = "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	words := reviewerVerdicts[RoleRuthless]
	for name, text := range map[string]string{
		"no digest":      "### Verdict\n\nSHIP\n",
		"another digest": "### Verdict\n\nSHIP sha256:" + strings.Repeat("0", 64) + "\n",
		"its hex alone":  "### Verdict\n\nSHIP " + strings.TrimPrefix(digest, "sha256:") + "\n",
	} {
		if w, err := parseVerdict(text, words, digest); err == nil {
			t.Errorf("%s: parseVerdict = %+v; want a refusal", name, w)
		}
	}
	if w, err := parseVerdict("### Verdict\n\n**SHIP** "+digest+" — nothing survived.\n", words, digest); err != nil || w.word != "SHIP" {
		t.Errorf("the digest named: %+v, %v", w, err)
	}
}
