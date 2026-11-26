package matrix

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// The runner-cell launcher (spc-2610161949559050 step 11): a runner cell's copy
// is made by its cell's runner in its clone, in cell mode, and written as its
// trial; nothing stands in for the runner.

// awaitRunner makes copy key's clone and records its hand-off awaiting its
// cell's runner, as the run's start does, and returns the clone.
func awaitRunner(t *testing.T, r *gittest.Repo, home string, m lab.Minted, rec RunRecord, key TrialKey, kind TaskKind) string {
	t.Helper()
	clone, err := makeCopyClone(m.ID, r.Root(), cloneStore(home, rec.RootSHA), rec.RunID, rec.FrozenCommit, key)
	if err != nil {
		t.Fatal(err)
	}
	h := HandOff{SchemaVersion: HandOffSchema, Run: rec.RunID, Key: key, Kind: kind, Source: pinOf(t, rec, key.Cell).Source,
		State: HandOffAwaitingRunner, Clone: clone, Brief: "make the copy", Reason: "waits for its launch"}
	if kind == KindCheck {
		if h.Gates, err = snapshotGates(r.Root()); err != nil {
			t.Fatal(err)
		}
	}
	writeHandOffForTest(t, r, m, h)
	return clone
}

func writeHandOffForTest(t *testing.T, r *gittest.Repo, m lab.Minted, h HandOff) {
	t.Helper()
	data, err := encodeCheck(h)
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	if err := proc.WriteState(handOffRel(h.Run, h.Key), data); err != nil {
		t.Fatal(err)
	}
}

func handOffNow(t *testing.T, r *gittest.Repo, m lab.Minted, rec RunRecord, key TrialKey) HandOff {
	t.Helper()
	h, err := ReadHandOff(r.Root(), m.ID, rec.RunID, key)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// fakeCell is a cell's dispatch that answers from a script, one outcome per
// try, writing answer at the request's Receipt path and transcript to the
// copy's transcript store first, and checking the answer as the runners'
// dispatch does. It counts the copies in flight. With meet, the first copies
// wait for one another until that many are in flight at once, a rendezvous
// rather than a sleep; with release, each copy says it entered on entered and
// waits for release to close.
type fakeCell struct {
	mu         sync.Mutex
	cond       *sync.Cond
	outs       []runner.Outcome
	answer     []byte
	transcript func(req runner.Request) string
	meet       int
	entered    chan struct{}
	release    chan struct{}
	reqs       []runner.Request
	inFlight   int
	peak       int
}

func (f *fakeCell) dispatch(ctx context.Context, req runner.Request, check *copyCheck) (runner.Outcome, error) {
	f.mu.Lock()
	if f.cond == nil {
		f.cond = sync.NewCond(&f.mu)
	}
	f.reqs = append(f.reqs, req)
	f.inFlight++
	f.peak = max(f.peak, f.inFlight)
	f.cond.Broadcast()
	for f.peak < f.meet {
		f.cond.Wait()
	}
	out := servedOutcome()
	if len(f.outs) > 0 {
		out = f.outs[0]
		if len(f.outs) > 1 {
			f.outs = f.outs[1:]
		}
	}
	answer, transcript := f.answer, f.transcript
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.inFlight--
		f.mu.Unlock()
	}()
	if f.release != nil {
		f.entered <- struct{}{}
		select {
		case <-f.release:
		case <-ctx.Done():
		}
	}
	if answer != nil {
		if err := os.WriteFile(req.Receipt, answer, 0o600); err != nil {
			return runner.Outcome{}, err
		}
	}
	if transcript != nil {
		if err := check.Store("opencode", req, runner.Answer{}, []byte(transcript(req))); err != nil {
			return runner.Outcome{}, err
		}
	}
	if out.Answer != nil {
		if err := check.validate("opencode", req, *out.Answer); err != nil {
			return runner.Outcome{Failure: &runner.Failure{Runner: "opencode", Reason: runner.ReasonInvalid, Detail: err.Error()},
				Receipt: runner.RoleReceipt{Role: req.Role, Brief: req.Brief, Contract: req.Receipt, Route: runner.RouteRecord{Asked: "opencode", Ran: "none"}}}, nil
		}
	}
	out.Receipt.Brief, out.Receipt.Contract = req.Brief, req.Receipt
	return out, nil
}

// servedOutcome is a runner's answer, its served side read from its
// harness's events.
func servedOutcome() runner.Outcome {
	sides := runner.Sides{ModelAsked: "openrouter/vendor/model", ServedProvider: "provider-a", HarnessVersion: "1.18.31",
		Usage: &oracle.Usage{PromptTokens: 1000, CompletionTokens: 200}, Cost: &oracle.Cost{Amount: 0.02, Unit: oracle.CostUSD},
		Unpinned: []oracle.Unpinned{{Field: oracle.ServedGenerationID, Reason: oracle.NotReported}}}
	return runner.Outcome{Answer: &runner.Answer{Model: "vendor/model-served", Sides: sides},
		Receipt: runner.RoleReceipt{Role: "implementer", Route: runner.RouteRecord{Asked: "opencode", Ran: "opencode", Model: "vendor/model-served", Sides: sides}}}
}

// madeReceipt is a code copy's receipt that it built the change.
var madeReceipt = []byte(`{"schema_version": 1, "outcome": "made"}`)

// inClone is a transcript naming only the copy's own clone and brief.
func inClone(clone string) func(runner.Request) string {
	return func(req runner.Request) string { return "read " + req.Brief + "\nedited feature.go in " + clone + "\n" }
}

func testLauncher(r *gittest.Repo, home string, m lab.Minted, d cellDispatch) *copyLauncher {
	return &copyLauncher{repoRoot: r.Root(), home: home, labID: m.ID,
		configure: func(CellPin) (cellDispatch, error) { return d, nil }, wait: noWait, now: func() time.Time { return time.Now().UTC() }}
}

