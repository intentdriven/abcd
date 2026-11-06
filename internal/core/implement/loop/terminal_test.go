package loop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The terminal loop (spc-2610230813348813 step 2), against a stand-in harness
// on PATH and a scratch ledger on a scratch remote.

const (
	terminalFakeMode  = "terminal"
	terminalScriptEnv = "ABCD_LOOP_FAKE_SCRIPT"
	terminalResetEnv  = "ABCD_LOOP_FAKE_RESET"
	terminalTamperEnv = "ABCD_LOOP_FAKE_TAMPER"
	thirdIssue        = "iss-2609230000000003"
	// terminalRoutes is every role a lane starts, on the claude runner.
	terminalRoutes = `{"roles":{"drain-judge":{"runner":"claude"},"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`
)

// terminalFakeTurn takes the next line of the launch's script, one behaviour
// per launch, and answers the fake harness's mode for it: shape (the default
// past the script's end) answers as the role's contract asks; noshape writes a
// receipt no verifier takes; late does not finish; none exits with no answer;
// limited meets the account's rate limit; dies is killed in its call; yes is a judge answering that the issue needs a
// decision.
func terminalFakeTurn(role string) string {
	path := os.Getenv(terminalScriptEnv)
	raw, _ := os.ReadFile(path)
	lines := strings.Fields(string(raw))
	count, _ := os.ReadFile(path + ".n")
	n, _ := strconv.Atoi(strings.TrimSpace(string(count)))
	// Atomic, since a hang limit can kill this process anywhere: a count truncated
	// and not yet written would read as zero, and send the next launch to the
	// script's first line (iss-2611061914290330). Killed before the rename, the next
	// launch repeats this line.
	_ = fsutil.WriteFileAtomic(path+".n", []byte(strconv.Itoa(n+1)), 0o600)
	turn := "shape"
	if n < len(lines) {
		turn = lines[n]
	}
	switch turn {
	case "noshape":
		return "noshape"
	case "late":
		return "hang"
	case "none":
		return "notloggedin"
	case "limited":
		return "ratelimit"
	case "crashes":
		return "startedcrash"
	case "spends":
		return "spentcrash"
	case "retries":
		return "retrycrash"
	case "apifails":
		return "apicrash"
	case "accountfails":
		return "accountzero"
	case "tamper":
		// A stage's line rewriting the driver's own run state.
		if p := os.Getenv(terminalTamperEnv); p != "" {
			if raw, err := os.ReadFile(p); err == nil {
				_ = os.WriteFile(p, []byte(strings.ReplaceAll(string(raw), `"stage": "implement"`, `"stage": "land"`)), 0o600)
			}
		}
		return "ok"
	case "unreadstate":
		// The run's state goes unreadable while the agent is in its call.
		if p := os.Getenv(terminalTamperEnv); p != "" {
			_ = os.Chmod(p, 0)
		}
		return "ok"
	case "dies":
		// The harness killed mid-call: no result, no receipt.
		_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
	case "yes":
		return "judge-" + JudgementYes
	case "slow":
		time.Sleep(time.Second)
	case "incall":
		// The agent in its call, as a check beside it could see.
		mark := os.Getenv(terminalScriptEnv) + ".in-call"
		_ = os.WriteFile(mark, nil, 0o600)
		time.Sleep(300 * time.Millisecond)
		_ = os.Remove(mark)
	}
	if strings.HasPrefix(role, RoleDrainJudge) {
		return "judge-" + JudgementNo
	}
	return "ok"
}

