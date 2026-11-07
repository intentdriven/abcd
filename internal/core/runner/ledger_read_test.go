package runner

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// layEntry writes e as its file in home's ledger, an id minted when it has none.
func layEntry(t *testing.T, home string, e LedgerEntry) LedgerEntry {
	t.Helper()
	if e.ID == "" {
		id, err := newLaunchID(time.Now())
		if err != nil {
			t.Fatal(err)
		}
		e.ID = id
	}
	if e.Schema == 0 {
		e.Schema = LedgerSchema
	}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	layRaw(t, home, e.ID+ledgerExt, string(raw)+"\n")
	return e
}

func layRaw(t *testing.T, home, name, body string) {
	t.Helper()
	dir := abcdhome.Path(home, ledgerFolder)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func usd(a float64) *oracle.Cost { return &oracle.Cost{Amount: a, Unit: oracle.CostUSD} }

func ms(n int64) *int64 { return &n }

func readAll(t *testing.T, home string, q LedgerQuery) LedgerListing {
	t.Helper()
	got, err := ReadLedger(home, q)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// Decision 1: only a key's cost is spend. A subscription's and an unknown
// route's are kept apart as estimates, and tokens and API time sum everything.
func TestTotalsCountMoneyForKeysAlone(t *testing.T) {
	home := t.TempDir()
	u := &oracle.Usage{PromptTokens: 1, CompletionTokens: 2, ReasoningTokens: 100, CacheReadTokens: 3, CacheWriteTokens: 4}
	layEntry(t, home, LedgerEntry{Route: RouteKey, Key: "a", Outcome: OutcomeFinished, Cost: usd(2), Usage: u, APIMillis: ms(10)})
	layEntry(t, home, LedgerEntry{Route: RouteKey, Key: "a", Outcome: OutcomeFailed, Cost: usd(1), Usage: u, APIMillis: ms(10)})
	layEntry(t, home, LedgerEntry{Route: RouteKey, Key: "b", Outcome: OutcomeStopped, Cost: usd(4)})
	layEntry(t, home, LedgerEntry{Route: RouteSubscription, Outcome: OutcomeFinished, Cost: usd(8), Usage: u})
	layEntry(t, home, LedgerEntry{Route: RouteUnknown, Outcome: OutcomeFinished, Cost: usd(16)})
	layEntry(t, home, LedgerEntry{Route: RouteHost, Outcome: OutcomeHanded})
	layEntry(t, home, LedgerEntry{Route: RouteKey, Key: "a", Outcome: OutcomeFinished, Cost: &oracle.Cost{Amount: 32, Unit: oracle.CostCredits}})
	got := readAll(t, home, LedgerQuery{All: true}).Totals
	if got.Launches != 7 || got.Finished != 4 || got.Failed != 1 || got.Stopped != 1 || got.Handed != 1 {
		t.Errorf("counts = %+v", got)
	}
	if got.SpendByKey["a"] != 3 || got.SpendByKey["b"] != 4 || len(got.SpendByKey) != 2 {
		t.Errorf("spend = %v", got.SpendByKey)
	}
	if got.EstimateUSD != 24 {
		t.Errorf("estimate = %v", got.EstimateUSD)
	}
	if got.Tokens != 30 {
		t.Errorf("tokens = %d, want the stated basis: prompt, completion, cache read and cache write", got.Tokens)
	}
	if got.APIMillis != 20 {
		t.Errorf("api ms = %d", got.APIMillis)
	}
}

// A planted or broken figure counts in no sum and overflows none.
func TestTotalsIgnoreFiguresNoLaunchReports(t *testing.T) {
	home := t.TempDir()
	layEntry(t, home, LedgerEntry{Route: RouteKey, Key: "a", Outcome: OutcomeFinished, Cost: usd(-5),
		Usage: &oracle.Usage{PromptTokens: -9}, APIMillis: ms(-1)})
	layEntry(t, home, LedgerEntry{Route: RouteKey, Key: "a", Outcome: OutcomeFinished, Usage: &oracle.Usage{PromptTokens: math.MaxInt}, APIMillis: ms(math.MaxInt64)})
	layEntry(t, home, LedgerEntry{Route: RouteKey, Key: "a", Outcome: OutcomeFinished, Usage: &oracle.Usage{PromptTokens: math.MaxInt}, APIMillis: ms(math.MaxInt64)})
	got := readAll(t, home, LedgerQuery{All: true}).Totals
	if got.SpendByKey["a"] != 0 || got.Tokens != math.MaxInt64 || got.APIMillis != math.MaxInt64 {
		t.Errorf("totals = %+v", got)
	}
}

// The filters: a session's own, a run's whoever drove it, and everything.
func TestReadingSelectsBySessionRunOrAll(t *testing.T) {
	home := t.TempDir()
	mine := Orchestrator{Kind: OrchestratorSession, ID: "sess-1"}
	layEntry(t, home, LedgerEntry{Orchestrator: mine, Run: "run-1", Route: RouteHost, Outcome: OutcomeHanded})
	layEntry(t, home, LedgerEntry{Orchestrator: Orchestrator{Kind: OrchestratorSession, ID: "sess-2"}, Run: "run-1", Route: RouteHost, Outcome: OutcomeHanded})
	layEntry(t, home, LedgerEntry{Orchestrator: Orchestrator{Kind: OrchestratorRun, ID: "run-2"}, Route: RouteHost, Outcome: OutcomeHanded})
	for _, tc := range []struct {
		q    LedgerQuery
		want int
	}{
		{LedgerQuery{Session: "sess-1"}, 1},
		{LedgerQuery{Run: "run-1"}, 2},
		{LedgerQuery{Run: "run-2"}, 1},
		{LedgerQuery{All: true}, 3},
		{LedgerQuery{Session: "nobody"}, 0},
	} {
		if got := readAll(t, home, tc.q); len(got.Entries) != tc.want {
			t.Errorf("%+v: %d entries, want %d", tc.q, len(got.Entries), tc.want)
		}
	}
}

// Criterion 4: a reader takes what it knows, passes over what it does not, and
// names an entry above its own version rather than refusing it; an entry it
// cannot read is named, never silently dropped; and it writes nothing.
func TestAReaderIsLenientAndNamesWhatItCannotRead(t *testing.T) {
	home := t.TempDir()
	layRaw(t, home, "lch-261104120000000000-aaaaaaaa.json",
		`{"schema_version":9,"id":"lch-261104120000000000-aaaaaaaa","route":"key","key":"a","outcome":"finished","cost":{"amount":1,"unit":"usd"},"a_field_from_later":{"x":1}}`+"\n")
	layRaw(t, home, "lch-261104120000000001-bbbbbbbb.json", "{ not json\n")
	layRaw(t, home, "lch-261104120000000002-cccccccc.json", `{"schema_version":1,"cost":"a lot"}`+"\n")
	layRaw(t, home, "stray.json", "{}\n")
	layRaw(t, home, ".abcd-tmp-0011223344556677", "half a write")
	before := snapshot(t, home)
	got := readAll(t, home, LedgerQuery{All: true})
	if len(got.Entries) != 1 || got.Totals.SpendByKey["a"] != 1 {
		t.Errorf("listing = %+v", got)
	}
	if len(got.Newer) != 1 || got.Newer[0] != "lch-261104120000000000-aaaaaaaa" {
		t.Errorf("newer = %v", got.Newer)
	}
	if len(got.Unreadable) != 3 {
		t.Errorf("unreadable = %+v, want the two broken entries and the stray, not the temporary file", got.Unreadable)
	}
	for _, u := range got.Unreadable {
		if u.File == "" || u.Why == "" {
			t.Errorf("unreadable = %+v", u)
		}
	}
	if after := snapshot(t, home); after != before {
		t.Errorf("a read changed the store:\n%s\n%s", before, after)
	}
}

func snapshot(t *testing.T, home string) string {
	t.Helper()
	var b strings.Builder
	ents, err := os.ReadDir(abcdhome.Path(home, ledgerFolder))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		raw, err := os.ReadFile(filepath.Join(abcdhome.Path(home, ledgerFolder), e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString(e.Name() + "=" + string(raw))
	}
	return b.String()
}

// A store that does not exist yet is an empty listing, and one behind a link is
// refused rather than read.
func TestReadingAStoreThatIsAbsentOrLinked(t *testing.T) {
	got, err := ReadLedger(t.TempDir(), LedgerQuery{All: true})
	if err != nil || len(got.Entries) != 0 {
		t.Fatalf("absent: %+v, %v", got, err)
	}
	home := t.TempDir()
	target := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, abcdhome.Path(home, ledgerFolder)); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadLedger(home, LedgerQuery{All: true}); err == nil {
		t.Fatal("a linked store was read")
	}
}

// Decision 12: running is judged on the process. An entry whose process is
// gone, or whose id another process now holds, reads as lost; never as running.
func TestRunningIsJudgedOnTheProcess(t *testing.T) {
	home := t.TempDir()
	layEntry(t, home, LedgerEntry{Run: "alive", Outcome: OutcomeRunning, PID: os.Getpid()})
	layEntry(t, home, LedgerEntry{Run: "gone", Outcome: OutcomeRunning, PID: 0})
	layEntry(t, home, LedgerEntry{Run: "reused", Outcome: OutcomeRunning, PID: os.Getpid(), PIDStartUnix: 1000})
	got := readAll(t, home, LedgerQuery{All: true})
	want := map[string]string{"alive": OutcomeRunning, "gone": OutcomeLost, "reused": OutcomeLost}
	for _, e := range got.Entries {
		if e.Outcome != want[e.Run] {
			t.Errorf("%s reads %s, want %s", e.Run, e.Outcome, want[e.Run])
		}
	}
	if got.Totals.Running != 1 || got.Totals.Lost != 2 {
		t.Errorf("totals = %+v", got.Totals)
	}
	raw, err := os.ReadFile(filepath.Join(abcdhome.Path(home, ledgerFolder), got.Entries[1].ID+ledgerExt))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"outcome":"running"`) {
		t.Errorf("a reader rewrote the entry: %s", raw)
	}
}

// With the process table unreadable a running entry stays as written, and the
// listing says it could not be judged.
func TestAnUnreadableTableLeavesRunningUnjudgedAndSaysSo(t *testing.T) {
	home := t.TempDir()
	layEntry(t, home, LedgerEntry{Run: "r", Outcome: OutcomeRunning, PID: 12345})
	got, err := readLedger(home, LedgerQuery{All: true}, func() (procTable, error) { return nil, os.ErrInvalid })
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].Outcome != OutcomeRunning || !got.LivenessUnread {
		t.Errorf("listing = %+v", got)
	}
}

// Criterion 12: two launches for one await are named as a pair; the same await
// in two lanes or runs is not one await, and a placeholder is no launch.
func TestTwoLaunchesForOneAwaitAreNamed(t *testing.T) {
	home := t.TempDir()
	a := layEntry(t, home, LedgerEntry{Run: "r1", Lane: "l1", Await: "receipt.json", Route: RouteKey, Key: "k", Outcome: OutcomeFailed})
	b := layEntry(t, home, LedgerEntry{Run: "r1", Lane: "l1", Await: "receipt.json", Route: RouteKey, Key: "k", Outcome: OutcomeFinished})
	layEntry(t, home, LedgerEntry{Run: "r1", Lane: "l2", Await: "receipt.json", Route: RouteKey, Key: "k", Outcome: OutcomeFinished})
	layEntry(t, home, LedgerEntry{Run: "r2", Lane: "l1", Await: "receipt.json", Route: RouteKey, Key: "k", Outcome: OutcomeFinished})
	layEntry(t, home, LedgerEntry{Run: "r3", Lane: "l1", Await: "receipt.json", Route: RouteKey, Key: "k", Outcome: OutcomeFinished})
	layEntry(t, home, LedgerEntry{Run: "r3", Lane: "l1", Await: "receipt.json", Route: RouteHost, Outcome: OutcomeHanded})
	got := readAll(t, home, LedgerQuery{All: true}).Repeats
	if len(got) != 1 || got[0].Run != "r1" || got[0].Lane != "l1" || got[0].Await != "receipt.json" || len(got[0].Launches) != 2 {
		t.Fatalf("repeats = %+v", got)
	}
	for _, id := range []string{a.ID, b.ID} {
		if !strings.Contains(strings.Join(got[0].Launches, " "), id) {
			t.Errorf("the pair does not name %s: %v", id, got[0].Launches)
		}
	}
}