func launchOne(t *testing.T, l *copyLauncher, rec RunRecord, key TrialKey) (LaunchResult, error) {
	t.Helper()
	return l.launch(context.Background(), rec.RunID, &key, nil)
}

func trialOf(t *testing.T, r *gittest.Repo, m lab.Minted, rec RunRecord, key TrialKey) (TrialRecord, bool) {
	t.Helper()
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	ts, err := readTrials(proc, rec.RunID, rec.RunID+"/trials")
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range ts {
		if tr.Key == key {
			return tr, true
		}
	}
	return TrialRecord{}, false
}

// launchCell is a copy's dispatch through the runners' real cell mode: the
// implementer routed to opencode on the person's own layer, and on PATH a fake
// opencode that runs script, written in a folder of its own.
func launchCell(t *testing.T, home, script string) cellDispatch {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "opencode"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	for _, v := range []string{"OPENCODE_CONFIG", "OPENCODE_CONFIG_CONTENT", "OPENCODE_CONFIG_DIR"} {
		t.Setenv(v, "")
	}
	roots := layered.Roots{Repo: t.TempDir(), Home: home}
	machine := abcdhome.Path(home, "config.json")
	if err := os.MkdirAll(filepath.Dir(machine), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(machine, []byte(`{"roles":{"implementer":{"runner":"opencode"}},"runner":{"opencode":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := runner.Load(roots)
	if err != nil {
		t.Fatal(err)
	}
	return dispatchOn(c)
}

// openCodeStream is opencode 1.18.31's run mode as its fixture records it.
func openCodeStream(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "runner", "testdata", "opencode-run-1.18.31.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// fakeOpenCode is a fake opencode that logs its working folder and prompt to
// logDir, writes receipt at the prompt's Receipt path (none when receipt is
// empty), prints stream and exits with exit.
func fakeOpenCode(logDir, receipt, stream string, exit int) string {
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
	write := ""
	if receipt != "" {
		write = "printf '%s' " + q(receipt) + " > \"$r\"\n"
	}
	return "#!/bin/sh\n" +
		"pwd -P > " + q(filepath.Join(logDir, "cwd")) + "\n" +
		"r=''\nfor a in \"$@\"; do case \"$a\" in *'Receipt: '*) r=$(printf '%s\\n' \"$a\" | sed -n 's/^Receipt: //p');; esac; done\n" +
		"printf '%s\\n' \"$r\" > " + q(filepath.Join(logDir, "receipt-path")) + "\n" +
		"for a in \"$@\"; do case \"$a\" in *'Brief: '*) printf '%s\\n' \"$a\" | sed -n 's/^Brief: //p' > " + q(filepath.Join(logDir, "brief-path")) + ";; esac; done\n" +
		write +
		"printf '%s\\n' " + q(stream) + "\n" +
		fmt.Sprintf("exit %d\n", exit)
}

func readLog(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("the fake runner logged no %s: %v", name, err)
	}
	return strings.TrimSpace(string(raw))
}

// TestALaunchedCopyIsMadeInItsCloneInCellMode: a runner cell's code copy is
// dispatched through the runners' real cell mode, its runner started in the
// copy's clone on the brief in its own folder of the hand-off store, outside
// the clone, the clone store and the lab's state; its trial is kept awaiting
// its hidden tests' result, then written when the host hands that back, its
// served side the runner's role receipt and never marked self-reported.
func TestALaunchedCopyIsMadeInItsCloneInCellMode(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	clone := awaitRunner(t, r, home, m, rec, key, KindCode)
	logs := t.TempDir()
	d := launchCell(t, home, fakeOpenCode(logs, string(madeReceipt), openCodeStream(t), 0))
	res, err := launchOne(t, testLauncher(r, home, m, d), rec, key)
	if err != nil {
		t.Fatalf("launch = %v", err)
	}
	if len(res.Copies) != 1 || res.Copies[0].State != HandOffLaunched || res.Copies[0].Status != "" {
		t.Fatalf("the copy is launched and awaits its hidden tests' result: %+v", res.Copies)
	}
	realClone, err := filepath.EvalSymlinks(clone)
	if err != nil {
		t.Fatal(err)
	}
	if cwd := readLog(t, logs, "cwd"); cwd != realClone {
		t.Fatalf("the runner started in %s, not the copy's clone %s", cwd, realClone)
	}
	brief, receipt := readLog(t, logs, "brief-path"), readLog(t, logs, "receipt-path")
	alias := cloneAlias(t, home, m, rec, key)
	own := handoffDir(handoffStore(home, rec.RootSHA), rec.RunID, alias)
	for what, p := range map[string]string{"brief": brief, "receipt": receipt} {
		if filepath.Dir(p) != own {
			t.Errorf("the %s %s is not in the copy's own hand-off folder %s", what, p, own)
		}
		for name, place := range map[string]string{"clone": clone, "clone store": cloneStore(home, rec.RootSHA),
			"lab state": labStore(home, rec.RootSHA), "checkout": r.Root()} {
			if under(filepath.ToSlash(p), filepath.ToSlash(place)) {
				t.Errorf("the %s %s sits in the %s", what, p, name)
			}
		}
	}
	text, err := os.ReadFile(brief)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{key.Task, key.Cell, "openrouter", "vendor/model", rec.RunID} {
		if strings.Contains(string(text), leak) {
			t.Errorf("the brief names %q:\n%s", leak, text)
		}
	}
	if _, ok := trialOf(t, r, m, rec, key); ok {
		t.Fatal("a made code copy's trial waits for its hidden tests' result")
	}
	h := handOffNow(t, r, m, rec, key)
	if h.State != HandOffLaunched || h.Trial == nil || h.Trial.Compliance == nil {
		t.Fatalf("the hand-off keeps the launch's trial, its compliance checked: %+v", h)
	}
	// The host hands back the hidden tests' result alone.
	got, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, CopyBack{Receipt: []byte(`{"schema_version":1,"outcome":"made","score":{"passed":3,"total":4}}`)})
	if err != nil {
		t.Fatalf("CollectCopy = %v", err)
	}
	if got.Status != StatusFailed || got.Score == nil || got.Score.Passed != 3 || !got.HostReportedScore() {
		t.Fatalf("the trial carries the host's score, marked so: %+v", got)
	}
	if len(got.Roles) != 1 || got.Roles[0].Route.Ran != "opencode" || got.Usage.PromptTokens == 0 || got.Cost == nil || got.Cost.Unit != oracle.CostUSD {
		t.Fatalf("the served side is the runner's role receipt: %+v %+v %+v", got.Roles, got.Usage, got.Cost)
	}
	for _, u := range got.Unpinned {
		if u.Reason == oracle.SelfReported && u.Field != TrialScore {
			t.Errorf("a launched copy's %s is marked self-reported", u.Field)
		}
		if u.Field == TrialCalls {
			t.Errorf("a launched copy's calls are its role receipt, never listed as not reported")
		}
	}
	if strings.Contains(trialAt(t, home, m, rec, key), home) {
		t.Errorf("the trial carries the home path")
	}
	if c := ruleOf(t, *got.Compliance, RuleIsolation); c.State != StateKept {
		t.Errorf("a launched copy's isolation is abcd's launch in its clone: %+v", c)
	}
	// Its score is handed back once.
	if _, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, CopyBack{Receipt: []byte(`{"schema_version":1,"outcome":"made","score":{"passed":4,"total":4}}`)}); err == nil {
		t.Fatal("a second hand-back is refused")
	}
}

// TestALaunchedRunnerThatFailsIsTheCellsFailure: a runner that exits non-zero,
// one that answers in no structured event, and one that writes no answer at
// its Receipt path are each the cell's failure, written as a failed trial
// after one try: no fallback receipt, and no host dispatch.
func TestALaunchedRunnerThatFailsIsTheCellsFailure(t *testing.T) {
	for name, script := range map[string]func(logs string, stream string) string{
		"exits non-zero": func(logs, stream string) string { return fakeOpenCode(logs, string(madeReceipt), stream, 1) },
		"unparsable":     func(logs, _ string) string { return fakeOpenCode(logs, string(madeReceipt), "not an event", 0) },
		"no answer":      func(logs, stream string) string { return fakeOpenCode(logs, "", stream, 0) },
		"bad receipt": func(logs, stream string) string {
			return fakeOpenCode(logs, `{"outcome":"made","score":{"passed":9}}`, stream, 0)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r, home, m, rec, _ := checkLab(t)
			key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
			awaitRunner(t, r, home, m, rec, key, KindCode)
			logs := t.TempDir()
			d := launchCell(t, home, script(logs, openCodeStream(t)))
			res, err := launchOne(t, testLauncher(r, home, m, d), rec, key)
			if err != nil {
				t.Fatalf("a runner's failure is the cell's, not the run's: %v", err)
			}
			got, ok := trialOf(t, r, m, rec, key)
			if !ok || got.Status != StatusFailed || len(got.Fallbacks) != 0 || !got.Failed() || res.Copies[0].Status != StatusFailed {
				t.Fatalf("the copy is its cell's failure with no fallback: %+v %+v", got, res)
			}
			if got.Roles[0].Route.Ran != "none" {
				t.Fatalf("nothing else ran the role: %+v", got.Roles[0].Route)
			}
		})
	}
}

// TestACopyCarryingAFallbackReceiptIsExcluded: a dispatch that carries a
// fallback receipt never counts as the model's work: the copy is excluded with
// the receipt kept, counted as its cell's failure.
func TestACopyCarryingAFallbackReceiptIsExcluded(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	awaitRunner(t, r, home, m, rec, key, KindCode)
	fb := runner.FallbackReceipt{At: time.Now().UTC(), Role: "implementer", Asked: "opencode", Reason: runner.ReasonFailed, Detail: "it failed", Ran: runner.Host}
	f := &fakeCell{outs: []runner.Outcome{{Handoff: true, Fallback: &fb, Receipt: runner.RoleReceipt{Role: "implementer", Route: runner.RouteRecord{Asked: "opencode", Ran: runner.Host}}}}}
	if _, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, key); err != nil {
		t.Fatal(err)
	}
	got, ok := trialOf(t, r, m, rec, key)
	if !ok || got.Status != StatusExcluded || len(got.Fallbacks) != 1 || !got.Failed() || got.Passed() {
		t.Fatalf("a fallen-back copy is excluded as its cell's failure: %+v", got)
	}
}

// TestAProviderDownIsRetriedOnTheSamePinAndRecordedNotStarted: a provider down
// before the copy began is tried again on the identical pin, and a copy the
// outage outlasted is recorded not started, never scored.
func TestAProviderDownIsRetriedOnTheSamePinAndRecordedNotStarted(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	awaitRunner(t, r, home, m, rec, key, KindCode)
	// The outage is a real one, read through the opencode adapter with its
	// fake on PATH for that reading alone.
	var outage runner.Outcome
	t.Run("outage", func(t *testing.T) { outage = down(t) })
	f := &fakeCell{outs: []runner.Outcome{outage}}
	var pins []CellPin
	l := testLauncher(r, home, m, f.dispatch)
	l.configure = func(p CellPin) (cellDispatch, error) { pins = append(pins, p); return f.dispatch, nil }
	if _, err := launchOne(t, l, rec, key); err != nil {
		t.Fatal(err)
	}
	if len(f.reqs) != outageTries {
		t.Fatalf("the copy is tried %d times on its pin, not %d", len(f.reqs), outageTries)
	}
	for _, p := range pins {
		if !reflect.DeepEqual(p, pinOf(t, rec, key.Cell)) {
			t.Fatalf("a try ran on another pin: %+v", p)
		}
	}
	got, ok := trialOf(t, r, m, rec, key)
	if !ok || got.Status != StatusNotStarted || got.Score != nil || got.Cost != nil || got.Attempted() || got.Failed() {
		t.Fatalf("a copy the outage outlasted is not started and never scored: %+v", got)
	}
}

// TestALaunchRefusesAHostCellsCopyAndARefusedCells: the launcher never starts
// a host cell's copy, nor one whose cell the secret scan refused, nor one
// already launched; each is refused by its key with nothing written, and
// passed over in a launch of the whole run.
func TestALaunchRefusesAHostCellsCopyAndARefusedCells(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	host := TrialKey{Task: "feature-a", Cell: "host-big", Repeat: 1}
	handOffWithClone(t, r, home, m, rec, host, KindCode)
	refused := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 2}
	handOffIn(t, r, m, rec, refused, KindCode, HandOffRefused, "its clone fails the secret scan")
	f := &fakeCell{answer: madeReceipt}
	l := testLauncher(r, home, m, f.dispatch)
	for key, want := range map[TrialKey]string{host: "is a host cell's", refused: "was refused before it started"} {
		if _, err := launchOne(t, l, rec, key); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("launching %s: %v; want a refusal naming %q", key, err, want)
		}
	}
	res, err := l.launch(context.Background(), rec.RunID, nil, nil)
	if err != nil || len(res.Copies) != 0 {
		t.Fatalf("a launch of the run passes both over: %+v %v", res, err)
	}
	if len(f.reqs) != 0 {
		t.Fatalf("the launcher dispatched %d copies", len(f.reqs))
	}
	if h := handOffNow(t, r, m, rec, host); h.State != HandOffWaiting {
		t.Fatalf("the host copy's hand-off is unchanged: %+v", h)
	}
	if _, err := launchOne(t, l, rec, TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 9}); err == nil {
		t.Fatal("a copy the run did not hand off is refused")
	}
}

// TestAProviderBoundCloneIsScannedAgainBeforeItsLaunch: a clone that picked up
// a secret-shaped file after it was handed off is refused at its launch,
// before its runner starts, and its clone removed.
func TestAProviderBoundCloneIsScannedAgainBeforeItsLaunch(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	clone := awaitRunner(t, r, home, m, rec, key, KindCode)
	if err := os.WriteFile(filepath.Join(clone, "config.txt"), []byte("token = "+"gh"+"p_"+testsecret.Synthetic(411, 36)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &fakeCell{answer: madeReceipt}
	res, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.reqs) != 0 || res.Copies[0].State != HandOffRefused {
		t.Fatalf("a clone the scan refuses is never launched: %d dispatches, %+v", len(f.reqs), res.Copies)
	}
	if _, err := os.Lstat(clone); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the refused clone is removed: %v", err)
	}
}

// TestALaunchTheCapsRemainderDoesNotCoverIsRefused: a metered copy is launched
// only while the spending cap's remainder, less the copies in flight, covers
// its declared cost; one it does not cover is left waiting and named.
func TestALaunchTheCapsRemainderDoesNotCoverIsRefused(t *testing.T) {
	r, home, m, rec, _ := checkLab(t, func(in *runInputs) {
		in.Declaration = []byte(strings.Replace(string(in.Declaration), `"settings": {"temperature": 0.6}`,
			`"settings": {"temperature": 0.6}, "cost_per_copy": {"amount": 10, "unit": "usd"}`, 1))
	})
	spent := codeTrial(t, rec, "or-cheap", 1, StatusPassed)
	spent.Cost = &oracle.Cost{Amount: 12, Unit: oracle.CostUSD}
	if err := writeTrial(r.Root(), m.ID, spent); err != nil {
		t.Fatal(err)
	}
	for rep := 2; rep <= 3; rep++ {
		awaitRunner(t, r, home, m, rec, TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: rep}, KindCode)
	}
	f := &fakeCell{answer: madeReceipt, transcript: func(req runner.Request) string { return "read " + req.Brief + "\n" }}
	res, err := testLauncher(r, home, m, f.dispatch).launch(context.Background(), rec.RunID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 25 cap, 12 spent: one copy at 10 fits, and the second does not.
	if len(f.reqs) != 1 || len(res.Copies) != 2 {
		t.Fatalf("one copy is launched and one left waiting: %d dispatches, %+v", len(f.reqs), res.Copies)
	}
	var waiting *LaunchedCopy
	for i, c := range res.Copies {
		if c.State == HandOffAwaitingRunner {
			waiting = &res.Copies[i]
		}
	}
	if waiting == nil || !strings.Contains(waiting.Reason, "cap") {
		t.Fatalf("the copy the cap does not cover waits, named: %+v", res.Copies)
	}
	if h := handOffNow(t, r, m, rec, waiting.Key); h.State != HandOffAwaitingRunner {
		t.Fatalf("its hand-off still waits: %+v", h)
	}
}

// TestCopiesInFlightNeverExceedTheDeclaredConcurrency: a launch of the run
// makes its copies at most as many at once as the run declares, and as many
// as that when it can.
func TestCopiesInFlightNeverExceedTheDeclaredConcurrency(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	for rep := 1; rep <= 5; rep++ {
		awaitRunner(t, r, home, m, rec, TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: rep}, KindCode)
	}
	// The first copies wait until as many as the run declares are in flight,
	// so the figure is reached however slowly each copy gets to its runner.
	f := &fakeCell{answer: madeReceipt, meet: rec.Concurrency, transcript: func(req runner.Request) string { return "read " + req.Brief + "\n" }}
	res, err := testLauncher(r, home, m, f.dispatch).launch(context.Background(), rec.RunID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Copies) != 5 || len(f.reqs) != 5 {
		t.Fatalf("every copy is launched: %+v", res.Copies)
	}
	if f.peak != rec.Concurrency {
		t.Fatalf("%d copies were in flight at once; the run declares %d", f.peak, rec.Concurrency)
	}
}

// TestACopyIsLaunchedOnce: a copy's claim is created exclusively, so a second
// launch of it, at once or after, never runs it again.
func TestACopyIsLaunchedOnce(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	awaitRunner(t, r, home, m, rec, key, KindCode)
	alias := cloneAlias(t, home, m, rec, key)
	_, root, err := claimCopy(handoffStore(home, rec.RootSHA), rec.RunID, alias)
	if err != nil {
		t.Fatal(err)
	}
	root.Close()
	f := &fakeCell{answer: madeReceipt}
	res, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, key)
	if err != nil || len(f.reqs) != 0 || !strings.Contains(res.Copies[0].Reason, "claimed") {
		t.Fatalf("a claimed copy is not launched again: %+v %v, %d dispatches", res, err, len(f.reqs))
	}
}

// TestATranscriptNamingASiblingsBriefFolderBreaksOwnCopyOnly: a launched
// copy whose transcript reads another copy's folder in the hand-off store read
// outside its own copy, and is excluded; its own brief folder is its own.
func TestATranscriptNamingASiblingsBriefFolderBreaksOwnCopyOnly(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	sibling := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 2}
	clone := awaitRunner(t, r, home, m, rec, key, KindCode)
	awaitRunner(t, r, home, m, rec, sibling, KindCode)
	other := handoffDir(handoffStore(home, rec.RootSHA), rec.RunID, cloneAlias(t, home, m, rec, sibling))
	f := &fakeCell{answer: madeReceipt, transcript: func(req runner.Request) string {
		return inClone(clone)(req) + "cat " + filepath.Join(other, launchBriefName) + "\n"
	}}
	if _, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, key); err != nil {
		t.Fatal(err)
	}
	got, ok := trialOf(t, r, m, rec, key)
	if !ok || got.Status != StatusExcluded {
		t.Fatalf("a copy that read a sibling's brief folder is excluded: %+v", got)
	}
	if c := ruleOf(t, *got.Compliance, RuleOwnCopyOnly); c.State != StateBroken || !strings.Contains(c.Detail, "hand-off store") || strings.Contains(c.Detail, home) {
		t.Fatalf("own_copy_only is broken, naming the hand-off store and not the home path: %+v", c)
	}
}

// TestALaunchedWritingCopyIsKeptAndACheckCopysFindingsRecorded: a writing
// copy's document is the answer its runner writes, kept for its judging; a
// check copy's findings are read into the run's record before its clone is
// deleted.
func TestALaunchedWritingCopyIsKeptAndACheckCopysFindingsRecorded(t *testing.T) {
	// The run's check task declares docs/reference as its material.
	r, home, m, rec, _ := checkLabWith(t, map[string]string{"docs/reference/cli.md": "# CLI\n\nIt refuses a hazard.\n"})
	doc := TrialKey{Task: "doc-page", Cell: "or-cheap", Repeat: 1}
	docClone := awaitRunner(t, r, home, m, rec, doc, KindWriting)
	f := &fakeCell{answer: []byte("# The guide\n\nIt does what it says.\n"), transcript: inClone(docClone)}
	if _, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, doc); err != nil {
		t.Fatal(err)
	}
	if got, ok := trialOf(t, r, m, rec, doc); !ok || got.Status != StatusPassed {
		t.Fatalf("the writing copy's trial is written as made: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(runDir(home, m, rec.RunID), writingRel("", doc))); err != nil {
		t.Fatalf("the writing copy's document is kept: %v", err)
	}

	check := TrialKey{Task: "fidelity", Cell: "or-cheap", Repeat: 1}
	checkClone := awaitRunner(t, r, home, m, rec, check, KindCheck)
	f = &fakeCell{answer: dfAnswer(t, "The guard refuses a push without a receipt."), transcript: inClone(checkClone)}
	if _, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, check); err != nil {
		t.Fatal(err)
	}
	if got, ok := trialOf(t, r, m, rec, check); !ok || got.Kind != KindCheck {
		t.Fatalf("the check copy's trial is written: %+v", got)
	}
	if _, err := os.Lstat(checkClone); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the check copy's clone is deleted once its findings are recorded: %v", err)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	if _, err := proc.ReadState(checkCopyRel(rec.RunID, check)); err != nil {
		t.Fatalf("the check copy's findings are recorded: %v", err)
	}
}

// meteredOutcome is servedOutcome with the runner reporting it spent amount.
func meteredOutcome(amount float64) runner.Outcome {
	out := servedOutcome()
	c := &oracle.Cost{Amount: amount, Unit: oracle.CostUSD}
	out.Answer.Sides.Cost, out.Receipt.Route.Sides.Cost = c, c
	return out
}

// TestASecondLaunchCountsWhatTheFirstLeftAgainstTheCap: a launch made before
// any score is handed back counts what the copies an earlier launch made are
// spending, though none holds a trial yet, so the spending cap holds across
// launches as it holds within one.
func TestASecondLaunchCountsWhatTheFirstLeftAgainstTheCap(t *testing.T) {
	r, home, m, rec, _ := checkLab(t, func(in *runInputs) {
		in.Declaration = []byte(strings.Replace(string(in.Declaration), `"settings": {"temperature": 0.6}`,
			`"settings": {"temperature": 0.6}, "cost_per_copy": {"amount": 10, "unit": "usd"}`, 1))
	})
	for rep := 1; rep <= 4; rep++ {
		awaitRunner(t, r, home, m, rec, TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: rep}, KindCode)
	}
	cell := func() *fakeCell {
		return &fakeCell{outs: []runner.Outcome{meteredOutcome(10)}, answer: madeReceipt,
			transcript: func(req runner.Request) string { return "read " + req.Brief + "\n" }}
	}
	first := cell()
	res, err := testLauncher(r, home, m, first.dispatch).launch(context.Background(), rec.RunID, nil, nil)
	if err != nil || len(first.reqs) != 2 {
		t.Fatalf("the first launch makes the two copies a cap of 25 covers at 10 each: %d dispatches, %+v %v", len(first.reqs), res.Copies, err)
	}
	second := cell()
	res, err = testLauncher(r, home, m, second.dispatch).launch(context.Background(), rec.RunID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.reqs) != 0 {
		t.Fatalf("a second launch, before any score is handed back, spent past the cap: %d more copies launched, %+v", len(second.reqs), res.Copies)
	}
	for _, c := range res.Copies {
		if c.State != HandOffAwaitingRunner || !strings.Contains(c.Reason, "cap") {
			t.Errorf("copy %s waits on the cap: %+v", c.Key, c)
		}
	}
}

// launchLockAt is run's launch lock in the hand-off store, as the test reads
// where it sits.
func launchLockAt(home string, rec RunRecord) string {
	return filepath.Join(handoffStore(home, rec.RootSHA), rec.RunID, "launch.lock")
}

// writeBegun records copy key's launch as begun, and never ended, as a launch
// killed outright leaves it.
func writeBegun(t *testing.T, r *gittest.Repo, m lab.Minted, rec RunRecord, key TrialKey, ended []byte) {
	t.Helper()
	h := handOffNow(t, r, m, rec, key)
	h.State, h.Reason = HandOffLaunching, "its launch began"
	data, err := encodeCheck(h)
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	if err := proc.WriteState(launchRel(rec.RunID, key, launchBegun), data); err != nil {
		t.Fatal(err)
	}
	if ended != nil {
		if err := proc.WriteState(launchRel(rec.RunID, key, launchEnded), ended); err != nil {
			t.Fatal(err)
		}
	}
}

// TestALaunchThatEndedWithNoRecordReadsAsInterrupted: a copy whose launch
// began and wrote no end, its run's launch lock free, was cut off with its
// launch (killed outright, or its machine lost): it reads as interrupted, so
// its task's review and judging stop waiting for it and it is not launched
// again; while a launch holds the lock it reads as launching.
func TestALaunchThatEndedWithNoRecordReadsAsInterrupted(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	awaitRunner(t, r, home, m, rec, key, KindCode)
	writeBegun(t, r, m, rec, key, nil)
	if err := os.MkdirAll(filepath.Dir(launchLockAt(home, rec)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsutil.WithFileLock(launchLockAt(home, rec), time.Second, func() error {
		if h := handOffNow(t, r, m, rec, key); h.State != HandOffLaunching {
			t.Errorf("while a launch holds the run's lock, a begun copy is launching: %+v", h)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	h := handOffNow(t, r, m, rec, key)
	if h.State != HandOffInterrupted || !strings.Contains(h.Reason, "no record") {
		t.Fatalf("a begun copy with no end and the lock free reads as interrupted: %+v", h)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	for _, what := range []string{"its review", "its judging"} {
		if err := awaitHost(proc, rec.RunID, key.Task, what); err != nil {
			t.Errorf("%s waits for a copy whose launch is gone: %v", what, err)
		}
	}
	f := &fakeCell{answer: madeReceipt}
	if _, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, key); err == nil || len(f.reqs) != 0 {
		t.Fatalf("a copy whose launch is gone is not launched again: %v, %d dispatches", err, len(f.reqs))
	}
}

// TestATornEndRecordReadsAsInterrupted: an end record that does not parse (a
// launch killed while writing it) reads as the copy interrupted, naming the
// record, and the rest of the run still reads.
func TestATornEndRecordReadsAsInterrupted(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	other := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 2}
	awaitRunner(t, r, home, m, rec, key, KindCode)
	awaitRunner(t, r, home, m, rec, other, KindCode)
	writeBegun(t, r, m, rec, key, []byte(`{"schema_vers`))
	h := handOffNow(t, r, m, rec, key)
	if h.State != HandOffInterrupted || !strings.Contains(h.Reason, launchRel(rec.RunID, key, launchEnded)) {
		t.Fatalf("a torn end record reads as interrupted, naming it: %+v", h)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	all, err := readHandOffs(proc, rec.RunID)
	if err != nil || len(all) != 2 || all[1].State != HandOffAwaitingRunner {
		t.Fatalf("the rest of the run still reads: %+v %v", all, err)
	}
}

// TestASecondLaunchOfARunIsRefusedWhileOneRuns: a run is launched by one
// launch at a time; a second, while the first holds the run's launch lock, is
// refused naming who holds it, and starts nothing.
func TestASecondLaunchOfARunIsRefusedWhileOneRuns(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	first, second := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}, TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 2}
	awaitRunner(t, r, home, m, rec, first, KindCode)
	awaitRunner(t, r, home, m, rec, second, KindCode)
	held := &fakeCell{answer: madeReceipt, entered: make(chan struct{}, 1), release: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, err := testLauncher(r, home, m, held.dispatch).launch(context.Background(), rec.RunID, &first, nil)
		done <- err
	}()
	<-held.entered
	f := &fakeCell{answer: madeReceipt}
	_, err := testLauncher(r, home, m, f.dispatch).launch(context.Background(), rec.RunID, nil, nil)
	close(held.release)
	if ferr := <-done; ferr != nil {
		t.Fatalf("the first launch: %v", ferr)
	}
	if err == nil || !strings.Contains(err.Error(), "one launch at a time") || !strings.Contains(err.Error(), fmt.Sprintf("pid %d", os.Getpid())) {
		t.Fatalf("a second launch while one runs is refused naming who holds it: %v", err)
	}
	if len(f.reqs) != 0 {
		t.Fatalf("the refused launch started %d copies", len(f.reqs))
	}
}

// TestAClaimIsReleasedWhenTheLaunchCannotRecordItsStart: a copy whose start
// cannot be recorded (the lab halted mid-launch) gives its claim back, so it
// still waits for its runner and a later launch makes it.
func TestAClaimIsReleasedWhenTheLaunchCannotRecordItsStart(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	awaitRunner(t, r, home, m, rec, key, KindCode)
	halt := filepath.Join(runDir(home, m, rec.RunID), "..", "..", "halt-sweep.json")
	f := &fakeCell{answer: madeReceipt, transcript: func(req runner.Request) string { return "read " + req.Brief + "\n" }}
	l := testLauncher(r, home, m, f.dispatch)
	l.configure = func(CellPin) (cellDispatch, error) {
		// The lab halts after the copies are picked, before any is begun.
		return f.dispatch, os.WriteFile(halt, []byte("{}"), 0o600)
	}
	if _, err := launchOne(t, l, rec, key); err == nil {
		t.Fatal("a launch whose start cannot be recorded fails")
	}
	if err := os.Remove(halt); err != nil {
		t.Fatal(err)
	}
	claim := filepath.Join(handoffDir(handoffStore(home, rec.RootSHA), rec.RunID, cloneAlias(t, home, m, rec, key)), launchClaimName)
	if _, err := os.Lstat(claim); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the copy's claim is given back: %v", err)
	}
	if h := handOffNow(t, r, m, rec, key); h.State != HandOffAwaitingRunner {
		t.Fatalf("the copy still waits for its runner: %+v", h)
	}
	if res, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, key); err != nil || res.Copies[0].State != HandOffLaunched {
		t.Fatalf("a later launch makes it: %+v %v", res, err)
	}
}

// endLands is a run's state as a reader racing a launch's last moment sees
// it: the end record of hidden is not there at the first read, and is at
// every read after.
type endLands struct {
	stateReader
	hidden string
	reads  int
}

func (e *endLands) ReadState(rel string) ([]byte, error) {
	if rel == e.hidden {
		e.reads++
		if e.reads == 1 {
			return nil, fmt.Errorf("%s is not written: %w", rel, fs.ErrNotExist)
		}
	}
	return e.stateReader.ReadState(rel)
}

// TestAnEndRecordWrittenAsTheLockIsFreedIsRead: a launch that writes its end
// record and lets go of the run's lock between a reader's first read and its
// look at the lock is read at its end, never as interrupted.
func TestAnEndRecordWrittenAsTheLockIsFreedIsRead(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	awaitRunner(t, r, home, m, rec, key, KindCode)
	h := handOffNow(t, r, m, rec, key)
	end := h
	end.State, end.Reason = HandOffLaunched, "its runner made the copy"
	data, err := encodeCheck(end)
	if err != nil {
		t.Fatal(err)
	}
	writeBegun(t, r, m, rec, key, data)
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	got := launchOverlay(&endLands{stateReader: proc, hidden: launchRel(rec.RunID, key, launchEnded)}, rec.RunID, h)
	if got.State != HandOffLaunched {
		t.Fatalf("a copy whose end landed as its lock was freed reads at its end: %+v", got)
	}
}

// TestALaunchClosesTheRunsFoldersToListingWhileItsCopiesRun: while a launch's
// copies run, the run's folder in the hand-off store and in the clone store
// cannot be listed, by the copy or by anything else of the account, though
// the copy keeps its own brief folder and clone; each has its mode back once
// the launch ends, whether it ended as it should, was stopped (a SIGTERM or
// a SIGHUP, which the command turns into a stop) or failed.
func TestALaunchClosesTheRunsFoldersToListingWhileItsCopiesRun(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the superuser lists a folder whatever its mode")
	}
	for name, end := range map[string]func(ctx context.Context, stop context.CancelFunc, f *fakeCell, req runner.Request, check *copyCheck) (runner.Outcome, error){
		"an end": func(ctx context.Context, _ context.CancelFunc, f *fakeCell, req runner.Request, check *copyCheck) (runner.Outcome, error) {
			return f.dispatch(ctx, req, check)
		},
		"a stop": func(ctx context.Context, stop context.CancelFunc, _ *fakeCell, _ runner.Request, _ *copyCheck) (runner.Outcome, error) {
			stop()
			<-ctx.Done()
			return runner.Outcome{}, ctx.Err()
		},
		"a failed launch": func(context.Context, context.CancelFunc, *fakeCell, runner.Request, *copyCheck) (runner.Outcome, error) {
			return runner.Outcome{}, errors.New("the runner refused the dispatch whole")
		},
	} {
		t.Run(name, func(t *testing.T) {
			r, home, m, rec, _ := checkLab(t)
			key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
			clone := awaitRunner(t, r, home, m, rec, key, KindCode)
			awaitRunner(t, r, home, m, rec, TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 2}, KindCode)
			handRun := filepath.Join(handoffStore(home, rec.RootSHA), rec.RunID)
			cloneRunDir := filepath.Dir(clone)
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			f := &fakeCell{answer: madeReceipt, transcript: inClone(clone)}
			var listed []error
			var ownErr error
			dispatch := func(ctx context.Context, req runner.Request, check *copyCheck) (runner.Outcome, error) {
				for _, dir := range []string{filepath.Dir(filepath.Dir(req.Brief)), filepath.Dir(req.Dir)} {
					_, err := os.ReadDir(dir)
					listed = append(listed, err)
				}
				if _, err := os.ReadDir(filepath.Dir(req.Brief)); err != nil {
					ownErr = err
				} else if _, err := os.ReadDir(req.Dir); err != nil {
					ownErr = err
				}
				return end(ctx, stop, f, req, check)
			}
			_, err := testLauncher(r, home, m, dispatch).launch(ctx, rec.RunID, &key, nil)
			if (name == "an end") != (err == nil) {
				t.Fatalf("the launch returned %v", err)
			}
			refused := len(listed) >= 2
			for _, err := range listed {
				refused = refused && errors.Is(err, fs.ErrPermission)
			}
			if !refused {
				t.Fatalf("listing the run's folders while its copy ran: %v; want each refused", listed)
			}
			if ownErr != nil {
				t.Fatalf("the copy could not list its own folders: %v", ownErr)
			}
			for _, dir := range []string{handRun, cloneRunDir} {
				fi, err := os.Stat(dir)
				if err != nil || fi.Mode().Perm() != 0o700 {
					t.Fatalf("%s after the launch: %v, %v; want its mode 0700 back", filepath.Base(dir), fi.Mode().Perm(), err)
				}
			}
		})
	}
}

