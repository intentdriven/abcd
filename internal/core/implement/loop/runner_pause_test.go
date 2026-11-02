package loop

// runner_pause_test.go proves iss-2611012047534449, following
// dcn-2611012113432054: a runner that fails to start a role pauses the lane
// and says why, never handing the role to the host; each later step retries
// the paused role once on the runner; a --route retries it at once; and a role
// already with the host, with nothing received, is retried the same way.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// launches counts the runs the named fake harness was started for.
func (e driveEnv) launches(t *testing.T, harness string) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(e.log, harness+".launches"))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(raw), "launched\n")
}

// recordSays reports whether a line of the run's record at stage (any stage
// when "") carries every one of want.
func recordSays(t *testing.T, root, id, stage string, want ...string) bool {
	t.Helper()
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range st.Record {
		if stage != "" && e.Stage != stage {
			continue
		}
		all := true
		for _, w := range want {
			all = all && strings.Contains(e.Note, w)
		}
		if all {
			return true
		}
	}
	return false
}

// pauseOnClaude drives the implement stage to the claude runner, which fails
// to start it as an unkeyed `claude --bare` does.
func pauseOnClaude(t *testing.T) (root, id string, steps Stages, o Options, env driveEnv, res StepResult) {
	t.Helper()
	root, id, steps, o = startToImplement(t)
	env = newDriveEnv(t, "notloggedin", "claude")
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	return root, id, steps, o, env, res
}

// TestARunnerThatFailsToStartPausesTheLaneAndSaysWhy: the role is not handed
// to the host, the step says the lane is paused, and the reason carries the
// runner's first stderr line rather than the exit status alone.
func TestARunnerThatFailsToStartPausesTheLaneAndSaysWhy(t *testing.T) {
	root, id, _, _, env, res := pauseOnClaude(t)
	if env.launches(t, "claude") != 1 {
		t.Fatalf("the runner is started once: %d", env.launches(t, "claude"))
	}
	if res.Awaiting != nil {
		t.Fatalf("a paused role is handed to no one, the host included: %+v", res.Awaiting)
	}
	if res.Fallback != nil && res.Fallback.Ran == runner.Host {
		t.Fatalf("the host does not run a role its runner failed to start: %+v", res.Fallback)
	}
	if !strings.Contains(res.Next, "paused") || !strings.Contains(res.Next, "Not logged in") {
		t.Fatalf("the step says the lane is paused, and why: %s", res.Next)
	}
	if strings.Contains(res.Next, "a second line") {
		t.Fatalf("the pause carries stderr's first line alone: %s", res.Next)
	}
	if !recordSays(t, root, id, "", "paused", "Not logged in", "exited with status 1") {
		t.Fatal("the run record names the pause with the runner's first stderr line")
	}
}

// TestEachLaterStepRetriesThePausedRoleOnceOnTheRunner: a later step starts
// the runner again for the paused role, once, and the run record names the
// retry; once the runner starts, its receipt completes the stage.
func TestEachLaterStepRetriesThePausedRoleOnceOnTheRunner(t *testing.T) {
	root, id, steps, o, env, _ := pauseOnClaude(t)
	cfg := runnerConfig(t, claudeImplementer, "")
	if _, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}}); err != nil {
		t.Fatal(err)
	}
	if n := env.launches(t, "claude"); n != 2 {
		t.Fatalf("the next step retries the paused role once on the runner: %d launches", n)
	}
	if !recordSays(t, root, id, StageRetry, "lane-1", RoleImplementer, runner.Claude) {
		t.Fatal("the run record names the retry")
	}
	t.Setenv(loopFakeEnv, "ok")
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if n := env.launches(t, "claude"); n != 3 {
		t.Fatalf("each step retries once: %d launches", n)
	}
	if res.PerformedStage != StageImplement || res.Route == nil || res.Route.Ran != runner.Claude {
		t.Fatalf("the retried runner's receipt completes the stage: %+v route %+v", res, res.Route)
	}
}

