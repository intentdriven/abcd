package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// fakeToken is a provider-shaped key value, built when the test runs.
func fakeToken() string { return "sk-ant-api03-" + testsecret.Synthetic(7, 40) }

// entryBytes is what an entry's file holds, byte for byte.
func entryBytes(t *testing.T, home string, e LedgerEntry) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(abcdhome.Path(home, ledgerFolder), e.ID+ledgerExt))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// A failure's words can quote what it failed on. Whatever an error carries, a
// key's value never reaches an entry, and the reason still says what happened.
func TestAFailureReasonCarriesNoSecret(t *testing.T) {
	token := fakeToken()
	for name, failure := range map[string]error{
		"a plain error": fmt.Errorf("the store refused a span holding %s", token),
		"a failure":     fail(OpenCode, ReasonFailed, "the local server at http://localhost/?key=%s did not answer", token),
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			if _, err := RecordLaunch(context.Background(), NewLedger(home), ledgerStart(), func(context.Context) LaunchEnd {
				return LaunchEnd{Err: failure}
			}); err != nil {
				t.Fatal(err)
			}
			got := ledgerFiles(t, home)
			if len(got) != 1 {
				t.Fatalf("entries = %+v", got)
			}
			if raw := entryBytes(t, home, got[0]); strings.Contains(raw, token) || strings.Contains(raw, token[:24]) {
				t.Errorf("the entry carries the key: %s", raw)
			}
			if got[0].Reason == "" {
				t.Error("the reason was lost with the key")
			}
		})
	}
}

// A validator's refusal quotes the answer it read, which the agent wrote and
// which can repeat its brief. An entry says the validator refused and no more.
func TestAValidatorsWordsNeverReachAnEntry(t *testing.T) {
	f := newFake(t, "stream", Claude)
	answering(t, streamInit+streamWindow+streamResult)
	c := mustLoad(t, claudeRoute, "")
	h := newHarness(t, c, true)
	const brief = "the secret plan the brief described"
	token := fakeToken()
	h.d.Validate = func(string, Request, Answer) error {
		return fmt.Errorf("the receipt says %q and %s", brief, token)
	}
	if _, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer")); err != nil {
		t.Fatal(err)
	}
	e := oneEntry(t, c)
	raw := entryBytes(t, c.Ledger().home, e)
	if strings.Contains(raw, brief) || strings.Contains(raw, token) {
		t.Errorf("the entry carries the agent's words: %s", raw)
	}
	if e.Outcome != OutcomeFailed || !strings.Contains(e.Reason, "validator refused") {
		t.Errorf("entry = %+v", e)
	}
}

