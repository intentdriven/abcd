// Package reservesweep deletes spent reservation branches from origin
// (iss-2610142200389875). A reservation (internal/core/claim) is a branch
// `refs/heads/abcd-reserved/<record-id>` that ends with a tombstone and is
// never deleted by the verb that wrote it, and the reservations ruleset
// forbids its deletion to everyone but the one automation the ruleset names
// as its bypass actor, so the branches pile up on the forge.
//
// The sweep fetches the reservations and origin's default branch, judges
// each branch (judge), and with Apply deletes the spent ones, each with a
// compare-and-swap on the exact tip it judged, so a branch re-reserved
// meanwhile survives. Without Apply it deletes nothing and says what it
// would. It never writes to stdout: a front door renders its Report.
package reservesweep

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// DefaultAge is how long a branch must have been untouched, past its lease,
// before the sweep counts it spent.
const DefaultAge = 7 * 24 * time.Hour

// MinAge is the shortest age a sweep takes: a day, the longest a lease holds
// (lease.Max), so no age the sweep accepts is shorter than a lease.
const MinAge = 24 * time.Hour

// maxChain bounds the first-parent walk of one branch, as the drain's
// hand-back walk bounds its own (dcn-2610161806433529); a longer branch is
// kept, its history unread.
const maxChain = 256

// maxObjectBytes bounds the read of one commit object (claim's bound).
const maxObjectBytes = 8192

// Options carries what a caller may set about a sweep.
type Options struct {
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// Age is how long a branch must have been untouched; zero means
	// DefaultAge. Shorter than MinAge is refused.
	Age time.Duration
	// Apply deletes the spent branches; without it nothing is deleted.
	Apply bool
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now().UTC().Truncate(time.Second)
	}
	return time.Now().UTC().Truncate(time.Second)
}

// Outcome of one deletion.
const (
	// Deleted: origin took the deletion.
	Deleted = "deleted"
	// Moved: the branch moved after it was judged, so the compare-and-swap
	// refused the deletion and the branch stands.
	Moved = "moved"
	// Failed: origin refused the deletion for another reason (the ruleset,
	// a hook, the network); Error says why.
	Failed = "failed"
)

// Branch is one reservation branch as the sweep judged it.
type Branch struct {
	Record string
	// Ref is the branch's full ref on origin.
	Ref    string
	Commit string
	// Kind is the tip's kind.
	Kind  Kind
	Spent bool
	// Reason is one of the Spent* or Kept* reasons.
	Reason string
	// Why says what made a commit unknown, or the tip unreadable.
	Why string
	// SettledAt is when the tip last held or was written; zero when the tip
	// could not be read.
	SettledAt time.Time
	// Outcome is the deletion's outcome under Apply; "" otherwise.
	Outcome string
	// Error is why a deletion failed.
	Error string
}

// Report is one sweep.
type Report struct {
	AsOf    time.Time
	Age     time.Duration
	Applied bool
	// DefaultUnread is why origin's default branch could not be read, so no
	// branch was judged spent by its record being finished there.
	DefaultUnread string
	Branches      []Branch
}

// Counts are the report's totals: branches spent, kept and deleted.
func (r Report) Counts() (spent, kept, deleted int) {
	for _, b := range r.Branches {
		if b.Spent {
			spent++
		} else {
			kept++
		}
		if b.Outcome == Deleted {
			deleted++
		}
	}
	return spent, kept, deleted
}

// Failed reports whether any deletion failed for a reason other than the
// branch having moved.
func (r Report) Failed() bool {
	return slices.ContainsFunc(r.Branches, func(b Branch) bool { return b.Outcome == Failed })
}

// beforeDelete, when a test sets it, runs between the judgement and the
// deletions: the window another writer's push lands in.
var beforeDelete func(root string)

// Sweep fetches the reservations and origin's default branch, judges every
// reservation branch, and under Apply deletes the spent ones.
func Sweep(root string, o Options) (Report, error) {
	age := o.Age
	if age == 0 {
		age = DefaultAge
	}
	if age < MinAge {
		return Report{}, fmt.Errorf("an age of %s is shorter than a day, the longest a lease holds; nothing was deleted", age)
	}
	now := o.now()
	rep := Report{AsOf: now, Age: age, Applied: o.Apply, Branches: []Branch{}}
	gap, err := claim.Fetch(root, claim.Options{Now: func() time.Time { return now }})
	if err != nil {
		return rep, err
	}
	snap, err := claim.View(root, now)
	if err != nil {
		return rep, fmt.Errorf("%s; nothing was deleted", fsutil.RedactHome(err.Error()))
	}
	rep.DefaultUnread = gap
	var finished map[string][]peers.RecordAt
	if gap == "" {
		if ref := gitutil.DefaultRef(root); strings.HasPrefix(ref, "refs/remotes/"+claim.Remote+"/") {
			if finished, err = peers.RecordsAt(root, ref); err != nil {
				rep.DefaultUnread = "could not read " + gitutil.ShortRef(ref) + ": " + fsutil.RedactHome(err.Error())
				finished = nil
			}
		} else {
			rep.DefaultUnread = "this checkout knows no default branch of " + claim.Remote
		}
	}
	tree, err := gitutil.EmptyTree(root)
	if err != nil {
		return rep, fmt.Errorf("%s; nothing was deleted", fsutil.RedactHome(err.Error()))
	}
	cut := shallowCommits(root)
	for _, r := range snap.Reservations {
		b := Branch{Record: r.Record, Ref: claim.RefPrefix + r.Record, Commit: r.Commit}
		f := facts{Unreadable: r.Unreadable, EffectiveExpiry: r.EffectiveExpiry, Terminal: terminal(finished, r.Record)}
		if f.Unreadable == "" {
			f.Chain, f.ChainWhole = readChain(root, r.Record, r.Commit, tree, cut)
		}
		v := judge(f, now, age)
		b.Spent, b.Reason, b.SettledAt = v.Spent, v.Reason, v.SettledAt
		b.Why = r.Unreadable
		if len(f.Chain) > 0 {
			b.Kind = f.Chain[0].Kind
		}
		if v.Reason == KeptUnknown || v.Reason == KeptHistory {
			for _, c := range f.Chain {
				if c.Kind == KindUnknown {
					b.Why = c.Why
					break
				}
			}
		}
		rep.Branches = append(rep.Branches, b)
	}
	if !o.Apply {
		return rep, nil
	}
	if beforeDelete != nil {
		beforeDelete(root)
	}
	for i := range rep.Branches {
		b := &rep.Branches[i]
		if !b.Spent {
			continue
		}
		b.Outcome, b.Error = deleteBranch(root, b.Ref, b.Commit)
	}
	return rep, nil
}

