package runner

// ledger_read.go reads the launch ledger: leniently, field by field, and
// without ever writing, so a reader on an older build is never locked out of
// an entry and never changes one (itd-2611031342011261, criterion 4). A launch
// the store cannot show is named, never dropped silently.

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// TokenBasis is what a token total counts, said wherever one is shown:
// reasoning tokens sit inside the completion count, so they add nothing.
const TokenBasis = "every count the harness reports: prompt, completion, cache read and cache write"

// CostNote is said wherever a cost is shown (the fourth scope condition).
const CostNote = "a cost is the harness's own estimate at its price table, and the provider's invoice is the final figure; " +
	"only a key's cost counts as spend"

// maxLedgerEntry bounds one entry's read; an entry is well under a kilobyte.
const maxLedgerEntry = 64 << 10

// LedgerQuery says which launches to read: one session's, one run's, or every
// one.
type LedgerQuery struct {
	Session string
	Run     string
	All     bool
}

// LedgerProblem is a file in the store the reader could not take as an entry.
type LedgerProblem struct {
	File string `json:"file"`
	Why  string `json:"why"`
}

// AwaitRepeat is two or more launches for one await: the first place a role
// paid for twice shows (iss-2611031337115764).
type AwaitRepeat struct {
	Run      string   `json:"run"`
	Lane     string   `json:"lane,omitempty"`
	Await    string   `json:"await"`
	Launches []string `json:"launches"`
}

// LedgerTotals adds a listing up. SpendByKey is money, in USD, per key;
// EstimateUSD is what the rest would have cost, which is no money spent.
type LedgerTotals struct {
	Launches    int                `json:"launches"`
	Running     int                `json:"running"`
	Finished    int                `json:"finished"`
	Failed      int                `json:"failed"`
	Stopped     int                `json:"stopped"`
	Lost        int                `json:"lost"`
	Handed      int                `json:"handed_to_host"`
	SpendByKey  map[string]float64 `json:"spend_usd_by_key"`
	EstimateUSD float64            `json:"estimate_usd"`
	Tokens      int64              `json:"tokens"`
	APIMillis   int64              `json:"api_ms"`
}

// LedgerListing is what a read of the ledger found, oldest launch first.
type LedgerListing struct {
	Entries []LedgerEntry `json:"entries"`
	Totals  LedgerTotals  `json:"totals"`
	// TokenBasis and CostNote say what the totals count.
	TokenBasis string `json:"token_basis"`
	CostNote   string `json:"cost_note"`
	// Unreadable is every file the reader could not take as an entry. It
	// cannot say whose launch one was, so a filtered listing names them too.
	Unreadable []LedgerProblem `json:"unreadable"`
	// Newer is the id of every entry a later abcd wrote, read as far as this
	// one can.
	Newer []string `json:"newer"`
	// Repeats is every await with two or more launches.
	Repeats []AwaitRepeat `json:"repeats"`
	// LivenessUnread is true when the process table could not be read, so a
	// running entry stands as written and none was judged lost.
	LivenessUnread bool `json:"liveness_unread,omitempty"`
}

// tempFilePrefix opens the atomic writer's temporary files, passed over by a
// reader and cleared by the pruner once a crash has left them.
const tempFilePrefix = ".abcd-tmp-"

// procTable says whether the process an entry names is the one running.
type procTable func(e LedgerEntry) bool

// liveProcTable reads this machine's process table once.
func liveProcTable() (procTable, error) {
	read := sessionname.Live().ReadTable
	if read == nil {
		return nil, errors.ErrUnsupported
	}
	t, err := read()
	if err != nil {
		return nil, err
	}
	return func(e LedgerEntry) bool {
		if e.PID <= 0 {
			return false
		}
		if e.PIDStartUnix != 0 {
			return t.Alive(sessionname.Host{PID: e.PID, StartUnix: e.PIDStartUnix})
		}
		// The launcher's id, with no start of its own: the launcher was running
		// when it wrote the entry, so a process that started later holds the id
		// since. An entry that says no time cannot be judged that way.
		if e.StartedAt.IsZero() {
			return slices.ContainsFunc(t.Procs, func(p machineload.Proc) bool { return p.PID == e.PID })
		}
		return t.HeldSince(e.PID, e.StartedAt)
	}, nil
}