// What the harness said, the agent's answer and the launch's environment stay
// out of an entry on a launch that succeeds as on one that fails.
func TestASuccessfulLaunchKeepsTheAnswerOutOfItsEntry(t *testing.T) {
	f := newFake(t, "stream", Claude)
	token := fakeToken()
	answering(t, streamInit+streamWindow+
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"the plan is `+token+`"}]},"session_id":"s1"}`+"\n"+
		`{"type":"result","subtype":"success","is_error":false,"result":"the answer is `+token+`","session_id":"s1","total_cost_usd":0.75}`+"\n")
	t.Setenv(fakeCredEnv, token)
	c := mustLoad(t, claudeRoute, "")
	if _, err := newHarness(t, c, true).d.Dispatch(context.Background(), f.request("ruthless-reviewer")); err != nil {
		t.Fatal(err)
	}
	if raw := entryBytes(t, c.Ledger().home, oneEntry(t, c)); strings.Contains(raw, token) || strings.Contains(raw, "the plan") {
		t.Errorf("the entry carries the answer: %s", raw)
	}
}

// The entry's shape is the promise that no prompt, answer or credential has a
// place in it: a field added here is added on purpose, past this list.
func TestAnEntryHasNoFieldForAPromptOrACredential(t *testing.T) {
	now := time.Now()
	raw, err := json.Marshal(LedgerEntry{Schema: 1, ID: "i", Orchestrator: Orchestrator{Kind: "k", ID: "i", Name: "n"}, Repo: "r", Run: "r", Lane: "l", Await: "a",
		Role: "r", Runner: "r", ModelAsked: "m", Route: "r", Key: "k", RouteWhy: "w", Model: "m", Usage: &oracle.Usage{}, Cost: &oracle.Cost{},
		APIMillis: ms(1), WallMillis: 1, Outcome: "o", Reason: "r", StartedAt: now, EndedAt: &now, PID: 1, PIDStartUnix: 1})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	var got []string
	for k := range fields {
		got = append(got, k)
	}
	slices.Sort(got)
	want := []string{"api_ms", "await", "cost", "ended_at", "id", "key", "lane", "model", "model_asked", "orchestrator", "outcome", "pid", "pid_start_unix",
		"reason", "repo", "role", "route", "route_why", "run", "runner", "schema_version", "started_at", "usage", "wall_ms"}
	if !slices.Equal(got, want) {
		t.Errorf("an entry's fields = %v, want %v", got, want)
	}
}

// A key is named by the credential store's own name. Anything shaped like a
// value refuses the launch, whichever provider's value it is.
func TestAKeyShapedLikeAValueIsRefused(t *testing.T) {
	values := map[string]string{
		"lower-case hex":      testsecret.SyntheticHex(11, 40),
		"short hex":           testsecret.SyntheticHex(12, 32),
		"mixed case":          testsecret.Synthetic(13, 39),
		"a slack shaped word": "xoxb-" + testsecret.SyntheticHex(14, 24),
		"a provider prefix":   "sk-" + testsecret.SyntheticHex(15, 20),
		"a key behind a word": "work-" + testsecret.SyntheticHex(16, 32),
		"upper case":          "BUILD-KEY",
	}
	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			start := ledgerStart()
			start.Key = value
			_, err := RecordLaunch(context.Background(), NewLedger(home), start, func(context.Context) LaunchEnd {
				t.Errorf("launched with %q", value)
				return LaunchEnd{}
			})
			if err == nil || strings.Contains(err.Error(), value) {
				t.Errorf("err = %v, want a refusal that does not repeat it", err)
			}
			if got := ledgerFiles(t, home); len(got) != 0 {
				t.Errorf("entries = %+v", got)
			}
		})
	}
	for _, name := range strings.Fields("build-key anthropic.main-2 team_a nightly-2026-11 k") {
		t.Run(name, func(t *testing.T) {
			start := ledgerStart()
			start.Key = name
			if _, err := RecordLaunch(context.Background(), NewLedger(t.TempDir()), start, func(context.Context) LaunchEnd { return LaunchEnd{} }); err != nil {
				t.Errorf("a plain name was refused: %v", err)
			}
		})
	}
}

// The store holds every repository's launches, so one repository's file cannot
// shorten what this machine keeps: a figure below what the machine holds is
// named and ignored, and a figure above it stands.
func TestARepositoryCannotShortenWhatTheMachineKeeps(t *testing.T) {
	c := mustLoad(t, `{"ledger":{"retention_days":400}}`, `{"ledger":{"retention_days":40,"retention_count":1}}`)
	if r := c.Ledger().retention; r != (Retention{Days: 400, Count: DefaultRetentionCount}) {
		t.Errorf("retention = %+v", r)
	}
	for _, key := range []string{"ledger.retention_days", "ledger.retention_count"} {
		if !slices.ContainsFunc(c.Diagnostics, func(d string) bool { return strings.Contains(d, key) && strings.Contains(d, "ignored") }) {
			t.Errorf("%s is not named as ignored: %q", key, c.Diagnostics)
		}
	}
	if r := mustLoad(t, "", `{"ledger":{"retention_count":1}}`).Ledger().retention; r.Count != DefaultRetentionCount {
		t.Errorf("a repository lowered the shipped count: %+v", r)
	}
	if r := mustLoad(t, `{"ledger":{"retention_days":90,"retention_count":500}}`, `{"ledger":{"retention_days":120,"retention_count":900}}`).Ledger().retention; r != (Retention{Days: 120, Count: 900}) {
		t.Errorf("a longer figure did not stand: %+v", r)
	}
	if r := mustLoad(t, `{"ledger":{"retention_days":40,"retention_count":7}}`, "").Ledger().retention; r != (Retention{Days: 40, Count: 7}) {
		t.Errorf("the machine's own shorter figure did not stand: %+v", r)
	}
}

// What the attack would have cost: another repository's old entries survive a
// launch from the repository whose file asks for one entry.
func TestAHostileRepositorysFigurePrunesNothingTheMachineKeeps(t *testing.T) {
	c := mustLoad(t, `{"ledger":{"retention_days":400}}`, `{"ledger":{"retention_count":1}}`)
	l, names := agedEntries(t, c.Ledger().retention, 40, 100, 300)
	if got, want := survivors(t, l, names), []int{40, 100, 300}; !slices.Equal(got, want) {
		t.Errorf("held %v, want %v", got, want)
	}
}

// A figure past any sensible store is refused: it would overflow the date
// arithmetic, and the ledger never grows without bound.
func TestARetentionFigurePastItsBoundIsRefused(t *testing.T) {
	for _, bad := range []string{
		`{"ledger":{"retention_days":36501}}`,
		`{"ledger":{"retention_days":9223372036854775807}}`,
		`{"ledger":{"retention_count":1000001}}`,
	} {
		if _, err := Load(roots(t, "", bad)); err == nil || !strings.Contains(err.Error(), "ledger") {
			t.Errorf("%s: err = %v", bad, err)
		}
	}
}

// A launcher that died before the harness started leaves an entry judged on the
// launcher's id. Once that id is held by a process that started after the entry
// was written, it is someone else's, and the entry is lost.
func TestAnEntryOnAnIdReusedSinceItsLaunchIsLost(t *testing.T) {
	home := t.TempDir()
	layEntry(t, home, LedgerEntry{Run: "reused", Outcome: OutcomeRunning, PID: os.Getpid(), StartedAt: time.Now().Add(-48 * time.Hour)})
	layEntry(t, home, LedgerEntry{Run: "ours", Outcome: OutcomeRunning, PID: os.Getpid(), StartedAt: time.Now().Add(time.Second)})
	want := map[string]string{"reused": OutcomeLost, "ours": OutcomeRunning}
	for _, e := range readAll(t, home, LedgerQuery{All: true}).Entries {
		if e.Outcome != want[e.Run] {
			t.Errorf("%s reads %s, want %s", e.Run, e.Outcome, want[e.Run])
		}
	}
}

// A machine that dies mid-write leaves the atomic writer's temporary file, which
// no launch will ever finish. The next writer clears one that old, and leaves
// one a racing launch may still be writing.
func TestTemporaryFilesACrashLeftArePruned(t *testing.T) {
	l, names := agedEntries(t, DefaultRetention(), 1)
	dir := abcdhome.Path(l.home, ledgerFolder)
	stale, fresh := filepath.Join(dir, tempFilePrefix+"aaaa"), filepath.Join(dir, tempFilePrefix+"bbbb")
	for _, p := range []string{stale, fresh} {
		if err := os.WriteFile(p, []byte("half a wri"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := retentionNow.Add(-48 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	recent := retentionNow.Add(-time.Minute)
	if err := os.Chtimes(fresh, recent, recent); err != nil {
		t.Fatal(err)
	}
	survivors(t, l, names)
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the crash's temporary file stayed: %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("a racing launch's temporary file went: %v", err)
	}
}

// Pruning races a read: a file listed and gone before it is opened was pruned,
// which is no fault to name.
func TestAFilePrunedDuringAReadIsNotNamed(t *testing.T) {
	home := t.TempDir()
	layRaw(t, home, "lch-261104120000000000-aaaaaaaa.json", `{"schema_version":1}`+"\n")
	root, err := os.OpenRoot(abcdhome.Path(home, ledgerFolder))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	var out LedgerListing
	if _, ok := readLedgerFile(root, "lch-261104120000000001-bbbbbbbb.json", &out); ok || len(out.Unreadable) != 0 {
		t.Errorf("ok %v, unreadable %+v", ok, out.Unreadable)
	}
}

// JSON that is not an entry is named, not counted as a launch with no figures.
func TestAFileWithNoVersionIsNotAnEntry(t *testing.T) {
	for _, body := range []string{"null\n", "{}\n", "[]\n", "123\n", `{"schema_version":"1"}` + "\n", ""} {
		home := t.TempDir()
		layRaw(t, home, "lch-261104120000000000-aaaaaaaa.json", body)
		got := readAll(t, home, LedgerQuery{All: true})
		if len(got.Entries) != 0 || len(got.Unreadable) != 1 {
			t.Errorf("%q: entries %+v, unreadable %+v", body, got.Entries, got.Unreadable)
		}
	}
}

// Criterion 4: a later abcd may change a field's shape, and a reader refuses
// nothing for that beyond naming the entry as newer. It takes the rest.
func TestANewerEntryWithAFieldOfAnotherShapeIsStillRead(t *testing.T) {
	home := t.TempDir()
	layRaw(t, home, "lch-261104120000000000-aaaaaaaa.json",
		`{"schema_version":2,"route":"key","key":"a","outcome":"finished","usage":"its shape changed","cost":{"amount":1,"unit":"usd"}}`+"\n")
	got := readAll(t, home, LedgerQuery{All: true})
	if len(got.Entries) != 1 || len(got.Newer) != 1 || len(got.Unreadable) != 0 || got.Totals.SpendByKey["a"] != 1 {
		t.Errorf("listing = %+v", got)
	}
}

// A torn file and an oversized one are named on every read and left alone, and
// pruning takes them by name when their time comes, so neither stays for ever.
func TestATornOrOversizedFileIsNamedThenPrunedByName(t *testing.T) {
	l, names := agedEntries(t, DefaultRetention(), 90, 91, 1)
	dir := abcdhome.Path(l.home, ledgerFolder)
	if err := os.WriteFile(filepath.Join(dir, names[90]), []byte(`{"schema_version":1,"rou`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, names[91]), []byte(strings.Repeat("x", maxLedgerEntry+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	got := readAll(t, l.home, LedgerQuery{All: true})
	if len(got.Unreadable) != 2 || len(got.Entries) != 1 {
		t.Fatalf("listing = %+v", got)
	}
	if held := survivors(t, l, names); !slices.Equal(held, []int{1}) {
		t.Errorf("held %v, want the torn and the oversized gone by age", held)
	}
}

