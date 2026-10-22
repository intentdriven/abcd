package loop

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The production wiring of the tier escalation, held through Drive, the call
// `implement step` makes: the runner configuration names the runner a role
// runs on, the reviewer route and a role pointed at a provider name no tier,
// and the tables LoadTiering reads decide the tier.

// driveRound starts a run, takes its lane to its validators and drives one
// step with o and cfg, returning the step and the ruthless reviewer's route
// in round 1.
func driveRound(t *testing.T, o Options, rs Runners) (*gittest.Repo, string, StepResult, Round) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	implemented(t, repo, start.RunID, DefaultStages(), "one.txt")
	if rs.Transcripts == nil {
		rs.Transcripts = &memTranscripts{}
	}
	res, err := Drive(context.Background(), repo.Root(), start.RunID, DefaultStages(), o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("the step hands the ruthless reviewer out: %+v", res)
	}
	return repo, start.RunID, res, routeOfRole(t, repo, start.RunID, 1, RoleRuthless)
}

// machineRoots is a home holding the machine's files and the repository's
// root, for LoadTiering.
func machineRoots(t *testing.T, repo string, files map[string]string) layered.Roots {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(abcdhome.Path(home, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return layered.Roots{Repo: repo, Home: home}
}

// fakeReviewer is a reviewer route that is reachable.
type fakeReviewer struct{}

func (fakeReviewer) Review(context.Context, string) (string, error) { return "", nil }

// TestAStepWhoseReviewTheReviewerRouteTakesRecordsNoTier: with the reviewer
// route reachable, the ruthless review it takes names no tier; with the route
// named and no server for it, every review falls back to the host, so the
// review is asked for its row's tier like any host-run one.
func TestAStepWhoseReviewTheReviewerRouteTakesRecordsNoTier(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	roots := machineRoots(t, repo.Root(), map[string]string{"oracle-routing.json": `{"schema_version":1,"agents":{"ruthless-reviewer":{"tier":"economy"}}}`})
	tiers, _, err := LoadTiering(roots, &ReviewRoute{Name: oracle.ReviewRP, Reviewer: fakeReviewer{}})
	if err != nil {
		t.Fatal(err)
	}
	_, _, res, r := driveRound(t, Options{Tiers: tiers}, Runners{Config: runnerConfig(t, "", "")})
	if res.Tier != "" || r.Tier != "" || !strings.Contains(r.Note, "rp reviewer route") {
		t.Fatalf("a review the reviewer route takes is asked for no tier: %+v %+v", res.Tier, r)
	}
	absent, _, err := LoadTiering(roots, &ReviewRoute{Name: oracle.ReviewRP, Absent: "oracle.review is rp, and no oracle.rp.url is named"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, res, r = driveRound(t, Options{Tiers: absent}, Runners{Config: runnerConfig(t, "", "")})
	if res.Tier != oracle.Economy || r.Tier != oracle.Economy || strings.Contains(r.Note, "reviewer route") {
		t.Fatalf("a reviewer route with no server takes no review, so the host's is asked for its tier: %+v %+v", res.Tier, r)
	}
}
