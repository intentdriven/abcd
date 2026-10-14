package loop

// ratelimit_test.go is criterion 8 of itd-2609201925079472 (spc-2609301921521360,
// "The rate-limit checkpoint"): a runner's rate-limit response ends the
// window early for the whole run, since every lane spends the same budget;
// every lane with work in flight is checkpointed to its own branch,
// next_eligible_at is written once, and the record names the lane the
// response came from.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// rateLimitPause is the pause these runs are started with. The bundled pause
// is zero, and a zero pause would let the step after a rate limit start
// straight away, so the window's early end is pinned to a pause of its own.
const rateLimitPause = 300

// claudeImplementer routes the implementer alone to the claude runner.
const claudeImplementer = `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`

// rateLimitedPair is a run of two lanes side by side whose second
// implementer, started through the claude runner, meets a rate limit
// mid-work: lane-1's implementer, on the host, is out with a commit made and
// an edit left uncommitted. It returns the fixture, the driven step's result
// and lane-1's commit.
func rateLimitedPair(t *testing.T) (*parFixture, StepResult, string) {
	t.Helper()
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{Pace: strp(fmt.Sprintf("%d/%d", BundledWorkMinutes, rateLimitPause)), SubAgents: strp("3")})
	f.stepUntil(t, "lane-1's implementer is out and lane-2 is at its implement stage", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && st.Lanes[1].Stage == StageImplement && len(st.Lanes[1].Awaits) == 0
	})
	l1 := f.lane(t, "lane-1")
	head1 := laneCommit(t, f.repo, l1, "one.txt")
	if err := os.WriteFile(filepath.Join(l1.Worktree, "wip.txt"), []byte("lane-1 at work\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.Symlink(self, filepath.Join(bin, "claude")); err != nil {
		t.Fatal(err)
	}
	// The fake claude comes first on PATH, so no real harness is reached;
	// git and the forge stub stay where the fixture put them.
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(loopFakeEnv, "ratelimit")
	t.Setenv(loopFakeLogEnv, "")
	res, err := Drive(context.Background(), f.repo.Root(), f.runID, f.stages, f.opts(),
		Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatalf("a rate-limit response is the step's answer, not its refusal: %v", err)
	}
	return f, res, head1
}

// entries are the run record's notes at stage, lane by lane.
func entries(st State, stage string) map[string][]string {
	out := map[string][]string{}
	for _, e := range st.Record {
		if e.Stage == stage {
			out[e.Lane] = append(out[e.Lane], e.Note)
		}
	}
	return out
}

