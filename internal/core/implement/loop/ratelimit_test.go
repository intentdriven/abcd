package loop

// ratelimit_test.go is criterion 8 of itd-2609201925079472 (spc-2609301921521360,
// the rate-limit checkpoint): a runner's rate-limit response mid-lane, with two
// lanes in flight, ends the window early for the whole run; every lane with
// work in flight is checkpointed to its own branch, next_eligible_at is
// written once, and the record names the lane the response came from. The
// runner is the fake claude harness on PATH, playing a provider that refuses
// the run at a usage limit.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// rateLimitEnv puts the fake claude harness, in the given mode, first on PATH,
// ahead of git and the stub forge the fixture needs.
func rateLimitEnv(t *testing.T, mode string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.Symlink(self, filepath.Join(bin, "claude")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(loopFakeEnv, mode)
	t.Setenv(loopFakeLogEnv, t.TempDir())
}

// TestARateLimitResponseCheckpointsEveryLaneAndEndsTheWindow: lane 1 has its
// implementer out on the host and lane 2's implementer is routed to the claude
// runner, which answers with a rate-limit response.
func TestARateLimitResponseCheckpointsEveryLaneAndEndsTheWindow(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("2"), Pace: strp("120/300")})
	f.stepUntil(t, "lane 1's implementer is out and lane 2 is ready for its own", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && st.Lanes[1].Stage == StageImplement && len(st.Lanes[1].Awaits) == 0
	})
	oneHead := laneCommit(t, f.repo, f.lane(t, "lane-1"), "one.txt")
	twoHead := strings.TrimSpace(f.repo.Git("rev-parse", "refs/heads/"+f.lane(t, "lane-2").Branch))

	rateLimitEnv(t, "ratelimit")
	store := &memTranscripts{}
	f.now = f.now.Add(10 * time.Minute)
	res, err := Drive(context.Background(), f.repo.Root(), f.runID, f.stages, f.opts(),
		Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: store})
	if err != nil {
		t.Fatalf("a rate-limit response is not a failure of the step: %v", err)
	}
	until := f.now.Add(300 * time.Minute)
	if res.RateLimit == nil || res.RateLimit.Lane != "lane-2" || res.RateLimit.Role != RoleImplementer || res.RateLimit.Runner != runner.Claude {
		t.Fatalf("the result names the lane, the role and the runner the response came from: %+v", res.RateLimit)
	}
	if res.NextEligibleAt == nil || !res.NextEligibleAt.Equal(until) || !strings.Contains(res.Next, until.Format(time.RFC3339)) {
		t.Fatalf("the window ends early with next_eligible_at %s: %+v", until, res)
	}
	if len(store.stored) != 1 {
		t.Fatalf("the rate-limited run's transcript is stored: %v", store.stored)
	}

	st := f.state(t)
	if st.NextEligibleAt == nil || !st.NextEligibleAt.Equal(until) {
		t.Fatalf("next_eligible_at is written into the state: %v", st.NextEligibleAt)
	}
	if len(st.Fallbacks) != 0 {
		t.Fatalf("a rate limit is not a fallback to the host: %+v", st.Fallbacks)
	}
	if len(st.Lanes[1].Awaits) != 0 || len(st.Lanes[0].Awaits) != 1 {
		t.Fatalf("the rate-limited agent's slot is freed and lane 1's agent is still out: %+v", st.Lanes)
	}
	text := recordText(st)
	for _, want := range []string{
		"lane-2 " + StageRateLimit + " the implementer's runner claude answered with a rate-limit response",
		"lane-1 " + StageCheckpoint + " lane-1 checkpointed to " + st.Lanes[0].Branch + " at " + shortSHA(oneHead),
		"lane-2 " + StageCheckpoint + " lane-2 checkpointed to " + st.Lanes[1].Branch + " at " + shortSHA(twoHead),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("the record carries %q:\n%s", want, text)
		}
	}
	if !strings.Contains(text, "ends early on lane-2's rate-limit response") {
		t.Fatalf("the pause names the response's lane:\n%s", text)
	}

	// A second response inside the pause, from the lane still in flight,
	// frees its slot and is recorded, but next_eligible_at is written once.
	second := f.now.Add(5 * time.Minute)
	if _, err := rateLimited(f.repo.Root(), f.runID, "lane-1", f.await(t, "lane-1", RoleImplementer),
		&runner.Failure{Runner: runner.Claude, Reason: runner.ReasonRateLimited, Detail: "again"}, Options{Now: func() time.Time { return second }}); err != nil {
		t.Fatal(err)
	}
	st = f.state(t)
	pauses, limits := 0, 0
	for _, e := range st.Record {
		switch e.Stage {
		case "pause":
			pauses++
		case StageRateLimit:
			limits++
		}
	}
	if !st.NextEligibleAt.Equal(until) || pauses != 1 || limits != 2 || st.SlotsInUse() != 0 {
		t.Fatalf("next_eligible_at is written once for the run: %v, %d pause(s), %d response(s), %d slot(s):\n%s",
			st.NextEligibleAt, pauses, limits, st.SlotsInUse(), recordText(st))
	}

	// Before next_eligible_at nothing moves and the state is unchanged.
	before := stateBytes(t, f.repo.Root(), f.runID)
	f.now = until.Add(-time.Minute)
	_, err = advance(f.repo.Root(), f.runID, f.stages, f.opts())
	if r := mustRefusal(t, err); r.Stage != "pause" || !strings.Contains(r.Reason, until.Format(time.RFC3339)) {
		t.Fatalf("a step inside the pause is refused naming the time: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("a refused step changes nothing")
	}

	// At next_eligible_at a window opens and each lane's work is handed out
	// afresh from its branch, where it was checkpointed.
	f.now = until
	f.stepUntil(t, "both implementers are out again", func(st State) bool {
		return len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	if st := f.state(t); st.NextEligibleAt != nil || st.Lanes[0].Branch == "" || st.Lanes[1].Branch == "" {
		t.Fatalf("the new window runs on the lanes' own branches: %+v", st)
	}
}