// ReadLedger reads home's ledger for q. An absent store is an empty listing; a
// store that cannot be opened (behind a link, or not a folder) is an error.
func ReadLedger(home string, q LedgerQuery) (LedgerListing, error) {
	return readLedger(home, q, liveProcTable)
}

// Read is ReadLedger on this ledger's store.
func (l *Ledger) Read(q LedgerQuery) (LedgerListing, error) { return ReadLedger(l.home, q) }

func readLedger(home string, q LedgerQuery, table func() (procTable, error)) (LedgerListing, error) {
	out := LedgerListing{Entries: []LedgerEntry{}, TokenBasis: TokenBasis, CostNote: CostNote,
		Unreadable: []LedgerProblem{}, Newer: []string{}, Repeats: []AwaitRepeat{},
		Totals: LedgerTotals{SpendByKey: map[string]float64{}}}
	root, err := fsutil.OpenHomeScope(home, abcdhome.Rel(ledgerFolder))
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) && !storeIsFile(home) {
		return out, nil
	}
	if err != nil {
		return LedgerListing{}, fmt.Errorf("runner: the launch ledger at %s could not be read: %w", abcdhome.Display(ledgerFolder), err)
	}
	defer root.Close()
	dir, err := root.Open(".")
	if err != nil {
		return LedgerListing{}, fmt.Errorf("runner: the launch ledger at %s could not be read: %w", abcdhome.Display(ledgerFolder), err)
	}
	names, err := dir.Readdirnames(-1)
	dir.Close()
	if err != nil {
		return LedgerListing{}, fmt.Errorf("runner: the launch ledger at %s could not be read: %w", abcdhome.Display(ledgerFolder), err)
	}
	slices.Sort(names)
	for _, name := range names {
		if strings.HasPrefix(name, tempFilePrefix) {
			continue
		}
		e, ok := readLedgerFile(root, name, &out)
		if !ok || !q.selects(e) {
			continue
		}
		if e.Schema > LedgerSchema {
			out.Newer = append(out.Newer, e.ID)
		}
		out.Entries = append(out.Entries, e)
	}
	judgeRunning(&out, table)
	for _, e := range out.Entries {
		out.Totals.add(e)
	}
	out.Repeats = awaitRepeats(out.Entries)
	return out, nil
}

// storeIsFile reports whether the store's own name holds something other than
// a folder, which is a fault to name; a parent that is not a folder only means
// abcd's home was never made.
func storeIsFile(home string) bool {
	fi, err := os.Lstat(abcdhome.Path(home, ledgerFolder))
	return err == nil && !fi.IsDir()
}

// readLedgerFile takes one file as an entry, or names it on out.
func readLedgerFile(root *os.Root, name string, out *LedgerListing) (LedgerEntry, bool) {
	bad := func(why string) (LedgerEntry, bool) {
		out.Unreadable = append(out.Unreadable, LedgerProblem{File: name, Why: why})
		return LedgerEntry{}, false
	}
	if !launchFileRe.MatchString(name) {
		return bad("its name is not a launch entry's")
	}
	raw, err := fsutil.ReadGuardedInRoot(root, name, maxLedgerEntry)
	if errors.Is(err, os.ErrNotExist) {
		// Pruned since the folder was listed: no fault, and no launch to name.
		return LedgerEntry{}, false
	}
	if err != nil {
		return bad("it could not be read as a regular file of at most 64 KiB")
	}
	var e LedgerEntry
	var shape *json.UnmarshalTypeError
	if err := json.Unmarshal(raw, &e); err != nil && !(errors.As(err, &shape) && e.Schema > LedgerSchema) {
		// A later abcd may reshape a field: the decoder fills the rest, and the
		// entry is named as newer. At this version a bad field is a bad entry.
		return bad("it is not an entry's JSON")
	}
	if e.Schema < 1 {
		return bad("it names no schema version, so it is no launch's entry")
	}
	e.ID = strings.TrimSuffix(name, ledgerExt)
	return e, true
}