func TestARateLimitMidLaneCheckpointsEveryLaneAndEndsTheWindow(t *testing.T) {
	f, res, head1 := rateLimitedPair(t)
	until := f.now.Add(rateLimitPause * time.Minute)

	// The window ends early, for the whole run, with the response named.
	if res.NextEligibleAt == nil || !res.NextEligibleAt.Equal(until) {
		t.Fatalf("next_eligible_at = %v, want %v", res.NextEligibleAt, until)
	}
	if rl := res.RateLimit; rl == nil || rl.Lane != "lane-2" || rl.Role != RoleImplementer || rl.Runner != runner.Claude ||
		!strings.Contains(rl.Response, "rejected rate_limit_event") {
		t.Fatalf("the step names the rate limit: %+v", res.RateLimit)
	}
	st := f.state(t)
	if st.NextEligibleAt == nil || !st.NextEligibleAt.Equal(until) {
		t.Fatalf("the state's next_eligible_at = %v, want %v", st.NextEligibleAt, until)
	}
	limits := entries(st, StageRateLimit)
	if len(limits["lane-2"]) != 1 || !strings.Contains(limits["lane-2"][0], "claude") || !strings.Contains(limits["lane-2"][0], "rejected rate_limit_event") {
		t.Fatalf("the record names the response and the lane it came from: %+v", limits)
	}
	if p := entries(st, "pause")[""]; len(p) != 1 || !strings.Contains(p[0], until.Format(time.RFC3339)) {
		t.Fatalf("the record names the pause once: %+v", p)
	}

	// The lane the response came from is checkpointed to its branch: its
	// agent's uncommitted work and partial receipt are saved aside, its
	// worktree is back at its last commit, and its implementer is out no more.
	l2 := f.lane(t, "lane-2")
	head2 := strings.TrimSpace(f.repo.Git("-C", l2.Worktree, "rev-parse", "HEAD"))
	if len(l2.Awaits) != 0 {
		t.Fatalf("the rate-limited agent's slot is freed: %+v", l2.Awaits)
	}
	if _, err := os.Stat(filepath.Join(l2.Worktree, "halfway.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lane-2's worktree is reset to its last commit (%v)", err)
	}
	if s := f.repo.Git("-C", l2.Worktree, "status", "--porcelain"); s != "" {
		t.Fatalf("lane-2's worktree is clean at its branch: %q", s)
	}
	dir := f.abs(RunRelDir + "/" + f.runID + "/lane-2")
	at, meta := asideOf(t, dir)
	if meta["head"] != head2 || !strings.Contains(meta["why"].(string), "rate limit") {
		t.Fatalf("aside.json names the head and the rate limit: %v", meta)
	}
	if patch, err := os.ReadFile(filepath.Join(at, AsidePatchName)); err != nil || !bytes.Contains(patch, []byte("half done")) {
		t.Fatalf("the uncommitted work is saved aside: %v", err)
	}
	cps := entries(st, StageCheckpoint)
	if len(cps["lane-2"]) != 1 || !strings.Contains(cps["lane-2"][0], shortSHA(head2)) || !strings.Contains(cps["lane-2"][0], at[len(f.repo.Root())+1:]) {
		t.Fatalf("the record names lane-2's checkpoint, its head and its aside: %+v", cps)
	}

	// Every other lane with work in flight is checkpointed at its branch's
	// head: its agent is not stopped and its worktree is not touched.
	l1 := f.lane(t, "lane-1")
	if len(l1.Awaits) != 1 || l1.Awaits[0].Role != RoleImplementer {
		t.Fatalf("lane-1's implementer is still out: %+v", l1.Awaits)
	}
	if _, err := os.Stat(filepath.Join(l1.Worktree, "wip.txt")); err != nil {
		t.Fatalf("lane-1's worktree is untouched: %v", err)
	}
	if len(cps["lane-1"]) != 1 || !strings.Contains(cps["lane-1"][0], shortSHA(head1)) || !strings.Contains(cps["lane-1"][0], RoleImplementer) {
		t.Fatalf("the record names lane-1's checkpoint at its head %s: %+v", shortSHA(head1), cps)
	}

	// Inside the pause a step refuses naming the time and changes nothing,
	// while an agent already out may still hand back its receipt.
	before := stateBytes(t, f.repo.Root(), f.runID)
	_, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
	if r, ok := AsRefusal(err); !ok || r.Stage != "pause" || !strings.Contains(r.Reason, until.Format(time.RFC3339)) {
		t.Fatalf("a step inside the pause refuses naming the time: %v", err)
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("a refused step inside the pause changes no state")
	}
	if err := os.Remove(filepath.Join(l1.Worktree, "wip.txt")); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(10 * time.Minute)
	f.receipt(t, "lane-1", f.await(t, "lane-1", RoleImplementer), head1)

	// After the pause, a fresh implementer takes lane-2's brief again, from
	// the checkpoint.
	f.now = until
	f.stepUntil(t, "lane-2's implementer is handed out again", func(st State) bool {
		for _, l := range st.Lanes {
			if l.ID == "lane-2" && l.awaitsRole(RoleImplementer) {
				return true
			}
		}
		return false
	})
	if a := f.await(t, "lane-2", RoleImplementer); a.Brief != l2.Brief {
		t.Fatalf("the fresh implementer takes the same brief: %+v", a)
	}
}

