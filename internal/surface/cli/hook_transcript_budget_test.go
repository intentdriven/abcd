package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/history"
)

// The prompt hook's transcript step runs beside the rest of the hook, right
// after the root lookup, and finishes the transcript it begins until a bound
// counted from the hook's start, under the host's limit; the session-start
// drain does the same under its own hook's limit (iss-2611101302324425). The
// transcripts here are made up.

// budgetCheckout is an away checkout with one staged transcript.
func budgetCheckout(t *testing.T) (root, rootSHA string) {
	t.Helper()
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	root = awayCheckout(t)
	rootSHA = gitAt(t, root, "rev-list", "--max-parents=0", "HEAD")
	stageRaw(t, root, rootSHA, "sess-budget", `{"type":"assistant","text":"made-up work"}`+"\n")
	setAway(t)
	return root, rootSHA
}

// runPrompt runs the prompt hook with its own buffers.
func runPrompt(t *testing.T, root string, so io.Writer, se *bytes.Buffer) {
	t.Helper()
	cmd := NewRootCommand()
	cmd.SetOut(so)
	cmd.SetErr(se)
	cmd.SetIn(strings.NewReader(awayPromptPayload("sess-budget", root, "commit this change")))
	cmd.SetArgs([]string{"hook", "prompt-router"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("prompt-router: %v\n%s", err, se.String())
	}
}

// assertPromptKept checks the prompt's own output survived the step.
func assertPromptKept(t *testing.T, stdout string) {
	t.Helper()
	out := decodePromptHook(t, stdout)
	assertAwayBlock(t, "a prompt whose transcript step ran out", out.Context)
	if !strings.Contains(out.Context, "COMMITTING") {
		t.Errorf("a prompt whose transcript step ran out lost its rules:\n%s", out.Context)
	}
	if !strings.Contains(out.Message, "abcd mode back") {
		t.Errorf("a prompt whose transcript step ran out lost the away line: %q", out.Message)
	}
}

// swapDrain replaces the hooks' drain for the rest of the test.
func swapDrain(t *testing.T, f func(context.Context, string, string, history.DrainBudget) (history.DrainResult, error)) {
	t.Helper()
	old := drainStaged
	drainStaged = f
	t.Cleanup(func() { drainStaged = old })
}

// setBound sets one of the drains' bounds for the rest of the test.
func setBound(t *testing.T, bound *time.Duration, d time.Duration) {
	t.Helper()
	old := *bound
	*bound = d
	t.Cleanup(func() { *bound = old })
}

// stderrLines returns the lines of stderr that contain want.
func stderrLines(stderr, want string) []string {
	var got []string
	for _, l := range strings.Split(stderr, "\n") {
		if strings.Contains(l, want) {
			got = append(got, l)
		}
	}
	return got
}

// TestTranscriptBoundsSitUnderTheHostsLimits: the limits the two drains'
// bounds are derived from are the ones hooks/hooks.json gives those hooks, and
// each bound leaves its margin under its limit.
func TestTranscriptBoundsSitUnderTheHostsLimits(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Timeout int `json:"timeout"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	limit := func(event string) time.Duration {
		t.Helper()
		groups := cfg.Hooks[event]
		if len(groups) != 1 || len(groups[0].Hooks) != 1 {
			t.Fatalf("hooks.json: want one %s hook, got %d group(s)", event, len(groups))
		}
		return time.Duration(groups[0].Hooks[0].Timeout) * time.Second
	}
	for _, c := range []struct {
		event         string
		limit, margin time.Duration
		bound         time.Duration
	}{
		{"UserPromptSubmit", promptHookLimit, promptTranscriptMargin, promptTranscriptBound},
		{"SessionStart", sessionStartHookLimit, sessionStartTranscriptMargin, sessionStartTranscriptBound},
	} {
		if got := limit(c.event); got != c.limit {
			t.Errorf("%s: hooks.json gives %s, the drain's bound is derived from %s", c.event, got, c.limit)
		}
		if c.margin <= 0 || c.margin >= c.limit || c.bound != c.limit-c.margin {
			t.Errorf("%s: bound %s is not the limit %s less a margin %s under it", c.event, c.bound, c.limit, c.margin)
		}
	}
}

// TestPromptRouterKeepsItsOutputWhenTheTranscriptStepUsesItsBound: a
// transcript step that uses its whole bound leaves the prompt's rules, away
// block and away line in place, puts nothing on the context channel, and says
// on stderr what it left.
func TestPromptRouterKeepsItsOutputWhenTheTranscriptStepUsesItsBound(t *testing.T) {
	root, _ := budgetCheckout(t)
	setBound(t, &promptTranscriptBound, 300*time.Millisecond)
	swapDrain(t, func(ctx context.Context, _, _ string, _ history.DrainBudget) (history.DrainResult, error) {
		<-ctx.Done() // a step slow enough to use all of it
		return history.DrainResult{Unfinished: []history.DrainProgress{{SessionID: "sess-budget", Done: 3 << 20, Total: 9 << 20}}}, nil
	})
	var so, se bytes.Buffer
	runPrompt(t, root, &so, &se)

	assertPromptKept(t, so.String())
	if strings.Contains(so.String(), "sess-budget") {
		t.Errorf("the transcript step put a session id on the context channel")
	}
	left := stderrLines(se.String(), "sess-budget")
	if len(left) != 1 || !strings.Contains(left[0], "session sess-budget with 3.0 MiB of 9.0 MiB redacted and kept") ||
		!strings.Contains(left[0], "the next prompt continues") || !strings.Contains(left[0], "`abcd history drain`") {
		t.Errorf("want one stderr line saying what was left and how to finish it, got %d", len(left))
	}
}

// firstWrite is a writer that says when it was first written to.
type firstWrite struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	wrote chan struct{}
}

func (w *firstWrite) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf.Len() == 0 && len(p) > 0 {
		close(w.wrote)
	}
	return w.buf.Write(p)
}

func (w *firstWrite) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// TestPromptTranscriptStepRunsBesideTheRestOfTheHook: the hook writes its own
// output while the transcript step is still running; a step run before the
// rest of the hook would hold that output until it ended.
func TestPromptTranscriptStepRunsBesideTheRestOfTheHook(t *testing.T) {
	root, _ := budgetCheckout(t)
	setBound(t, &promptTranscriptBound, time.Minute)
	so := &firstWrite{wrote: make(chan struct{})}
	var besideIt bool
	swapDrain(t, func(ctx context.Context, _, _ string, _ history.DrainBudget) (history.DrainResult, error) {
		select {
		case <-so.wrote:
			besideIt = true
		case <-ctx.Done():
		}
		return history.DrainResult{}, nil
	})
	var se bytes.Buffer
	runPrompt(t, root, so, &se)
	if !besideIt {
		t.Error("the hook wrote nothing while its transcript step ran")
	}
	assertPromptKept(t, so.String())
}

// TestPromptTranscriptBoundCountsFromTheHooksStart: the step starts after the
// root lookup, and its deadline is promptTranscriptBound after the hook
// started, not after the step did; the step drains with the prompt's budget.
func TestPromptTranscriptBoundCountsFromTheHooksStart(t *testing.T) {
	root, rootSHA := budgetCheckout(t)
	setBound(t, &promptTranscriptBound, 2*time.Second)
	const delay = 400 * time.Millisecond
	prev := routerRootCommit
	t.Cleanup(func() { routerRootCommit = prev })
	routerRootCommit = func(context.Context, string) (string, error) {
		time.Sleep(delay) // the hook's own work, before the step starts
		return rootSHA, nil
	}
	var mu sync.Mutex
	var called, deadline time.Time
	var hadDeadline bool
	var budget history.DrainBudget
	swapDrain(t, func(ctx context.Context, _, _ string, b history.DrainBudget) (history.DrainResult, error) {
		mu.Lock()
		defer mu.Unlock()
		called = time.Now()
		deadline, hadDeadline = ctx.Deadline()
		budget = b
		return history.DrainResult{}, nil
	})
	before := time.Now()
	var so, se bytes.Buffer
	runPrompt(t, root, &so, &se)

	mu.Lock()
	defer mu.Unlock()
	if !hadDeadline || called.IsZero() {
		t.Fatal("the transcript step ran with no deadline, or did not run")
	}
	if deadline.Before(before.Add(promptTranscriptBound)) {
		t.Errorf("the deadline is earlier than the bound allows")
	}
	if !deadline.Before(called.Add(promptTranscriptBound - delay/2)) {
		t.Errorf("the deadline was counted from the step's start, not the hook's (%s after the step began)", deadline.Sub(called))
	}
	if budget != livePromptDrainBudget {
		t.Errorf("the prompt drained with budget %+v, want %+v", budget, livePromptDrainBudget)
	}
}

// TestPromptRouterWithItsBoundSpentStillStoresAOnePartTranscript: a bound
// already spent when the step begins still redacts the first part, as the
// whole transcript was at every prompt before, so a transcript of one part is
// stored at that prompt and the prompt keeps its output.
func TestPromptRouterWithItsBoundSpentStillStoresAOnePartTranscript(t *testing.T) {
	root, rootSHA := budgetCheckout(t)
	setBound(t, &promptTranscriptBound, time.Nanosecond)

	var so, se bytes.Buffer
	runPrompt(t, root, &so, &se)

	assertPromptKept(t, so.String())
	if staged, err := history.ListStaged(root, rootSHA); err != nil || len(staged) != 0 {
		t.Errorf("a spent bound left a one-part transcript staged: %d (%v)", len(staged), err)
	}
	if recs, _ := history.List(root, rootSHA); len(recs) != 1 {
		t.Errorf("a spent bound stored %d record(s), want 1", len(recs))
	}
}

// TestPromptTranscriptStepSurvivesAPanic: a panic in the transcript step ends
// the step alone; the hook exits 0 with its output and says so in one stderr
// line that quotes nothing of the fault's value.
func TestPromptTranscriptStepSurvivesAPanic(t *testing.T) {
	root, _ := budgetCheckout(t)
	swapDrain(t, func(context.Context, string, string, history.DrainBudget) (history.DrainResult, error) {
		panic("made-up fault text")
	})
	var so, se bytes.Buffer
	runPrompt(t, root, &so, &se)
	assertPromptKept(t, so.String())
	if got := stderrLines(se.String(), "internal fault"); len(got) != 1 {
		t.Errorf("want one stderr line naming the fault, got %d", len(got))
	}
	if strings.Contains(se.String(), "made-up fault text") {
		t.Error("the stderr line quoted the fault's value")
	}
}

// TestPromptRouterNamesAKeepFaultOnce: a drain that could not keep a part says
// so in one stderr line, with the home path collapsed.
func TestPromptRouterNamesAKeepFaultOnce(t *testing.T) {
	root, _ := budgetCheckout(t)
	home, _ := os.UserHomeDir()
	swapDrain(t, func(context.Context, string, string, history.DrainBudget) (history.DrainResult, error) {
		return history.DrainResult{KeepFault: "history: keep a redacted part: " + home + "/made-up/progress: refused"}, nil
	})
	var so, se bytes.Buffer
	runPrompt(t, root, &so, &se)
	assertPromptKept(t, so.String())
	got := stderrLines(se.String(), "could not be kept")
	if len(got) != 1 {
		t.Fatalf("want one stderr line naming the fault, got %d", len(got))
	}
	if strings.Contains(got[0], home+"/") {
		t.Error("the line carries the home path")
	}
	if !strings.Contains(got[0], "past the bound") {
		t.Error("the hook's line does not say the transcript was finished past its bound")
	}
}

// TestPromptRouterSweepsPartsWithNothingStaged: a prompt with nothing staged
// still removes kept parts whose staged file is gone.
func TestPromptRouterSweepsPartsWithNothingStaged(t *testing.T) {
	root, rootSHA := budgetCheckout(t)
	var so, se bytes.Buffer
	runPrompt(t, root, &so, &se) // stores the one staged transcript
	if staged, _ := history.ListStaged(root, rootSHA); len(staged) != 0 {
		t.Fatal("the test needs nothing staged")
	}
	store, err := history.Resolve(root, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(store.Staging, "progress", "20200101T000000Z-gone.raw.parts")
	if err := os.MkdirAll(orphan, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphan, "0000000000000000.part"), []byte("{}\nredacted"), 0o600); err != nil {
		t.Fatal(err)
	}
	so.Reset()
	se.Reset()
	runPrompt(t, root, &so, &se)
	if _, err := os.Lstat(orphan); err == nil {
		t.Error("a prompt with nothing staged left the parts of a staged file that is gone")
	}
}

// TestSessionStartDrainRunsUnderItsBound: the session-start drain runs with
// the session-start budget, under a deadline sessionStartTranscriptBound after
// the hook started, not after the drain did, and says what it left.
func TestSessionStartDrainRunsUnderItsBound(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	repo, rootSHA := sessionEndRepo(t)
	stageRaw(t, repo, rootSHA, "sess-start", `{"text":"made-up work"}`+"\n")
	var called, deadline time.Time
	var hadDeadline bool
	var budget history.DrainBudget
	swapDrain(t, func(ctx context.Context, _, _ string, b history.DrainBudget) (history.DrainResult, error) {
		called = time.Now()
		deadline, hadDeadline = ctx.Deadline()
		budget = b
		return history.DrainResult{Unfinished: []history.DrainProgress{{SessionID: "sess-start", Done: 1 << 20, Total: 4 << 20}}}, nil
	})
	// The hook's start, as the hook reads it, is an hour back, so a deadline
	// counted from anywhere else is far from the one wanted.
	start := time.Now().Add(-time.Hour)
	prev := sessionStartNow
	t.Cleanup(func() { sessionStartNow = prev })
	sessionStartNow = func() time.Time { return start }
	stdout, stderr := runHook(t, `{"session_id":"s","hook_event_name":"SessionStart","cwd":`+mustJSON(t, repo)+`}`, "hook", "session-start")
	if called.IsZero() || !hadDeadline {
		t.Fatal("the session-start drain did not run, or ran with no deadline")
	}
	if !deadline.Equal(start.Add(sessionStartTranscriptBound)) {
		t.Errorf("the session-start drain's deadline is not sessionStartTranscriptBound after the hook's start (%s off)",
			deadline.Sub(start.Add(sessionStartTranscriptBound)))
	}
	if budget != sessionStartDrainBudget {
		t.Errorf("the session-start drain ran with budget %+v, want %+v", budget, sessionStartDrainBudget)
	}
	if !strings.Contains(stdout+stderr, "session sess-start with 1.0 MiB of 4.0 MiB redacted and kept") {
		t.Error("the session start did not say what its drain left")
	}
}

// TestSessionStartDrainSurvivesAPanic: a panic in the session-start drain ends
// the drain alone, as the prompt's step does; the hook exits 0, says so once,
// and quotes nothing of the fault's value.
func TestSessionStartDrainSurvivesAPanic(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	repo, rootSHA := sessionEndRepo(t)
	stageRaw(t, repo, rootSHA, "sess-start-panic", `{"text":"made-up work"}`+"\n")
	swapDrain(t, func(context.Context, string, string, history.DrainBudget) (history.DrainResult, error) {
		panic("made-up fault text")
	})
	stdout, stderr := runHook(t, `{"session_id":"s","hook_event_name":"SessionStart","cwd":`+mustJSON(t, repo)+`}`, "hook", "session-start")
	if got := strings.Count(stdout+stderr, "internal fault"); got != 1 {
		t.Errorf("want the fault named once, got %d", got)
	}
	if strings.Contains(stdout+stderr, "made-up fault text") {
		t.Error("the session start quoted the fault's value")
	}
}

// TestHistoryDrainNamesAKeepFaultWithNothingStored: the drain verb says a
// fault keeping a part even when the pass stored and failed nothing, as when
// the entry it met was found stored meanwhile.
func TestHistoryDrainNamesAKeepFaultWithNothingStored(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	t.Chdir(repo)
	old := drainEverything
	drainEverything = func(string, string, history.DrainBudget) (history.DrainResult, error) {
		return history.DrainResult{KeepFault: "history: keep a redacted part: made-up refusal"}, nil
	}
	t.Cleanup(func() { drainEverything = old })
	out := string(runCLI(t, "history", "drain"))
	if !strings.Contains(out, "nothing to drain") {
		t.Fatal("the test needs the pass that stored nothing")
	}
	if got := strings.Count(out, "could not be kept"); got != 1 {
		t.Errorf("want the keep fault named once, got %d", got)
	}
	if strings.Contains(out, "bound") {
		t.Error("the drain verb, which has no bound, names one in its keep-fault line")
	}
}

// TestHistoryDrainNamesNoBoundInItsKeepFaultLine: the drain verb runs under no
// bound, so a pass that stored a transcript and met a keep fault says so
// without naming one.
func TestHistoryDrainNamesNoBoundInItsKeepFaultLine(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	t.Chdir(repo)
	old := drainEverything
	drainEverything = func(string, string, history.DrainBudget) (history.DrainResult, error) {
		return history.DrainResult{Captured: []history.Record{{SessionID: "made-up-session"}},
			KeepFault: "history: keep a redacted part: made-up refusal"}, nil
	}
	t.Cleanup(func() { drainEverything = old })
	out := string(runCLI(t, "history", "drain"))
	if !strings.Contains(out, "stored  made-up-session") {
		t.Fatal("the test needs the pass that stored a transcript")
	}
	if got := strings.Count(out, "could not be kept"); got != 1 {
		t.Fatalf("want the keep fault named once, got %d", got)
	}
	if strings.Contains(out, "bound") {
		t.Error("the drain verb, which has no bound, names one in its keep-fault line")
	}
}