func (q LedgerQuery) selects(e LedgerEntry) bool {
	switch {
	case q.All:
		return true
	case q.Run != "":
		return e.Run == q.Run || e.Orchestrator.Kind == OrchestratorRun && e.Orchestrator.ID == q.Run
	default:
		return e.Orchestrator.Kind == OrchestratorSession && e.Orchestrator.ID == q.Session
	}
}

// judgeRunning turns each running entry whose process is gone into a lost one,
// in the listing only. The table is read only when an entry needs it.
func judgeRunning(out *LedgerListing, table func() (procTable, error)) {
	var alive procTable
	for i := range out.Entries {
		e := &out.Entries[i]
		if e.Outcome != OutcomeRunning {
			continue
		}
		if alive == nil && !out.LivenessUnread {
			t, err := table()
			if err != nil {
				out.LivenessUnread = true
				continue
			}
			alive = t
		}
		if alive != nil && !alive(*e) {
			e.Outcome = OutcomeLost
		}
	}
}

// add counts one entry. A figure no harness reports (below zero, or a sum past
// what a number holds) counts in no sum, or leaves the sum at its largest.
func (t *LedgerTotals) add(e LedgerEntry) {
	t.Launches++
	switch e.Outcome {
	case OutcomeRunning:
		t.Running++
	case OutcomeFinished:
		t.Finished++
	case OutcomeFailed:
		t.Failed++
	case OutcomeStopped:
		t.Stopped++
	case OutcomeLost:
		t.Lost++
	case OutcomeHanded:
		t.Handed++
	}
	if c := e.Cost; c != nil && c.Unit == oracle.CostUSD && oracle.ReadableCost(c.Amount) {
		if e.Route == RouteKey && e.Key != "" {
			if oracle.ReadableCost(t.SpendByKey[e.Key] + c.Amount) {
				t.SpendByKey[e.Key] += c.Amount
			}
		} else if oracle.ReadableCost(t.EstimateUSD + c.Amount) {
			t.EstimateUSD += c.Amount
		}
	}
	if u := e.Usage; u != nil {
		for _, n := range []int{u.PromptTokens, u.CompletionTokens, u.CacheReadTokens, u.CacheWriteTokens} {
			t.Tokens = addCapped(t.Tokens, int64(n))
		}
	}
	if e.APIMillis != nil {
		t.APIMillis = addCapped(t.APIMillis, *e.APIMillis)
	}
}

func addCapped(sum, n int64) int64 {
	if n <= 0 {
		return sum
	}
	if sum > math.MaxInt64-n {
		return math.MaxInt64
	}
	return sum + n
}

// awaitRepeats names every await two or more launches ran for. A host's
// placeholder is no launch.
func awaitRepeats(entries []LedgerEntry) []AwaitRepeat {
	out := []AwaitRepeat{}
	at := map[[3]string]int{}
	seen := map[[3]string][]string{}
	var order [][3]string
	for _, e := range entries {
		if e.Await == "" || e.Run == "" || e.Route == RouteHost {
			continue
		}
		k := [3]string{e.Run, e.Lane, e.Await}
		if _, ok := seen[k]; !ok {
			order = append(order, k)
		}
		seen[k] = append(seen[k], e.ID)
		at[k]++
	}
	for _, k := range order {
		if at[k] > 1 {
			out = append(out, AwaitRepeat{Run: k[0], Lane: k[1], Await: k[2], Launches: seen[k]})
		}
	}
	return out
}