// A second lane's rate-limit response inside the pause checkpoints that lane
// too, and next_eligible_at stays as it was written.
func TestASecondRateLimitInsideThePauseKeepsNextEligibleAt(t *testing.T) {
	f, res, head1 := rateLimitedPair(t)
	until := *res.NextEligibleAt
	l1 := f.lane(t, "lane-1")
	f.now = f.now.Add(20 * time.Minute)
	again, err := rateLimitWindow(f.repo.Root(), f.runID, "lane-1", l1.Awaits[0],
		&runner.Failure{Runner: runner.Claude, Reason: runner.ReasonRateLimited, Detail: "its event stream reports a rejected rate_limit_event and the run did not finish"}, f.opts())
	if err != nil {
		t.Fatal(err)
	}
	st := f.state(t)
	if again.NextEligibleAt == nil || !again.NextEligibleAt.Equal(until) || !st.NextEligibleAt.Equal(until) {
		t.Fatalf("next_eligible_at is written once: %v, state %v, want %v", again.NextEligibleAt, st.NextEligibleAt, until)
	}
	if p := entries(st, "pause")[""]; len(p) != 1 {
		t.Fatalf("one pause is recorded: %+v", p)
	}
	limits := entries(st, StageRateLimit)
	if len(limits["lane-1"]) != 1 || len(limits["lane-2"]) != 1 {
		t.Fatalf("the record names each response with its lane: %+v", limits)
	}
	l1 = f.lane(t, "lane-1")
	if len(l1.Awaits) != 0 {
		t.Fatalf("lane-1's rate-limited agent's slot is freed: %+v", l1.Awaits)
	}
	if _, err := os.Stat(filepath.Join(l1.Worktree, "wip.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lane-1's uncommitted work is saved aside and its worktree reset (%v)", err)
	}
	if got := strings.TrimSpace(f.repo.Git("-C", l1.Worktree, "rev-parse", "HEAD")); got != head1 {
		t.Fatalf("lane-1 keeps its committed work: HEAD %s, want %s", got, head1)
	}
}