// TestALaunchKilledOutrightLeavesNoFolderClosed: a launch killed outright
// (SIGKILL, a power loss) runs no deferred restore and leaves the run's
// folders closed to listing. The next launch of the run, holding its lock,
// gives them their mode back before it reads anything, though it finds no
// copy to launch, and so does a hand-back that finds the lock free; a
// hand-back while a launch holds the lock leaves them closed, and reads a
// pattern that meets a folder it cannot list as unexamined, never as one that
// matched nothing.
func TestALaunchKilledOutrightLeavesNoFolderClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the superuser lists a folder whatever its mode")
	}
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 1}
	clone := awaitRunner(t, r, home, m, rec, key, KindCode)
	host := TrialKey{Task: "feature-a", Cell: "host-big", Repeat: 1}
	hostClone := handOffWithClone(t, r, home, m, rec, host, KindCode)
	host2 := TrialKey{Task: "feature-a", Cell: "host-big", Repeat: 2}
	hostClone2 := handOffWithClone(t, r, home, m, rec, host2, KindCode)
	f := &fakeCell{answer: madeReceipt, transcript: inClone(clone)}
	if _, err := launchOne(t, testLauncher(r, home, m, f.dispatch), rec, key); err != nil {
		t.Fatal(err)
	}
	dirs := []string{filepath.Join(handoffStore(home, rec.RootSHA), rec.RunID), filepath.Join(cloneStore(home, rec.RootSHA), rec.RunID)}
	closeAll := func() {
		t.Helper()
		for _, dir := range dirs {
			if err := os.Chmod(dir, listingDenied); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Cleanup(func() {
		for _, dir := range dirs {
			os.Chmod(dir, 0o700)
		}
	})
	modes := func(what string, want fs.FileMode) {
		t.Helper()
		for _, dir := range dirs {
			fi, err := os.Stat(dir)
			if err != nil || fi.Mode().Perm() != want {
				t.Fatalf("%s: %s holds %v (%v); want %v", what, filepath.Base(filepath.Dir(dir)), fi.Mode().Perm(), err, want)
			}
		}
	}
	closeAll()
	got, err := testLauncher(r, home, m, f.dispatch).launch(context.Background(), rec.RunID, nil, nil)
	if err != nil || len(got.Copies) != 0 {
		t.Fatalf("a launch with no copy to launch = %+v, %v", got, err)
	}
	modes("after a launch that found no copy to launch", 0o700)

	closeAll()
	back := func(k TrialKey, c string) TrialRecord {
		t.Helper()
		tr, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, k, CopyBack{Receipt: hostReceipt(t, map[string]any{"score": map[string]int{"passed": 1, "total": 1}}),
			Transcript: []byte(`{"type":"tool_use","name":"bash","input":{"command":"cd ` + c + ` && cat ../*/README.md"}}` + "\n")})
		if err != nil {
			t.Fatalf("CollectCopy %s = %v", k, err)
		}
		return tr
	}
	var held TrialRecord
	if err := fsutil.WithFileLock(launchLockPath(handoffStore(home, rec.RootSHA), rec.RunID), 0, func() error {
		held = back(host, hostClone)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	modes("after a hand-back while a launch holds the lock", listingDenied)
	if held.Compliance == nil {
		t.Fatalf("the hand-back carries no compliance check: %+v", held)
	}
	if got := ruleOf(t, *held.Compliance, RuleOwnCopyOnly); got.State != StateUnexamined {
		t.Fatalf("a host copy's glob over a folder the check cannot list: own_copy_only %s (%s); want unexamined", got.State, got.Detail)
	}
	back(host2, hostClone2)
	modes("after a hand-back that found the lock free", 0o700)
}

// TestACloneIsDeletedFromAFolderClosedToListing: a copy's clone is deleted
// whole while its run's folder is closed to listing, as a launch holds it,
// leaving nothing beside it; a link in the clone's place is removed, never
// followed.
func TestACloneIsDeletedFromAFolderClosedToListing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the superuser lists a folder whatever its mode")
	}
	run := filepath.Join(t.TempDir(), "run")
	clone := filepath.Join(run, "copy_a")
	for _, d := range []string{filepath.Join(clone, ".git", "objects"), filepath.Join(clone, "widget")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(clone, ".git", "objects", "ab"), []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	kept := t.TempDir()
	if err := os.WriteFile(filepath.Join(kept, "keep.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(kept, filepath.Join(run, "copy_b")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(run, listingDenied); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(run, 0o700) })
	for _, dir := range []string{clone, filepath.Join(run, "copy_b")} {
		if err := removeClone(dir); err != nil {
			t.Fatalf("removeClone(%s) with its folder closed: %v", filepath.Base(dir), err)
		}
		if _, err := os.Lstat(dir); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("%s is still there: %v", filepath.Base(dir), err)
		}
	}
	if _, err := os.Stat(filepath.Join(kept, "keep.txt")); err != nil {
		t.Fatalf("a link's target was followed: %v", err)
	}
	if err := os.Chmod(run, 0o700); err != nil {
		t.Fatal(err)
	}
	if ents, err := os.ReadDir(run); err != nil || len(ents) != 0 {
		t.Fatalf("the run's folder holds %v (%v); want nothing", ents, err)
	}
}