// TestARoleWithTheHostSinceBeforeTheRouteIsRetriedOnTheRunner: a lane handed
// to the host before any route existed, with nothing received, moves once the
// role is routed: the next step starts the runner for it. One whose receipt
// the host has written is the host's to hand back, and is not retried.
func TestARoleWithTheHostSinceBeforeTheRouteIsRetriedOnTheRunner(t *testing.T) {
	t.Run("nothing received", func(t *testing.T) {
		root, id, steps, o := startToImplement(t)
		if res, err := advance(root, id, steps, o); err != nil || res.Awaiting == nil {
			t.Fatalf("the host is handed the implementer: %+v %v", res, err)
		}
		env := newDriveEnv(t, "ok", "claude")
		res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
		if err != nil {
			t.Fatal(err)
		}
		if env.launches(t, "claude") != 1 || res.PerformedStage != StageImplement {
			t.Fatalf("the routed runner takes the host's role: %d launches, %+v", env.launches(t, "claude"), res)
		}
		if !recordSays(t, root, id, StageRetry, "with the host") {
			t.Fatal("the run record names the retry and why")
		}
	})
	t.Run("a receipt written", func(t *testing.T) {
		root, id, steps, o := startToImplement(t)
		res, err := advance(root, id, steps, o)
		if err != nil || res.Awaiting == nil {
			t.Fatalf("the host is handed the implementer: %+v %v", res, err)
		}
		if err := os.WriteFile(res.Awaiting.Receipt, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		env := newDriveEnv(t, "ok", "claude")
		if _, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}); err != nil {
			t.Fatal(err)
		}
		if n := env.launches(t, "claude"); n != 0 {
			t.Fatalf("a role whose receipt the host wrote is not retried: %d launches", n)
		}
	})
}

// TestARouteRetriesAPausedRoleAtOnce: a --route for the paused role is not
// refused as one the step will not use; the step retries the role on it.
func TestARouteRetriesAPausedRoleAtOnce(t *testing.T) {
	root, id, steps, o, env, _ := pauseOnClaude(t)
	t.Setenv(loopFakeEnv, "ok")
	cfg := runnerConfig(t, claudeImplementer, "")
	text := "implementer=runner:claude"
	routes, err := oracle.ParseRunnerRoutes([]string{text}, []string{RoleImplementer})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Override(routes); err != nil {
		t.Fatal(err)
	}
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatalf("a --route for the paused role retries it: %v", err)
	}
	if env.launches(t, "claude") != 2 || res.PerformedStage != StageImplement || res.Route == nil || res.Route.Override != text {
		t.Fatalf("the --route ran the paused role at once: %d launches, %+v route %+v", env.launches(t, "claude"), res, res.Route)
	}
}