// A validator that meets the rate limit edits nothing, and may share the
// worktree with its round: only its partial return is saved aside, the
// worktree is left as it is, and the first step after the pause hands its
// review to a fresh validator.
func TestARateLimitedValidatorLeavesTheWorktreeAndIsHandedOutAgain(t *testing.T) {
	f := newParFixture(t, "1. One\n", Options{})
	f.stepUntil(t, "the implementer is out", func(st State) bool { return len(st.Lanes) == 1 && len(st.Lanes[0].Awaits) == 1 })
	f.implement(t, "lane-1", "one.txt")
	f.stepUntil(t, "the ruthless reviewer is out", func(st State) bool { return st.Lanes[0].awaitsRole(RoleRuthless) })
	a := f.await(t, "lane-1", RoleRuthless)
	if err := os.WriteFile(f.abs(a.Receipt), []byte("# Review\n\nhalf"), 0o600); err != nil {
		t.Fatal(err)
	}
	l := f.lane(t, "lane-1")
	if err := os.WriteFile(filepath.Join(l.Worktree, "note.txt"), []byte("left by no one in particular\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := rateLimitWindow(f.repo.Root(), f.runID, "lane-1", a,
		&runner.Failure{Runner: runner.Claude, Reason: runner.ReasonRateLimited, Detail: "its event stream reports a rejected rate_limit_event and the run did not finish"}, f.opts())
	if err != nil {
		t.Fatal(err)
	}
	if res.NextEligibleAt == nil || res.RateLimit == nil || res.RateLimit.Role != RoleRuthless {
		t.Fatalf("the validator's rate limit ends the window: %+v", res)
	}
	if f.lane(t, "lane-1").awaitsRole(RoleRuthless) {
		t.Fatal("the rate-limited validator's slot is freed")
	}
	if _, err := os.Stat(f.abs(a.Receipt)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the partial return leaves its path (%v)", err)
	}
	at, _ := asideOf(t, f.abs(RunRelDir+"/"+f.runID+"/lane-1"))
	if got, err := os.ReadFile(filepath.Join(at, filepath.Base(a.Receipt))); err != nil || string(got) != "# Review\n\nhalf" {
		t.Fatalf("the partial return is saved aside: %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, "note.txt")); err != nil {
		t.Fatalf("a validator's rate limit leaves the worktree as it is: %v", err)
	}
	f.now = *res.NextEligibleAt
	f.stepUntil(t, "the ruthless reviewer is handed out again", func(st State) bool { return st.Lanes[0].awaitsRole(RoleRuthless) })
}

// A rate-limit response read after a restart re-told the await it came from
// meets an await the gone agent no longer holds: the receipt path is the same,
// but its Since is the restart's, so the fresh agent's work is not saved aside
// and its worktree is not reset (iss-2610100913088696).
func TestARateLimitAfterARestartLeavesTheReToldAwait(t *testing.T) {
	f := newParFixture(t, "1. One\n", Options{})
	f.stepUntil(t, "the implementer is out", func(st State) bool { return len(st.Lanes) == 1 && st.Lanes[0].awaitsRole(RoleImplementer) })
	gone := f.await(t, "lane-1", RoleImplementer)
	f.now = f.now.Add(time.Minute)
	if _, err := Restart(f.repo.Root(), f.runID, "lane-1", "", f.opts()); err != nil {
		t.Fatal(err)
	}
	fresh := f.await(t, "lane-1", RoleImplementer)
	if fresh.Receipt != gone.Receipt || fresh.Since.Equal(gone.Since) {
		t.Fatalf("the restart re-tells the await at the same path with a new Since: %+v, was %+v", fresh, gone)
	}
	l := f.lane(t, "lane-1")
	if err := os.WriteFile(filepath.Join(l.Worktree, "fresh.txt"), []byte("the fresh agent at work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := rateLimitWindow(f.repo.Root(), f.runID, "lane-1", gone,
		&runner.Failure{Runner: runner.Claude, Reason: runner.ReasonRateLimited, Detail: "its event stream reports a rejected rate_limit_event and the run did not finish"}, f.opts()); err != nil {
		t.Fatal(err)
	}
	if got := f.await(t, "lane-1", RoleImplementer); !got.Since.Equal(fresh.Since) {
		t.Fatalf("the re-told await stays out: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, "fresh.txt")); err != nil {
		t.Fatalf("the fresh agent's worktree is not reset: %v", err)
	}
	asideOf(t, f.abs(RunRelDir+"/"+f.runID+"/lane-1"))
}

// keptValidator is a lane whose round has its ruthless and security reviewers
// out, the ruthless one met by a rate limit whose partial return cannot be
// saved aside (a symlink stands at its return's path): its await is kept,
// marked, and the window ends. The security reviewer is out and live, with a
// partial return of its own. It returns the fixture, the two awaits as they
// were before the rate limit, and the end of the pause.
func keptValidator(t *testing.T) (*parFixture, Await, Await, time.Time) {
	t.Helper()
	f := newParFixture(t, "1. One\n", Options{Pace: strp(fmt.Sprintf("%d/%d", BundledWorkMinutes, rateLimitPause))})
	f.stepUntil(t, "the implementer is out", func(st State) bool { return len(st.Lanes) == 1 && len(st.Lanes[0].Awaits) == 1 })
	f.implement(t, "lane-1", "one.txt")
	f.stepUntil(t, "both reviewers are out", func(st State) bool {
		return st.Lanes[0].awaitsRole(RoleRuthless) && st.Lanes[0].awaitsRole(RoleSecurity)
	})
	limited, live := f.await(t, "lane-1", RoleRuthless), f.await(t, "lane-1", RoleSecurity)
	if err := os.Symlink(filepath.Join(t.TempDir(), "elsewhere.md"), f.abs(limited.Receipt)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.abs(live.Receipt), []byte("# Review\n\nunder way"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := rateLimitWindow(f.repo.Root(), f.runID, "lane-1", limited,
		&runner.Failure{Runner: runner.Claude, Reason: runner.ReasonRateLimited, Detail: "its event stream reports a rejected rate_limit_event and the run did not finish"}, f.opts())
	if err != nil {
		t.Fatal(err)
	}
	if res.NextEligibleAt == nil || len(res.RateLimit.Checkpoints) != 1 || res.RateLimit.Checkpoints[0].Kept == "" {
		t.Fatalf("the refused save aside keeps the await: %+v", res.RateLimit)
	}
	if kept := f.await(t, "lane-1", RoleRuthless); kept.Kept == "" || !kept.Since.Equal(limited.Since) {
		t.Fatalf("the kept await is marked in the run state: %+v", kept)
	}
	return f, limited, live, *res.NextEligibleAt
}

// A validator whose partial return could not be saved aside at a rate limit
// is kept and marked; once the person settles what refused the save,
// `--restart <lane>` re-tells exactly that await from a fresh Since, as the
// checkpoint's record says it will, and the lane goes on to pass its round
// (iss-2610100913086397).
func TestARestartReTellsAValidatorARateLimitKept(t *testing.T) {
	f, limited, _, until := keptValidator(t)
	f.now = until
	if err := os.Remove(f.abs(limited.Receipt)); err != nil {
		t.Fatal(err)
	}
	res, err := Restart(f.repo.Root(), f.runID, "lane-1", "", f.opts())
	if err != nil {
		t.Fatalf("a restart takes the kept validator: %v", err)
	}
	fresh := f.await(t, "lane-1", RoleRuthless)
	if fresh.Kept != "" || !fresh.Since.Equal(until) || fresh.Receipt != limited.Receipt || fresh.Brief != limited.Brief {
		t.Fatalf("the kept await is re-told, unmarked, from the restart: %+v", fresh)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleRuthless || !strings.Contains(res.Next, RoleRuthless) {
		t.Fatalf("the restart names the fresh ruthless reviewer to start: %+v", res.StepResult)
	}
	if got := entries(f.state(t), stageRestart)["lane-1"]; len(got) != 1 || !strings.Contains(got[0], RoleRuthless) {
		t.Fatalf("the record names the restarted reviewer: %+v", got)
	}
	f.roundPassed(t, "lane-1")
	if l := f.lane(t, "lane-1"); l.Stage == StageValidate || len(l.Awaits) != 0 {
		t.Fatalf("the lane resumes and passes its round: %+v", l)
	}
}

// The restart re-tells only the await the rate limit marked: a validator of
// the same round that is out and live keeps its await, its Since and its
// partial return.
func TestARestartLeavesALiveValidatorOfTheSameRound(t *testing.T) {
	f, limited, live, until := keptValidator(t)
	f.now = until
	if err := os.Remove(f.abs(limited.Receipt)); err != nil {
		t.Fatal(err)
	}
	if _, err := Restart(f.repo.Root(), f.runID, "lane-1", "", f.opts()); err != nil {
		t.Fatal(err)
	}
	if got := f.await(t, "lane-1", RoleSecurity); got != live {
		t.Fatalf("the live reviewer's await is untouched: %+v, was %+v", got, live)
	}
	if got, err := os.ReadFile(f.abs(live.Receipt)); err != nil || string(got) != "# Review\n\nunder way" {
		t.Fatalf("the live reviewer's partial return stays where it writes it: %q %v", got, err)
	}
}

// With two awaits kept on one lane, a restart proves every one before it saves
// anything: when the second's partial return cannot be read, the restart is
// refused with nothing changed, so the first's partial return is not saved
// aside or moved from where its agent writes it.
func TestARestartOfTwoKeptAwaitsProvesBothBeforeSavingEither(t *testing.T) {
	f, _, _, until := keptValidator(t)
	f.now = until
	f.rewrite(t, func(st *State) {
		for k := range st.Lanes[0].Awaits {
			st.Lanes[0].Awaits[k].Kept = "its partial return could not be saved aside"
		}
	})
	awaits := f.lane(t, "lane-1").Awaits
	if len(awaits) != 2 {
		t.Fatalf("the round has its two reviewers out: %+v", awaits)
	}
	first, second := awaits[0], awaits[1]
	for _, p := range []string{first.Receipt, second.Receipt} {
		if err := os.Remove(f.abs(p)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(f.abs(first.Receipt), []byte("# Review\n\nsettled"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "elsewhere.md"), f.abs(second.Receipt)); err != nil {
		t.Fatal(err)
	}
	before := f.state(t)
	_, err := Restart(f.repo.Root(), f.runID, "lane-1", "", f.opts())
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, `"`+second.Receipt+`" cannot be read`) {
		t.Fatalf("the refusal names the unreadable partial return: %+v", r)
	}
	noAside(t, f.abs(RunRelDir+"/"+f.runID+"/lane-1"))
	if got, err := os.ReadFile(f.abs(first.Receipt)); err != nil || string(got) != "# Review\n\nsettled" {
		t.Fatalf("the first kept await's partial return stays at its path: %q %v", got, err)
	}
	if got := f.lane(t, "lane-1").Awaits; !slices.Equal(got, before.Lanes[0].Awaits) {
		t.Fatalf("the awaits are unchanged: %+v, was %+v", got, before.Lanes[0].Awaits)
	}
}

// An await's `kept` is version 10's: version 9 is its strict subset, read as a
// run no rate limit kept an await of and written back at version 10, and a
// file of version 9 or lower carrying `kept` is not one its version wrote, and
// is refused naming the version and the key.
func TestAVersion9StateCarryingAKeptAwaitIsRefused(t *testing.T) {
	f, _, _, _ := keptValidator(t)
	path := f.abs(StateRelPath(f.runID))
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// as rewrites the file as a version with lane 0 edited by edit.
	as := func(version int, edit func(lane map[string]any)) {
		var doc map[string]any
		if err := json.Unmarshal(current, &doc); err != nil {
			t.Fatal(err)
		}
		doc["schema_version"] = version
		edit(doc["lanes"].([]any)[0].(map[string]any))
		out, _ := json.MarshalIndent(doc, "", "  ")
		if err := os.WriteFile(path, out, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	unkept := func(lane map[string]any) {
		for _, a := range lane["awaits"].([]any) {
			delete(a.(map[string]any), "kept")
		}
	}
	serial := func(lane map[string]any) {
		lane["awaiting"] = lane["awaits"].([]any)[0]
		delete(lane, "awaits")
	}
	kept := func(lane map[string]any) {
		lane["awaits"].([]any)[0].(map[string]any)["kept"] = "its partial return could not be saved aside"
	}

	as(9, unkept)
	if st, err := ReadState(f.repo.Root(), f.runID); err != nil || st.SchemaVersion != SchemaVersion {
		t.Fatalf("a version-9 file without `kept` reads as the current version: %+v %v", st.SchemaVersion, err)
	}
	for name, tc := range map[string]struct {
		version int
		edit    func(lane map[string]any)
	}{
		"v9 kept": {9, func(lane map[string]any) { unkept(lane); kept(lane) }},
		"v8 kept": {8, func(lane map[string]any) { unkept(lane); kept(lane); serial(lane) }},
	} {
		as(tc.version, tc.edit)
		_, err := ReadState(f.repo.Root(), f.runID)
		r := mustRefusal(t, err)
		if !strings.Contains(r.Reason, fmt.Sprintf("schema version %d", tc.version)) || !strings.Contains(r.Reason, "`kept`") {
			t.Errorf("%s: refused naming the version and the key: %+v", name, r)
		}
	}
}
