package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// ledgerStart is the start every ledger test hands RecordLaunch.
func ledgerStart() LaunchStart {
	return LaunchStart{Role: "implementer", Runner: Claude, ModelAsked: "anthropic/fake", Repo: "/repo",
		Work: LaunchWork{Run: "run-1", Lane: "lane-1", Await: "receipt.json"}}
}

// ledgerFiles reads every entry file in home's ledger, each held to one line.
func ledgerFiles(t *testing.T, home string) []LedgerEntry {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(abcdhome.Path(home, ledgerFolder), "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []LedgerEntry
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Count(raw, []byte("\n")) != 1 || !bytes.HasSuffix(raw, []byte("\n")) {
			t.Fatalf("%s holds %q, want one line", p, raw)
		}
		var e LedgerEntry
		if err := json.Unmarshal(raw, &e); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}

// TestRecordLaunchWritesTheEntryBeforeTheLaunchAndCompletesItAfter is the
// first criterion's shape: while the launch runs one entry names it, marked
// running; when it ends the same entry carries the figures the harness
// reported, and nothing else was written.
func TestRecordLaunchWritesTheEntryBeforeTheLaunchAndCompletesItAfter(t *testing.T) {
	home := t.TempDir()
	l := newLedger(home)
	cost := &oracle.Cost{Amount: 1.25, Unit: oracle.CostUSD}
	usage := &oracle.Usage{PromptTokens: 10, CompletionTokens: 20, CacheReadTokens: 30, CacheWriteTokens: 40}
	ran := false
	end, err := RecordLaunch(context.Background(), l, ledgerStart(), func(context.Context) LaunchEnd {
		ran = true
		mid := ledgerFiles(t, home)
		if len(mid) != 1 {
			t.Fatalf("entries while running = %d, want 1", len(mid))
		}
		e := mid[0]
		if e.Outcome != OutcomeRunning || e.EndedAt != nil {
			t.Errorf("while running: outcome %q, ended %v", e.Outcome, e.EndedAt)
		}
		if e.Role != "implementer" || e.Runner != Claude || e.ModelAsked != "anthropic/fake" || e.Repo != "/repo" ||
			e.Run != "run-1" || e.Lane != "lane-1" || e.Await != "receipt.json" {
			t.Errorf("while running: %+v", e)
		}
		if e.Schema != LedgerSchema {
			t.Errorf("schema = %d", e.Schema)
		}
		if e.Route != RouteUnknown {
			t.Errorf("route while running = %q, want unknown until the stream shows one", e.Route)
		}
		var ans Answer
		ans.Model, ans.Usage, ans.Cost = "fake-model", usage, cost
		return LaunchEnd{Answer: ans}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ran || end.Answer.Model != "fake-model" {
		t.Fatalf("ran %v, end %+v", ran, end)
	}
	got := ledgerFiles(t, home)
	if len(got) != 1 {
		t.Fatalf("entries = %d, want the one completed", len(got))
	}
	e := got[0]
	if e.Outcome != OutcomeFinished || e.EndedAt == nil || e.Model != "fake-model" {
		t.Fatalf("completed: %+v", e)
	}
	if e.Usage == nil || *e.Usage != *usage || e.Cost == nil || *e.Cost != *cost {
		t.Fatalf("figures: usage %+v, cost %+v", e.Usage, e.Cost)
	}
	if e.Orchestrator.ID == "" {
		t.Fatal("an entry with no orchestrator")
	}
}

// A failure is recorded with its reason, and a killed launch as stopped with
// what its transcript showed it spent; neither failure is the ledger's to
// swallow, so the caller still sees it.
func TestRecordLaunchRecordsAFailureAndAKill(t *testing.T) {
	home := t.TempDir()
	l := newLedger(home)
	failed := fail(Claude, ReasonFailed, "claude exited with status 1")
	end, err := RecordLaunch(context.Background(), l, ledgerStart(), func(context.Context) LaunchEnd {
		return LaunchEnd{Err: failed}
	})
	if err != nil || !errors.Is(end.Err, failed) {
		t.Fatalf("err %v, end.Err %v", err, end.Err)
	}
	killed := failAt(ErrKilled, Claude, ReasonFailed, "claude did not finish within the time allowed")
	killed.Spent = &Spend{Usage: &oracle.Usage{PromptTokens: 7}, Cost: &oracle.Cost{Amount: 0.5, Unit: oracle.CostUSD}}
	if _, err := RecordLaunch(context.Background(), l, ledgerStart(), func(context.Context) LaunchEnd {
		return LaunchEnd{Err: killed}
	}); err != nil {
		t.Fatal(err)
	}
	var sawFailed, sawStopped bool
	for _, e := range ledgerFiles(t, home) {
		switch e.Outcome {
		case OutcomeFailed:
			sawFailed = strings.Contains(e.Reason, "status 1")
		case OutcomeStopped:
			sawStopped = strings.Contains(e.Reason, "time allowed") && e.Usage != nil && e.Usage.PromptTokens == 7 &&
				e.Cost != nil && e.Cost.Amount == 0.5
		}
	}
	if !sawFailed || !sawStopped {
		t.Fatalf("failed %v, stopped %v: %+v", sawFailed, sawStopped, ledgerFiles(t, home))
	}
}

// Decision 21: a role handed to the host gets a placeholder naming the host
// as the route, with no figures, so a run's listing is complete.
func TestRecordLaunchWithNoLaunchWritesTheHostsPlaceholder(t *testing.T) {
	home := t.TempDir()
	start := ledgerStart()
	start.Runner = Host
	if _, err := RecordLaunch(context.Background(), newLedger(home), start, nil); err != nil {
		t.Fatal(err)
	}
	got := ledgerFiles(t, home)
	if len(got) != 1 {
		t.Fatalf("entries = %d", len(got))
	}
	e := got[0]
	if e.Route != RouteHost || e.Outcome != OutcomeHanded || e.Usage != nil || e.Cost != nil || e.EndedAt == nil {
		t.Fatalf("placeholder = %+v", e)
	}
}

// The mechanism's claim is that no launch escapes the record, so an entry
// that cannot be written starts nothing, and the refusal names the store.
func TestRecordLaunchRefusesALaunchItCannotRecord(t *testing.T) {
	blocked := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(blocked), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(blocked, ledgerFolder), []byte("not a folder"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, l := range map[string]*Ledger{"no ledger": nil, "no home": newLedger(""), "no folder": newLedger(blocked)} {
		t.Run(name, func(t *testing.T) {
			_, err := RecordLaunch(context.Background(), l, ledgerStart(), func(context.Context) LaunchEnd {
				t.Error("the launch ran with no entry written")
				return LaunchEnd{}
			})
			if err == nil || !strings.Contains(err.Error(), "ledger") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

// A completion that cannot be written leaves the work standing: the answer
// reaches the caller, and the entry stays as it was, for a reader to judge.
func TestRecordLaunchKeepsTheAnswerWhenItCannotCompleteTheEntry(t *testing.T) {
	home := t.TempDir()
	dir := abcdhome.Path(home, ledgerFolder)
	end, err := RecordLaunch(context.Background(), newLedger(home), ledgerStart(), func(context.Context) LaunchEnd {
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
		return LaunchEnd{Answer: Answer{Model: "fake-model"}}
	})
	if err != nil || end.Answer.Model != "fake-model" {
		t.Fatalf("err %v, end %+v", err, end)
	}
	got := ledgerFiles(t, home)
	if len(got) != 1 || got[0].Outcome != OutcomeRunning {
		t.Fatalf("entries = %+v", got)
	}
}

// A key is named, never carried: a name shaped like a value refuses the launch
// before anything is written, so a mistaken caller cannot put one in the store.
func TestRecordLaunchRefusesAKeyNameThatIsNotAName(t *testing.T) {
	home := t.TempDir()
	for _, bad := range []string{"sk-ant-api03-abcdef", strings.Repeat("a", 200), "two words", "a/b"} {
		start := ledgerStart()
		start.Key = bad
		_, err := RecordLaunch(context.Background(), newLedger(home), start, func(context.Context) LaunchEnd {
			t.Errorf("launched with key name %q", bad)
			return LaunchEnd{}
		})
		if err == nil || strings.Contains(err.Error(), bad) {
			t.Errorf("key %q: err = %v, want a refusal that does not repeat it", bad, err)
		}
	}
	if got := ledgerFiles(t, home); len(got) != 0 {
		t.Fatalf("entries = %+v", got)
	}
	start := ledgerStart()
	start.Key = "build-key"
	if _, err := RecordLaunch(context.Background(), newLedger(home), start, func(context.Context) LaunchEnd {
		return LaunchEnd{}
	}); err != nil {
		t.Fatal(err)
	}
	got := ledgerFiles(t, home)
	if len(got) != 1 || got[0].Route != RouteKey || got[0].Key != "build-key" {
		t.Fatalf("entries = %+v", got)
	}
}

// A store folder and its files are the person's alone.
func TestLedgerStoreIsPrivate(t *testing.T) {
	home := t.TempDir()
	if _, err := RecordLaunch(context.Background(), newLedger(home), ledgerStart(), func(context.Context) LaunchEnd {
		return LaunchEnd{}
	}); err != nil {
		t.Fatal(err)
	}
	dir := abcdhome.Path(home, ledgerFolder)
	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != abcdhome.DirMode {
		t.Errorf("folder mode = %o", fi.Mode().Perm())
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(paths) != 1 {
		t.Fatalf("files = %v", paths)
	}
	fi, err = os.Stat(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != abcdhome.FileMode {
		t.Errorf("file mode = %o", fi.Mode().Perm())
	}
}

// The second criterion: the exported session by id and persona name, a session
// the process descends from, and with neither the terminal run, so no entry is
// unattributed.
func TestAnEntryNamesItsOrchestrator(t *testing.T) {
	home := t.TempDir()
	reg := filepath.Join(abcdhome.Path(home, "sessions"), "names.json")
	if err := os.MkdirAll(filepath.Dir(reg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reg, []byte(`{"schema_version":1,"live":[{"name":"Alice","session_id":"sess-1","host_pid":1,"host_start_unix":1,"sent":true}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	record := func() LedgerEntry {
		t.Helper()
		before := map[string]bool{}
		for _, e := range ledgerFiles(t, home) {
			before[e.ID] = true
		}
		if _, err := RecordLaunch(context.Background(), newLedger(home), ledgerStart(), func(context.Context) LaunchEnd {
			return LaunchEnd{}
		}); err != nil {
			t.Fatal(err)
		}
		for _, e := range ledgerFiles(t, home) {
			if !before[e.ID] {
				return e
			}
		}
		t.Fatal("no new entry")
		return LedgerEntry{}
	}
	t.Setenv(HostSessionEnv, "sess-1")
	if got := record().Orchestrator; got != (Orchestrator{Kind: OrchestratorSession, ID: "sess-1", Name: "Alice"}) {
		t.Errorf("exported session = %+v", got)
	}
	t.Setenv(HostSessionEnv, "sess-unknown")
	if got := record().Orchestrator; got != (Orchestrator{Kind: OrchestratorSession, ID: "sess-unknown"}) {
		t.Errorf("a session the register does not hold = %+v", got)
	}
	t.Setenv(HostSessionEnv, "not a session id")
	if got := record().Orchestrator; got != (Orchestrator{Kind: OrchestratorRun, ID: "run-1"}) {
		t.Errorf("no usable session = %+v", got)
	}
	t.Setenv(HostSessionEnv, "")
	start := ledgerStart()
	start.Work.Run, start.Fallback = "", "interview-1-turn-2"
	if _, err := RecordLaunch(context.Background(), newLedger(t.TempDir()), start, func(context.Context) LaunchEnd {
		return LaunchEnd{}
	}); err != nil {
		t.Fatal(err)
	}
}