// A directory or a link wearing an entry's name is never followed: the pruner
// removes the name, and what a link points at is untouched.
func TestPruningALinkRemovesTheLinkAlone(t *testing.T) {
	l, names := agedEntries(t, DefaultRetention(), 90)
	dir := abcdhome.Path(l.home, ledgerFolder)
	target := filepath.Join(t.TempDir(), "precious")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, names[90])); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, names[90])); err != nil {
		t.Fatal(err)
	}
	survivors(t, l, names)
	if _, err := os.Lstat(filepath.Join(dir, names[90])); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the link stayed: %v", err)
	}
	if raw, err := os.ReadFile(target); err != nil || string(raw) != "keep" {
		t.Errorf("the link's target was touched: %q, %v", raw, err)
	}
}

// An entry the reader could not read back records nothing, so a launch whose
// entry would pass the reader's bound is refused before it starts.
func TestAnEntryTooLargeToReadBackRefusesTheLaunch(t *testing.T) {
	home := t.TempDir()
	start := ledgerStart()
	start.Repo = "/" + strings.Repeat("r", maxLedgerEntry)
	_, err := RecordLaunch(context.Background(), NewLedger(home), start, func(context.Context) LaunchEnd {
		t.Error("the launch ran with an entry no reader can read")
		return LaunchEnd{}
	})
	if err == nil || !strings.Contains(err.Error(), "ledger") {
		t.Fatalf("err = %v", err)
	}
	if got := readAll(t, home, LedgerQuery{All: true}); len(got.Unreadable) != 0 || len(got.Entries) != 0 {
		t.Errorf("listing = %+v", got)
	}
}

