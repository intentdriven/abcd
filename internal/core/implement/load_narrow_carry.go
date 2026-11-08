package implement

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"slices"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// A narrow wait carries what it spent against its two caps to its next try
// (dcn-2611080342331070). A drive's harness ends each command after about ten
// minutes, so a lane gates its tests in short tries, each run again when the one
// before gives up: a two-minute wait never spends the WaitNarrowYieldCap of five,
// and each try is a new waiter whose budgets start at zero, so without a carry a
// lane could stand aside for ever, and a queue it cannot see past would hold it
// for ever too. So a narrow wait that gives up records, for its key, the time it
// has stood aside (holderWatch.asideSpent) and how long waiters queued ahead have
// alone kept it shut (the queue cap's clock), with the waiters it saw, and the
// next try of the same key starts from them, as one long wait would have gone on.
// A try that opens drops the entry: its command's next run is a new wait.
//
// The record is this account's own, under its machine home, owner-only and
// written atomically; only this account's waiters read it. One that is not a
// regular file this account owns, is open to another account's writes, is too
// large or does not parse is read as nothing carried, and never stops a wait.
// Only a narrow wait carries: a whole-run waiter's place is its process start,
// which every account reads from the table.
//
// An entry lapses one of its first try's timeouts after the last try gave up: a
// try that follows later than that is not that wait's retry, and starts afresh.

// WaitNarrowCarryFileName is the record of the narrow waits' carried budgets
// under ~/.abcd.noindex/.
const WaitNarrowCarryFileName = "load-wait-narrow-carry.json"

// The record's bounds: the entries kept, the latest first; the waiters each
// remembers as seen queued ahead; and the bytes read or written.
const (
	maxCarryEntries = 32
	maxCarrySeen    = 32
	maxCarryBytes   = 32 << 10
)

// carryOwner reads the owner of the record from its descriptor's FileInfo. It is a
// var because a test cannot make a file another account owns; production never
// reassigns it.
var carryOwner = fsutil.InfoOwnerUID

// narrowCarry is one key's carried budget.
type narrowCarry struct {
	// Key is narrowCarryKey's digest.
	Key string `json:"key"`
	// First is when the first try started, and Timeout its own timeout; Last is
	// when the latest try gave up. The entry lapses at Last plus Timeout.
	First   time.Time     `json:"first"`
	Timeout time.Duration `json:"timeout_ns"`
	Last    time.Time     `json:"last"`
	// Aside is the time spent standing aside for an overdue whole run.
	Aside time.Duration `json:"aside_ns"`
	// Queue is how long, by the last try's end, waiters queued ahead had alone
	// kept the wait shut, and QueueSeen those waiters; zero is no such run.
	Queue     time.Duration `json:"queue_ns"`
	QueueSeen []int         `json:"queue_seen,omitempty"`
}

// lapsed says the entry carries nothing at t: past its lapse, or from a clock
// that ran backwards.
func (e narrowCarry) lapsed(t time.Time) bool {
	return e.Timeout <= 0 || t.Before(e.First) || !t.Before(e.Last.Add(e.Timeout))
}

// narrowCarryFile is the record.
type narrowCarryFile struct {
	Entries []narrowCarry `json:"entries"`
}

// narrowCarryKey is the key a narrow wait carries its budget under: a digest of
// the lane's worktree (the checkout root the wait runs in), the stage's write
// root when a drive's stage runs it (runner.StageWritesEnv), and the gated
// command's words, each length-delimited by the JSON encoding so no two inputs
// run together into one.
//
// No other lane can take it over. Two lanes never share a worktree: each drive
// lane, and each session working beside another, has its own. A drive's stage
// names its lane's directory under its run in the write root, so the same lane in
// another run carries another key, even where a worktree's path is reused. A
// different command is another key, so a lane's one test run never lends its
// budget to its next, different one. Outside a drive the worktree and the command
// are all there is to tell a lane by: a session that runs the same command again
// in the same worktree within one timeout of the last try is that wait's retry.
func narrowCarryKey(dir, stage string, words []string) string {
	raw, _ := json.Marshal(struct {
		Dir   string   `json:"dir"`
		Stage string   `json:"stage"`
		Words []string `json:"words"`
	}{dir, stage, words})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// readCarryRecord reads the record, or nothing when it cannot be trusted: absent,
// not a regular file, not this account's, writable by another, too large, or
// not parsing.
func readCarryRecord(root *os.Root) narrowCarryFile {
	// The root follows a link inside it whatever the flags say, so the name is
	// judged unopened first, and the descriptor must be the file judged.
	named, err := root.Lstat(WaitNarrowCarryFileName)
	if err != nil || !named.Mode().IsRegular() {
		return narrowCarryFile{}
	}
	f, err := root.OpenFile(WaitNarrowCarryFileName, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return narrowCarryFile{}
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || !os.SameFile(named, fi) || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o022 != 0 || fi.Size() > maxCarryBytes {
		return narrowCarryFile{}
	}
	if uid, err := carryOwner(fi); err != nil || uid != uint32(os.Geteuid()) {
		return narrowCarryFile{}
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxCarryBytes+1))
	if err != nil || len(raw) > maxCarryBytes {
		return narrowCarryFile{}
	}
	var file narrowCarryFile
	if json.Unmarshal(raw, &file) != nil {
		return narrowCarryFile{}
	}
	return file
}

// loadCarry is key's carried budget at t, when a live one is recorded.
func loadCarry(root *os.Root, key string, t time.Time) (narrowCarry, bool) {
	for _, e := range readCarryRecord(root).Entries {
		if e.Key == key && !e.lapsed(t) && e.Aside >= 0 && e.Queue >= 0 {
			return e, true
		}
	}
	return narrowCarry{}, false
}

// storeCarry records e at t in place of its key's entry, or drops the key's entry
// when e carries nothing, pruning every lapsed entry and keeping the latest
// maxCarryEntries. It is done under the hold's lock, so two waiters of the account
// never write over each other's entries; a lock it cannot take, or a record it
// cannot write, carries nothing to the next try, and stops nothing.
func storeCarry(root *os.Root, e narrowCarry, t time.Time) {
	if len(e.QueueSeen) > maxCarrySeen {
		e.QueueSeen = e.QueueSeen[:maxCarrySeen]
	}
	_ = fsutil.WithFileLockIn(root, waitLockFileName, waitLockTimeout, func() error {
		file := readCarryRecord(root)
		had := len(file.Entries)
		file.Entries = slices.DeleteFunc(file.Entries, func(o narrowCarry) bool { return o.Key == e.Key || o.lapsed(t) })
		carries := (e.Aside > 0 || e.Queue > 0) && !e.lapsed(t)
		if carries {
			file.Entries = append(file.Entries, e)
		} else if had == 0 {
			return nil
		}
		slices.SortStableFunc(file.Entries, func(a, b narrowCarry) int { return b.Last.Compare(a.Last) })
		file.Entries = file.Entries[:min(len(file.Entries), maxCarryEntries)]
		raw, err := json.Marshal(file)
		if err != nil || len(raw)+1 > maxCarryBytes {
			return nil
		}
		return fsutil.WriteFileAtomicInRoot(root, WaitNarrowCarryFileName, append(raw, '\n'), abcdhome.FileMode)
	})
}