// terminalFolders are the folders whose membership finishes a record, as
// internal/core/claim's derived release reads them: an issue resolved or
// won't-fixed, an intent shipped or superseded.
var terminalFolders = map[string][]string{
	"iss": {"resolved", "wontfix"},
	"itd": {intent.BucketShipped, intent.BucketSuperseded},
}

// terminal reports whether id sits on the default branch only in terminal
// folders. It is narrower than the claim package's derived release, which
// counts a record in any terminal folder: a record also left in open/ is
// one the ledger refuses to read, and the sweep keeps its branch.
func terminal(finished map[string][]peers.RecordAt, id string) bool {
	recs := finished[id]
	if len(recs) == 0 {
		return false
	}
	for _, r := range recs {
		if !slices.Contains(terminalFolders[id[:3]], r.Folder) {
			return false
		}
	}
	return true
}

// readChain reads the branch's first-parent line from tip, at most maxChain
// commits, tip first. whole is true when it reached the root; a commit it
// cannot read is the last it returns, as KindUnknown, and a commit a shallow
// clone cut (cut) is read as unknown too, its parents being absent.
func readChain(root, record, tip, tree string, cut map[string]bool) (chain []commitInfo, whole bool) {
	at := tip
	for len(chain) < maxChain {
		obj, err := gitutil.RunCappedBytes(root, maxObjectBytes, "--no-replace-objects", "cat-file", "commit", at)
		if err != nil {
			return append(chain, commitInfo{Commit: at, Kind: KindUnknown, Why: "its commit object cannot be read"}), false
		}
		c := classify(record, at, obj, tree)
		if cut[at] {
			c.Kind, c.Why = KindUnknown, "a shallow clone cut its history"
		}
		chain = append(chain, c)
		if c.Kind == KindUnknown {
			return chain, false
		}
		if c.Parent == "" {
			return chain, true
		}
		at = c.Parent
	}
	return chain, false
}

// shallowCommits are the commits a shallow clone cut, whose parents are
// absent; empty in a whole clone.
func shallowCommits(root string) map[string]bool {
	out := map[string]bool{}
	p, err := gitutil.Run(root, "rev-parse", "--git-path", "shallow")
	if err != nil {
		return out
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return out
	}
	for _, l := range strings.Fields(string(data)) {
		out[l] = true
	}
	return out
}

// deleteBranch deletes ref on origin only while it still points at sha: one
// push with a lease on that exact tip, which origin's receive-pack checks
// under its ref lock, so a branch moved since it was judged survives.
func deleteBranch(root, ref, sha string) (outcome, why string) {
	if !gitutil.IsFullSHA(sha) || !strings.HasPrefix(ref, claim.RefPrefix) {
		return Failed, "the branch or its tip is not one the sweep reads"
	}
	out, err := gitutil.Net(root, deleteArgs(ref, sha)...)
	if err == nil {
		return Deleted, ""
	}
	if strings.Contains(out, "stale info") || strings.Contains(err.Error(), "stale info") {
		return Moved, ""
	}
	return Failed, fsutil.RedactHome(pushReason(out, err))
}

// deleteArgs is the one push the sweep makes: a deletion of ref on origin,
// leased on its tip sha.
func deleteArgs(ref, sha string) []string {
	return []string{"push", "--porcelain", "--force-with-lease=" + ref + ":" + sha, claim.Remote, ":" + ref}
}

// pushReason is a failed push's reason: the porcelain lines naming a refused
// ref, then git's error.
func pushReason(stdout string, err error) string {
	var lines []string
	for _, l := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(l, "!") {
			lines = append(lines, strings.Join(strings.Fields(l), " "))
		}
	}
	if len(lines) == 0 {
		return err.Error()
	}
	return strings.Join(lines, "; ") + "; " + err.Error()
}

// IsNoOrigin reports whether err is the refusal of a checkout with no origin.
func IsNoOrigin(err error) bool { return errors.Is(err, claim.ErrNoOrigin) }