// TestACounterWriteCutOffKeepsTheScriptsPlace: the launches that read terminalFakeTurn's
// count can be killed at any moment of a hang limit, and a write cut off there must
// leave the count as it was. Back at zero, the next launch takes the script's first
// line instead of its own. A file-size limit of nothing cuts every write off after
// its open, the same place each time, so this needs no kill to land anywhere.
func TestACounterWriteCutOffKeepsTheScriptsPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script")
	if err := os.WriteFile(path, []byte("shape\nnone\nnone\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".n", []byte("2"), 0o600); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launch := exec.Command("/bin/sh", "-c", `ulimit -f 0; exec "$0" brief`, exe)
	launch.Env = append(os.Environ(), loopFakeEnv+"="+terminalFakeMode, terminalScriptEnv+"="+path, loopFakeLogEnv+"=")
	// It exits as the line it took says, and "none" prints this, which shows it got
	// past the count.
	out, _ := launch.CombinedOutput()
	if !strings.Contains(string(out), "Not logged in") {
		t.Fatalf("the launch never took its line: %q", out)
	}
	got, err := os.ReadFile(path + ".n")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "2" {
		t.Fatalf("the count is %q after a write cut off; want it left at 2", got)
	}
}

// terminalWorld is one test's scratch ledger of three eligible issues, on a
// scratch remote, with the stand-in harness on PATH.
type terminalWorld struct {
	repo  *gittest.Repo
	bare  string
	env   driveEnv
	clock *clock
	// sleeps are the waits the loop asked for, each moving the clock.
	sleeps []time.Duration
	lines  []string
	// onSleep is called at each wait, before the clock moves.
	onSleep func(time.Duration)
	// preflights are the worktrees the landing's preflight ran in.
	preflights []string
	preflight  error
	stop       chan struct{}
	stages     Stages
	// needsPreflight makes a land stage wait for its receipt until the
	// preflight has run.
	needsPreflight bool
	// noReceipt makes a land stage wait for its receipt however often the
	// preflight has run.
	noReceipt bool
}

func newTerminalWorld(t *testing.T, script ...string) *terminalWorld {
	t.Helper()
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	repo.Write("commands/other.md", "# other\n")
	fileIssueAt(t, repo, thirdIssue, capture.SeverityMinor, capture.Category("documentation"), "Name the other flag in the page.", "commands/other.md")
	repo.Commit("the third issue")
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	w := &terminalWorld{repo: repo, bare: bare, clock: &clock{t: time.Now().UTC()}, stop: make(chan struct{})}
	w.env = judgeEnv(t, terminalFakeMode)
	path := filepath.Join(w.env.log, "script")
	if err := os.WriteFile(path, []byte(strings.Join(script, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(terminalScriptEnv, path)
	w.stages = w.fakeStages(t)
	return w
}

// fakeStages are a lane's stages with an implementer the runner starts, whose
// verifier reads the receipt it wrote, and a landing that says what it did.
func (w *terminalWorld) fakeStages(t *testing.T) Stages {
	worktree := t.TempDir()
	var n atomic.Int64
	return Stages{
		{Name: StageWorktree, Piece: 6, Run: func(c Context, l *Lane) (Outcome, error) {
			l.Branch, l.Worktree, l.BaseSHA = "build/"+l.ID, worktree, driveBase
			return Outcome{Note: "worktree done"}, nil
		}},
		{Name: StageBrief, Piece: 5, Run: func(c Context, l *Lane) (Outcome, error) {
			l.Brief = RunRelDir + "/brief.md"
			return Outcome{Note: "brief done"}, nil
		}},
		{Name: StageImplement, Piece: 7,
			Run: func(c Context, l *Lane) (Outcome, error) {
				return Outcome{Await: &Await{Role: RoleImplementer, Brief: filepath.Join(c.RepoRoot, l.Brief),
					Receipt: filepath.Join(c.RepoRoot, RunRelDir, fmt.Sprintf("receipt-%d.json", n.Add(1)))}}, nil
			},
			Verify: func(c Context, l *Lane, receipt string) error {
				raw, err := os.ReadFile(receipt)
				if err != nil || strings.TrimSpace(string(raw)) != "{}" {
					_ = os.Remove(receipt)
					return refuse("receipt", "", l.ID, "no receipt the contract accepts at "+receipt, "write it, then hand it back")
				}
				l.Receipts = append(l.Receipts, ReceiptRecord{Role: RoleImplementer, Receipt: receipt})
				return nil
			}},
		{Name: StageValidate, Piece: 8, Run: func(Context, *Lane) (Outcome, error) { return Outcome{Note: "validate done"}, nil }},
		{Name: StageLand, Piece: 9, Run: func(c Context, l *Lane) (Outcome, error) {
			if w.noReceipt || w.needsPreflight && len(w.preflights) == 0 {
				r := contend(string(StageLand), "", l.ID, "no preflight receipt names the lane's head", "run the preflight")
				r.checkWait = c.Now
				return Outcome{}, r
			}
			return Outcome{Note: "land done"}, nil
		}},
	}
}

func (w *terminalWorld) options(t *testing.T) TerminalOptions {
	t.Helper()
	cfg := runnerConfig(t, terminalRoutes, "")
	tr := &memTranscripts{}
	return TerminalOptions{
		Options: Options{Now: w.clock.now, Runners: cfg},
		Drain:   DrainOptions{Max: 3, Transcripts: tr, JudgeTimeout: 20 * time.Second},
		Runners: Runners{Config: cfg, Transcripts: tr, Timeout: 20 * time.Second},
		Stages:  w.stages,
		Stop:    w.stop,
		Say:     func(s string) { w.lines = append(w.lines, s) },
		Preflight: func(_ context.Context, worktree string) error {
			w.preflights = append(w.preflights, worktree)
			return w.preflight
		},
		Sleep: func(_ context.Context, d time.Duration) {
			w.sleeps = append(w.sleeps, d)
			if w.onSleep != nil {
				w.onSleep(d)
			}
			w.clock.t = w.clock.t.Add(d)
		},
	}
}

func (w *terminalWorld) launches(t *testing.T) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(w.env.log, "claude.launches"))
	if err != nil {
		return 0
	}
	return strings.Count(string(raw), "\n")
}

func lanesDone(t *testing.T, repo *gittest.Repo) []string {
	t.Helper()
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	var done []string
	for _, l := range st.Lanes {
		if l.Outcome == DrainLaneDone {
			done = append(done, l.Issue)
		}
	}
	slices.Sort(done)
	return done
}

// TestTheTerminalDrivesThreeIssuesToTheirLandings: one call, no input, three
// judged issues, three lanes landed, every role through the runner and every
// receipt recorded as Drive records it; the run ends at its cap.
func TestTheTerminalDrivesThreeIssuesToTheirLandings(t *testing.T) {
	w := newTerminalWorld(t)
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap {
		t.Fatalf("the run ends at its cap: %+v", res)
	}
	want := []string{eligibleIssue, secondIssue, thirdIssue}
	slices.Sort(want)
	if got := lanesDone(t, w.repo); !slices.Equal(got, want) {
		t.Fatalf("three lanes landed: %v\n%s", got, strings.Join(w.lines, "\n"))
	}
	// A judge and an implementer for each.
	if got := w.launches(t); got != 6 {
		t.Fatalf("six agents ran, got %d", got)
	}
	st, _, _ := readDrain(w.repo.Root())
	for _, l := range st.Lanes {
		run, err := ReadState(w.repo.Root(), l.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if n := len(run.Lanes[0].Receipts); n != 1 || run.Lanes[0].Receipts[0].Route == nil || run.Lanes[0].Receipts[0].Route.Ran != runner.Claude {
			t.Fatalf("each receipt names the runner that ran it: %+v", run.Lanes[0].Receipts)
		}
	}
	if len(w.lines) < 6 {
		t.Fatalf("one line per step: %q", w.lines)
	}
	for _, id := range want {
		if !strings.Contains(strings.Join(w.lines, "\n"), id) {
			t.Errorf("the lines name %s: %q", id, w.lines)
		}
	}
}

// TestAnEmptyLedgerEndsTheRun: nothing eligible ends it, with no agent run.
func TestAnEmptyLedgerEndsTheRun(t *testing.T) {
	w := newTerminalWorld(t)
	o := w.options(t)
	o.Drain.Skip, o.Drain.SkipReason = []string{eligibleIssue, secondIssue, thirdIssue}, "not this run"
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedEmpty || w.launches(t) != 0 {
		t.Fatalf("nothing eligible ends the run: %+v, %d launches", res, w.launches(t))
	}
}

// TestARoleAnsweredBadlyPausesAndALaterStepRetriesIt is the stand-in out of
// shape, late, not at all and killed in its call: each pauses the lane, nothing is handed back,
// and the retry lands it.
func TestARoleAnsweredBadlyPausesAndALaterStepRetriesIt(t *testing.T) {
	for _, bad := range []string{"noshape", "late", "none", "dies"} {
		t.Run(bad, func(t *testing.T) {
			w := newTerminalWorld(t, "shape", bad)
			o := w.options(t)
			o.Drain.Max = 1
			if bad == "late" {
				o.Runners.Timeout = 5 * time.Second
			}
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ended != DrainStoppedCap {
				t.Fatalf("the lane lands on the retry: %+v\n%s", res, strings.Join(w.lines, "\n"))
			}
			if got := lanesDone(t, w.repo); len(got) != 1 {
				t.Fatalf("one lane landed: %v\n%s", got, strings.Join(w.lines, "\n"))
			}
			st, _, _ := readDrain(w.repo.Root())
			if len(st.HandBacks) != 0 {
				t.Fatalf("nothing is handed back: %+v", st.HandBacks)
			}
			if !slices.Contains(w.sleeps, TerminalWaitFloor) {
				t.Fatalf("the retry waits at the floor: %v", w.sleeps)
			}
		})
	}
}

// TestALandingWaitingForItsReceiptRunsThePreflightInTheLanesWorktree: the
// wait for a receipt is the loop's to end, and it then steps again.
func TestALandingWaitingForItsReceiptRunsThePreflightInTheLanesWorktree(t *testing.T) {
	w := newTerminalWorld(t)
	w.needsPreflight = true
	o := w.options(t)
	o.Drain.Max = 1
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap || len(w.preflights) != 1 || w.preflights[0] == "" {
		t.Fatalf("the preflight ran once in the lane's worktree: %+v %q", res, w.preflights)
	}
	if len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("the lane landed after its preflight:\n%s", strings.Join(w.lines, "\n"))
	}
}

// TestAStopSetFromOutsideIsHonouredAtTheNextBoundary: set before the first
// move, nothing moves and no agent runs.
func TestAStopSetFromOutsideIsHonouredAtTheNextBoundary(t *testing.T) {
	w := newTerminalWorld(t)
	close(w.stop)
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalStopped || w.launches(t) != 0 {
		t.Fatalf("a stop is honoured at once: %+v, %d launches", res, w.launches(t))
	}
}

// TestNoCheckRunsWhileAStageIsInItsCall: the check before a stage, where a proof may run,
// comes between calls, whatever the renewals beside a call do.
func TestNoCheckRunsWhileAStageIsInItsCall(t *testing.T) {
	w := newTerminalWorld(t, slices.Repeat([]string{"incall"}, 12)...)
	o := w.options(t)
	mark := os.Getenv(terminalScriptEnv) + ".in-call"
	was := terminalRenewTick
	terminalRenewTick = 20 * time.Millisecond
	t.Cleanup(func() { terminalRenewTick = was })
	calls, during := 0, 0
	o.BeforeStage = func() error {
		calls++
		if _, err := os.Stat(mark); err == nil {
			during++
		}
		return nil
	}
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	if calls == 0 || w.launches(t) <= 3 {
		t.Fatalf("the world ran no stage beside its judges: %d checks, %d launches", calls, w.launches(t))
	}
	if during != 0 {
		t.Fatalf("the check ran %d times during a call", during)
	}
}

// TestAWaitInTheCheckEndsTheRunOnItsOwnReason: what ended the wait is what the run reports,
// so a person's stop is not read as a wall; a wall that did not hold is still one.
func TestAWaitInTheCheckEndsTheRunOnItsOwnReason(t *testing.T) {
	for _, ended := range []string{TerminalStopped, TerminalLimited, TerminalOutage} {
		t.Run(ended, func(t *testing.T) {
			w := newTerminalWorld(t)
			o := w.options(t)
			o.BeforeStage = func() error { return &WallEnded{Ended: ended} }
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ended != ended || w.launches(t) != 0 {
				t.Fatalf("ended %+v after %d launches", res, w.launches(t))
			}
		})
	}
}

// TestASettingsDifferenceStopsTheRunBeforeTheNextStage: the check runs before
// each stage, and the first refusal ends the run with no further launch.
func TestASettingsDifferenceStopsTheRunBeforeTheNextStage(t *testing.T) {
	w := newTerminalWorld(t)
	o := w.options(t)
	calls := 0
	o.BeforeStage = func() error {
		calls++
		if w.launches(t) >= 1 {
			return errors.New("the walled directory's settings differ from what abcd wrote")
		}
		return nil
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWall || w.launches(t) != 1 {
		t.Fatalf("the run stops after the one launch: %+v, %d launches", res, w.launches(t))
	}
	if calls < 2 || !strings.Contains(strings.Join(w.lines, "\n"), "differ") {
		t.Fatalf("the check runs before each stage and says why: %d calls, %q", calls, w.lines)
	}
}

// TestALaneThatDoesNotMoveHasADeadline: a step that starts no agent and moves no
// stage, here a role with nowhere to run, is waited through on the outage's
// schedule rather than at the floor for ever (adr-2610222222326104 condition
// 3), and nothing is handed back.
func TestALaneThatDoesNotMoveHasADeadline(t *testing.T) {
	w := newTerminalWorld(t)
	for i := range w.stages {
		if w.stages[i].Name == StageImplement {
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				return Outcome{Await: &Await{Role: RoleAuditor, Brief: filepath.Join(c.RepoRoot, l.Brief),
					Receipt: filepath.Join(c.RepoRoot, RunRelDir, "receipt-held.json")}}, nil
			}
		}
	}
	o := w.options(t)
	o.Drain.Max = 1
	sleep := o.Sleep
	o.Sleep = func(ctx context.Context, d time.Duration) {
		if len(w.sleeps) > 100 {
			t.Fatalf("the lane waits with no deadline: %v", w.sleeps[:8])
		}
		sleep(ctx, d)
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalOutage || res.Run == "" {
		t.Fatalf("the run ends at the outage's limit, naming the lane: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 0 {
		t.Fatalf("nothing is handed back: %+v", st.HandBacks)
	}
}