// Launches racing on a store nobody has made yet, with a reader alongside and
// the pruner at its tightest: each keeps its own whole file, and none is lost.
func TestLaunchesRacingOnAFreshStoreEachKeepTheirOwnFile(t *testing.T) {
	home := t.TempDir()
	const n = 24
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l := NewLedger(home)
			l.retention = Retention{Days: 1, Count: 1}
			start := ledgerStart()
			start.Work.Await = fmt.Sprintf("receipt-%d", i)
			_, err := RecordLaunch(context.Background(), l, start, func(context.Context) LaunchEnd {
				_, _ = ReadLedger(home, LedgerQuery{All: true})
				return LaunchEnd{Answer: Answer{Model: fmt.Sprintf("model-%d", i)}}
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	got := readAll(t, home, LedgerQuery{All: true})
	if len(got.Entries) != n || len(got.Unreadable) != 0 || got.Totals.Finished != n {
		t.Fatalf("listing: %d entries, %d finished, unreadable %+v", len(got.Entries), got.Totals.Finished, got.Unreadable)
	}
	seen := map[string]bool{}
	for _, e := range got.Entries {
		if want := "model-" + strings.TrimPrefix(e.Await, "receipt-"); e.Model != want {
			t.Errorf("entry for %s holds another launch's figures: %s", e.Await, e.Model)
		}
		seen[e.Await] = true
	}
	if len(seen) != n {
		t.Errorf("awaits = %d, want %d", len(seen), n)
	}
}

// While the harness runs the entry is judged by the harness's process, with the
// start that tells it from a later process given the same id.
func TestAStartedLaunchIsJudgedOnItsHarness(t *testing.T) {
	home := t.TempDir()
	if _, err := RecordLaunch(context.Background(), NewLedger(home), ledgerStart(), func(ctx context.Context) LaunchEnd {
		LaunchStarted(ctx, 4242)
		e := ledgerFiles(t, home)[0]
		if e.PID != 4242 || e.PIDStartUnix == 0 {
			t.Errorf("after the start: %+v", e)
		}
		LaunchExited(ctx)
		e = ledgerFiles(t, home)[0]
		if e.PID != os.Getpid() || e.PIDStartUnix != 0 {
			t.Errorf("after the exit: %+v", e)
		}
		return LaunchEnd{}
	}); err != nil {
		t.Fatal(err)
	}
}

// A launcher that dies with its harness running leaves the entry it wrote, on
// the harness's process: running while that lives and lost once it is gone.
func TestALaunchDyingBeforeItsEndIsRecordedRunningThenLost(t *testing.T) {
	home := t.TempDir()
	l := NewLedger(home)
	h, err := l.begin(ledgerStart())
	if err != nil {
		t.Fatal(err)
	}
	// The launcher, this process, wrote the entry and nothing else: dying here
	// is what the record has to survive. It reads as running under the live
	// launcher.
	if got := readAll(t, home, LedgerQuery{All: true}); got.Totals.Running != 1 {
		t.Fatalf("totals = %+v", got.Totals)
	}
	h.mu.Lock()
	h.entry.PID, h.entry.PIDStartUnix = 999999, time.Now().Unix()
	h.mu.Unlock()
	if err := h.write(); err != nil {
		t.Fatal(err)
	}
	if got := readAll(t, home, LedgerQuery{All: true}); got.Totals.Lost != 1 || got.Totals.Running != 0 {
		t.Fatalf("totals = %+v, want the launch whose process is gone lost", got.Totals)
	}
}