// TestARoleARunnerHoldsIsNotStartedAgain: a step started a runner for the
// role, and the claim it took under the lock keeps any other step from
// starting a second one; a claim whose bound has passed lapses, and the role
// is retried.
func TestARoleARunnerHoldsIsNotStartedAgain(t *testing.T) {
	root, id, steps, o, env, _ := pauseOnClaude(t)
	now := o.now()
	claim := func(until time.Time) {
		t.Helper()
		if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
			a := &st.Lanes[0].Awaits[0]
			a.Paused, a.Running = nil, &RunnerClaim{Runner: runner.Claude, Session: "peer-session", Until: until}
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	claim(now.Add(time.Hour))
	cfg := runnerConfig(t, claudeImplementer, "")
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if env.launches(t, "claude") != 1 || res.Retried != nil {
		t.Fatalf("a role a runner holds is not started again: %d launches, %+v", env.launches(t, "claude"), res.Retried)
	}
	claim(now.Add(-time.Minute))
	t.Setenv(loopFakeEnv, "ok")
	res, err = Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if env.launches(t, "claude") != 2 || res.PerformedStage != StageImplement {
		t.Fatalf("a lapsed claim is retried: %d launches, %+v", env.launches(t, "claude"), res)
	}
	if !recordSays(t, root, id, StageRetry, "lapsed") {
		t.Fatal("the record names the lapsed runner as why")
	}
}

// TestAPausedStateIsVersion18s: an await's `paused` is state version 18's,
// so a version-17 file carrying one is refused naming it, and the pause
// round-trips through the state as written.
func TestAPausedStateIsVersion18s(t *testing.T) {
	root, id, _, _, _, _ := pauseOnClaude(t)
	if SchemaVersion != 18 || schemaVersionUnpaused != 17 {
		t.Fatalf("the pause is state version 18's: %d %d", SchemaVersion, schemaVersionUnpaused)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	p := st.Lanes[0].Awaits[0].Paused
	if p == nil || p.Runner != runner.Claude || p.Reason != runner.ReasonFailed || p.Tries != 1 || !strings.HasPrefix(p.Stderr, "Not logged in") {
		t.Fatalf("the await carries the pause: %+v", p)
	}
	if st.Lanes[0].Awaits[0].Running != nil {
		t.Fatalf("a paused await holds no runner's claim: %+v", st.Lanes[0].Awaits[0].Running)
	}
	if n := len(st.Fallbacks); n != 1 || st.Fallbacks[0].Ran != ranNone {
		t.Fatalf("one fallback receipt naming none as the route that ran: %+v", st.Fallbacks)
	}
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	old := strings.Replace(string(stateBytes(t, root, id)), fmt.Sprintf(`"schema_version": %d,`, SchemaVersion), `"schema_version": 17,`, 1)
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(root, id); err == nil || !strings.Contains(err.Error(), "`paused`") {
		t.Fatalf("a version-17 file carrying a pause is refused naming it: %v", err)
	}
}

// TestARestartWithARouteInForceStartsTheRunner: a restart no longer re-tells
// the role to the host whatever the route: with the implementer routed, the
// fresh agent is started on the runner at once, and a runner that does not
// start it pauses the lane.
func TestARestartWithARouteInForceStartsTheRunner(t *testing.T) {
	repo, runID, _, dir, _ := deadLane(t)
	env := newDriveEnv(t, "notloggedin", "claude")
	cfg := runnerConfig(t, claudeImplementer, "")
	res, err := RestartDrive(context.Background(), repo.Root(), runID, "lane-1", "", DefaultStages(), Options{}, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if env.launches(t, "claude") != 1 {
		t.Fatalf("the restart consults the runner: %d launches", env.launches(t, "claude"))
	}
	asideOf(t, dir)
	if res.Awaiting != nil || res.Paused == nil || res.Paused.Paused == nil || !strings.Contains(res.Next, "Not logged in") {
		t.Fatalf("the runner that did not start it pauses the lane: %+v", res.StepResult)
	}
	if !recordSays(t, repo.Root(), runID, StageRetry, "restarted") {
		t.Fatal("the run record names the retry the restart made")
	}

	// With the route on the host, the restart re-tells it as before.
	if _, err := RestartDrive(context.Background(), repo.Root(), runID, "lane-1", "", DefaultStages(), Options{},
		Runners{Config: runnerConfig(t, "", ""), Transcripts: &memTranscripts{}}); err != nil {
		t.Fatal(err)
	}
	if env.launches(t, "claude") != 1 {
		t.Fatalf("a restart on the host starts no runner: %d launches", env.launches(t, "claude"))
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if a := st.Lanes[0].Awaits[0]; a.Paused != nil || a.Running != nil {
		t.Fatalf("a restart clears the pause and the claim: %+v", a)
	}
}

// TestARestartsRouteForARoleItDoesNotReTellIsRefused: a --route with a
// restart routes the role the restart re-tells; any other is refused before
// anything is saved aside.
func TestARestartsRouteForARoleItDoesNotReTellIsRefused(t *testing.T) {
	repo, runID, _, dir, _ := deadLane(t)
	newDriveEnv(t, "ok", "claude")
	cfg := runnerConfig(t, `{"runner":{"claude":{}}}`, "")
	routes, err := oracle.ParseRunnerRoutes([]string{"security-reviewer=runner:claude"}, []string{RoleSecurity})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Override(routes); err != nil {
		t.Fatal(err)
	}
	_, err = RestartDrive(context.Background(), repo.Root(), runID, "lane-1", "", DefaultStages(), Options{}, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if r, ok := AsRefusal(err); !ok || r.Stage != StageRoute || !strings.Contains(r.Reason, "re-tells no security-reviewer") {
		t.Fatalf("the --route is refused: %v", err)
	}
	noAside(t, dir)
}

// TestTheStderrLineIsRedactedAndBounded: the line a pause keeps is the first
// after the transcript's stderr marker, one line, with a token sealed and the
// home directory never spelled out, and bounded.
func TestTheStderrLineIsRedactedAndBounded(t *testing.T) {
	raw := []byte(`{"type":"system"}` + "\n" + stderrMarker + "\n\n  Not logged in · Please run /login  \nsecond\n")
	if got := firstStderrLine(raw); got != "Not logged in · Please run /login" {
		t.Fatalf("first stderr line = %q", got)
	}
	if got := firstStderrLine([]byte(`{"type":"result"}` + "\n")); got != "" {
		t.Fatalf("a transcript with no stderr has no line: %q", got)
	}
	root := t.TempDir()
	token := "ghp_" + strings.Repeat("A1b2C3d4E5", 4)[:36]
	got := redactStderr(root, "Invalid token "+token)
	if strings.Contains(got, token) {
		t.Fatalf("a token in the stderr line is never kept: %q", got)
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if got := redactStderr(root, "cannot read "+filepath.Join(home, "x.json")); strings.Contains(got, home) {
			t.Fatalf("the home directory is never kept: %q", got)
		}
	}
	if got := redactStderr(root, strings.Repeat("é", 300)); len(got) > maxStderrLine+3 || !strings.HasSuffix(got, "...") || !utf8.ValidString(got) {
		t.Fatalf("the line is bounded on a rune: %d %q", len(got), got)
	}
}
