package claim

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Reading the announcements (itd-2609151838312703, spc-2610150728396006 step
// 3): `abcd status` fetches them with the reservations and lists everyone's
// status; the prompt hook shows each new one once, from this checkout's disk
// alone. Both read through one intake, which walks each status branch from
// where this checkout last read it (namespace.since), reads every commit the
// walk added in one bounded batch, and keeps what it read in the local tier:
// when this checkout first fetched each announcement, which counts its life
// (C9), and the validated fields it carries, the only text a render shows
// (C6). Nothing a writer chose is ever shown but those fields, each already
// held to its grammar.

// How a reader names an announcement's sender (C17, C11).
const (
	// SenderVerified is a sender the forge's record of who pushed names as
	// the branch's owner.
	SenderVerified = "verified"
	// SenderClaimed is a sender the forge has not been asked about, or could
	// not say about: the native default.
	SenderClaimed = "claimed"
	// SenderDoubtful is an announcement whose sender is not the branch's
	// owner, by its own trailer or by the forge's record.
	SenderDoubtful = "doubtful"
)

// The reasons an announcement or a status branch is doubtful (C11). Each is
// fixed text: nothing a writer chose is echoed into one.
const (
	DoubtRewritten = "continuity not verified: the branch was rewritten rather than added to, so an announcement may be missing or shown twice"
	DoubtMore      = "continuity not verified: more was added than one read covers, so an announcement may be missing or shown twice"
	DoubtMerged    = "continuity not verified: a merge was passed, and what it brought in is not read"
	DoubtSender    = "it names a sender other than the branch's owner"
	DoubtPusher    = "the forge's record names another account as its pusher"
)

// announcementSlack is how far an announcement's own expiry may lie before
// the moment this checkout first fetched it and still be news: its life is
// then counted from that first fetch (C9), so a sender's clock up to this far
// slow costs it nothing. One whose own expiry passed earlier still is history
// a first read brought in, not news, and is not shown as standing.
const announcementSlack = time.Hour

// maxStatusBranches is the most status branches whose names are read that
// one listing fetches and one intake reads, the first in git's ref order:
// the rest are counted (BranchesNotRead), never read. A test lowers it.
var maxStatusBranches = 500

// maxKeptAnnouncements is the most kept announcements one read takes in, the
// newest kept: the rest are counted (AnnouncementsNotRead). A test lowers it.
var maxKeptAnnouncements = 2000

// maxListingLine bounds one line of origin's listing a reader keeps: a ref
// longer than it is still counted, by what it kept of it.
const maxListingLine = 4096

// statusWalkLimit is the most commits one read of one status branch takes
// in: a branch that gained more since this checkout last read it reads as
// doubtful (DoubtMore). A test lowers it.
var statusWalkLimit = maxWalk

// maxIntakeCommits is the most new commits one intake takes in, across every
// status branch (takeIn says in what order): a branch with more past it is
// not marked read at its tip, so a later intake takes in what this one left,
// and no branch is marked read past a commit it did not take in. It bounds
// what the prompt hook's read writes before its deadline.
var maxIntakeCommits = 100

// The status state's files, under statusBranches.stateDir.
const (
	seenDir    = "seen"    // one file per announcement commit read: its fields and first fetch
	shownDir   = "shown"   // one marker per line the prompt hook has shown
	doubtDir   = "doubt"   // one file per status branch found doubtful
	pusherDir  = "pusher"  // one file per announcement the forge was asked about
	skippedRel = "skipped" // the branch names origin's last listing held that are not read
	failedRel  = "fetch-failed"
)

// seenKeep is how long a read announcement is kept after its life ends, so a
// read of the same commit again (a rewritten branch) does not count it anew.
const seenKeep = 2 * MaxAnnouncementLife

// shownKeep is how long a shown marker of a line other than an announcement
// is kept: after it, a doubt or a hold still there is shown once more.
const shownKeep = 7 * 24 * time.Hour

// ErrStatusUnavailable is the prompt hook's answer when the status could not
// be read inside its budget, or at all: it says so, never shows nothing.
var ErrStatusUnavailable = errors.New("the status could not be read")

// Push is one push to a status branch as the forge's record of pushes holds
// it: the tip it moved the branch from (Before, the zero name when the push
// made the branch) and to (After), and the account that pushed. App is true
// when an app made the push: it then judges no commit it carried, which stays
// claimed, but it still shows how far the record reaches.
type Push struct {
	Before, After, Pusher string
	App                   bool
}

// PusherLookup asks the forge who pushed each update to the status branch
// abcd-status/<branch> on origin (C17): its record of pushes, newest first. A
// commit no push in the answer covers is one the forge cannot say about, and
// its sender stays claimed; an error leaves every sender claimed, with the
// reason, but ErrPartialRecord, answered beside the pushes that were read
// when the record was not read in full. ctx carries what is left of the
// fetch's pusherBudget as its deadline, and a lookup ends when it does. The
// front door wires it to the forge adapter; nil asks nothing.
type PusherLookup func(ctx context.Context, root, branch string) ([]Push, error)

// ErrPartialRecord is a PusherLookup's answer, beside the pushes it read,
// when the forge's record was not read in full: older pushes may lie past
// what was read. Nothing it leaves open is kept as asked: a commit no push
// read covers, or one only the owner's pushes cover, is asked about again,
// and only another account named as its pusher is kept.
var ErrPartialRecord = errors.New("the forge's record was not read in full")

// Announcement is one announcement as a reader shows it. Every value in it is
// either fixed text or a field held to its grammar by the announcement's
// validator: a kind from the list, an issue or intent id or "#N", a GitHub
// username, times. A render still passes each through termsafe.
type Announcement struct {
	// Branch is the status branch's owner, the name after abcd-status/.
	Branch string
	Commit string
	// Kind, Item and Sender are the announcement's fields; all "" when it is
	// unreadable.
	Kind, Item, Sender string
	// SentAt is the time it carries, by the sender's clock.
	SentAt time.Time
	// FirstFetched is when this checkout first read it, and Until when it
	// stops standing: FirstFetched plus the life it was sent with (C9).
	FirstFetched, Until time.Time
	// order is where the intake read it: later on the branch, or read by a
	// later intake, is larger. It tells apart two first fetched, and sent, in
	// one second.
	order int64
	// Check is SenderVerified, SenderClaimed or SenderDoubtful; Doubtful is
	// the reason when doubtful, and Pusher the login the forge named when it
	// is the forge's record that makes it so (a valid username, else "").
	Check    string
	Doubtful string
	Pusher   string
	// Unreadable is why the commit does not read as an announcement (its
	// tree is not the empty tree, its message is not the fixed form): it is
	// doubtful, and none of its text is shown.
	Unreadable string
}

// Standing reports whether the announcement still stands at now.
func (a Announcement) Standing(now time.Time) bool { return a.Unreadable == "" && now.Before(a.Until) }

// BranchDoubt is a status branch read as doubtful (C11): named, with the
// reason, while every other branch is still read.
type BranchDoubt struct {
	// Branch is the name after abcd-status/, always a lower-case username: a
	// branch by any other name is counted, never kept as a doubt.
	Branch string
	Reason string
	// Tip is the commit the branch ended in when it was found doubtful; "" for
	// a name that was never read.
	Tip     string
	FoundAt time.Time
}

// Unread is a namespace whose last fetch failed (C10): when it was last read
// (zero when never), when the fetch failed and git's reason, home-redacted.
type Unread struct {
	Since    time.Time
	FailedAt time.Time
	Reason   string
}

// StatusView is everyone's status as this checkout last fetched it, read from
// disk alone.
type StatusView struct {
	// Origin is false in a checkout with no remote named origin, which has no
	// status to read: everything else is then empty.
	Origin bool
	// NoTier is true in a checkout without its local tier, which keeps
	// nothing a fetch brings and so reads no announcement.
	NoTier bool
	// FetchedAt is when abcd last fetched the status branches here; zero when
	// nothing here stamped a fetch.
	FetchedAt time.Time
	// Unread is the status branches' failed fetch, nil when their last fetch
	// succeeded; ReservationsUnread is the reservations'.
	Unread, ReservationsUnread *Unread
	// Announcements is every one still standing, newest first, and every
	// unreadable one read in the last day.
	Announcements []Announcement
	// Doubts is every status branch read as doubtful whose name is a
	// lower-case username, by name.
	Doubts []BranchDoubt
	// UnreadNames counts the status branches whose names are not read: the
	// larger of origin's last listing's count and the count of such branches
	// this checkout holds a copy of (a fetch of the whole namespace brings
	// them). No such name is carried, since a pusher chose it; unreadKey,
	// built from a digest of the full names, keys the line that counts them,
	// and never leaves this package.
	UnreadNames int
	// UnreadNamesUnknown is true when what origin's last listing counted
	// could not be read back: UnreadNames then counts only the copies held
	// here, and says nothing of origin.
	UnreadNamesUnknown bool
	unreadKey          string
	// BranchesNotRead counts the status branches past maxStatusBranches, and
	// AnnouncementsNotRead the announcements kept here past
	// maxKeptAnnouncements: neither is read by this view.
	BranchesNotRead, AnnouncementsNotRead int
	// BranchesLeft counts the status branches holding new commits the
	// intake did not take in, past the most one intake takes
	// (maxIntakeCommits): none is marked read, so a later read takes in the
	// rest. leftKey, a digest of each with the newest commit not taken in,
	// keys the line that counts them.
	BranchesLeft int
	leftKey      string
	// Holds is every release being cut, judged as the merge hold judges them
	// (JudgedHolds).
	Holds []Hold
	// Lapsed is every release cut whose reservation lapsed with no tag
	// (Snapshot.Lapsed): the prompt block names each once, so a session
	// shown the hold is told it holds no more (iss-2610241107191205).
	Lapsed []Lapsed
	// holdsUnread is true when the holds could not be read, and the rest of
	// the view was: the prompt block says so in its own line.
	holdsUnread bool
}

// StatusFetch is what `abcd status`'s fetch could not read: Gap is why
// origin's default branch could not be read, and SendersUnchecked why the
// forge could not be asked who pushed, each "" when it could.
type StatusFetch struct {
	Gap              string
	SendersUnchecked string
	// NotDropped is how many of this checkout's copies of status branches
	// that should have been deleted before the fetch were not, and why; ""
	// when every one was.
	NotDropped string
}

// FetchStatus is `abcd status`'s read of the remote: it lists the status
// branches on origin (`git ls-remote`), then fetches, in one command, the
// reservations with origin's release tags, every status branch whose name is
// a lower-case GitHub username and origin's default branch, and stamps when.
// A branch under abcd-status/ by any other name is not fetched: two names that
// differ only in case break every fetch on a case-insensitive filesystem
// (decision 10), so it is counted, never named and never kept (C11). Then it
// reads in what the fetch brought (the intake), and asks the
// forge, through lookup when it is given, who pushed each standing
// announcement whose pusher is not yet known here (C17). A checkout with no
// remote named origin answers ErrNoOrigin. A listing or a fetch that fails is
// kept in the local tier, its time and reason, for every reader to say (C10),
// and answers its *FetchError. Before the fetch, every tracking ref under the
// status namespace that is not among the names it fetches is deleted
// (dropUnlisted): a branch deleted on origin, one whose name is not read, or
// one past the cap leaves nothing to read; the copies a deletion leaves twice
// are counted in NotDropped, and the fetch goes on.
func FetchStatus(root string, lookup PusherLookup, o Options) (StatusFetch, error) {
	now := o.now()
	ok, err := HasOrigin(root)
	if err != nil {
		return StatusFetch{}, refuse("could not list this checkout's remotes: %s; nothing was fetched", redacted(err))
	}
	if !ok {
		return StatusFetch{}, ErrNoOrigin
	}
	names, skipped, past, err := listStatusBranches(root, "")
	if err != nil {
		noteFetchFailure(root, statusBranches, now, redacted(err))
		return StatusFetch{}, &FetchError{Reason: redacted(err), what: statusBranches.noun, outcome: "nothing was fetched"}
	}
	// Only the names origin lists, and of them only those the fetch reads,
	// are kept: the named refspecs prune nothing, and a branch left from an
	// earlier fetch would read as fetched now.
	// The count is kept before the fetch stamps its time, so a reader never
	// finds a fetch newer than the count it kept (keptSkipped).
	keepSkipped(root, skipped, past, now)
	notDropped := statusBranches.dropUnlisted(root, names)
	status := statusBranches
	status.only = names
	gap, err := fetch(root, now, "nothing was fetched", reservations, status)
	if err != nil {
		return StatusFetch{}, err
	}
	keepFirstFetched(root, now, nil)
	res := StatusFetch{Gap: gap, NotDropped: notDropped}
	in, err := takeIn(context.Background(), root, now)
	if err != nil {
		// The reader takes it in again and names what it could not read.
		return res, nil
	}
	if lookup != nil {
		res.SendersUnchecked = checkSenders(root, now, lookup, in.held)
	}
	return res, nil
}

// listStatusBranches asks origin which status branches it holds (`git
// ls-remote origin refs/heads/abcd-status/*`), read line by line, never
// whole, so however many origin lists the listing is read: the first
// maxStatusBranches names that are lower-case GitHub usernames, in origin's
// order, then sorted; how many more such names it listed (past), which are
// not read and not fetched; and the tally of every other name, which is not
// fetched and is never kept. own, when not "", is the sender's own branch
// name: it is always among names when origin lists it, past the cap too,
// so an announcement is built on its tip; past still counts it, since the
// reader does not read it there.
func listStatusBranches(root, own string) (names []string, skipped nameTally, past int, err error) {
	args := []string{"ls-remote", "--refs", Remote, StatusRefPrefix + "*"}
	trace(args...)
	names = []string{}
	err = gitutil.NetEach(root, maxListingLine, func(line string) {
		sha, ref, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || !gitutil.IsFullSHA(sha) {
			return
		}
		name, ok := strings.CutPrefix(ref, StatusRefPrefix)
		switch {
		case !ok || name == "":
		case !statusName(name):
			skipped.add(name)
		case len(names) < maxStatusBranches:
			names = append(names, name)
		default:
			past++
			if name == own {
				names = append(names, name)
			}
		}
	}, args...)
	if err != nil {
		return nil, nameTally{}, 0, errors.New(redacted(err))
	}
	sort.Strings(names)
	return names, skipped, past, nil
}

// nameTally counts names a reader does not read, without keeping them: how
// many, and a digest of every name in full, in the order they were added, so
// two sets are told apart however long or alike their names. Both listings it
// counts (origin's and this checkout's refs) name branches in git's ref
// order.
type nameTally struct {
	n int
	h hash.Hash
}

// add counts name.
func (t *nameTally) add(name string) {
	if t.h == nil {
		t.h = sha256.New()
	}
	t.n++
	t.h.Write([]byte(hashName(name)))
}

// digest is the tally's digest; "" when it counted nothing.
func (t nameTally) digest() string {
	if t.h == nil {
		return ""
	}
	return hex.EncodeToString(t.h.Sum(nil))
}

// statusName reports whether name, after abcd-status/, is a branch the reader
// reads: a GitHub username in lower case.
func statusName(name string) bool { return validUsername(name) && name == strings.ToLower(name) }

// ReadStatus is everyone's status as this checkout last fetched it, from
// disk alone: no fetch and no push (invariant 7). It reads the release holds
// as the merge hold judges them, then takes in whatever a fetch brought since
// the last read, under ctx, then reads what is kept. It is the one read of the
// holds, and of the lapsed release cuts with them, that `abcd status` and the
// prompt hook make: neither reads them again. A
// read of the holds that fails does not stop the rest: the announcements are
// still read, and the error, naming the holds' read, is answered with them.
// An intake that fails, or a ctx that ends first, answers its error with what
// was read before it: the view's origin, its fetch's date and its failed
// fetches always, and the holds when they were read.
func ReadStatus(ctx context.Context, root string, now time.Time, o Options) (StatusView, error) {
	now = now.UTC().Truncate(time.Second)
	ok, err := HasOrigin(root)
	if err != nil {
		return StatusView{}, err
	}
	if !ok {
		return StatusView{}, nil
	}
	v := StatusView{Origin: true, FetchedAt: lastFetched(root, statusBranches), NoTier: tierPresent(root) != nil}
	v.Unread = unreadOf(root, statusBranches)
	v.ReservationsUnread = unreadOf(root, reservations)
	var holdsErr error
	if v.Holds, v.Lapsed, err = readHolds(root, now, o); err != nil {
		holdsErr = &holdsUnreadError{err}
	}
	in, err := takeIn(ctx, root, now)
	if err != nil {
		return v, errors.Join(holdsErr, err)
	}
	v.Announcements, v.AnnouncementsNotRead = keptAnnouncementsCounted(root, now, in.held)
	v.BranchesLeft, v.leftKey = len(in.left), in.leftKey()
	for _, d := range keptDoubts(root, now) {
		// A doubt of a branch this checkout no longer holds is not shown.
		if in.held == nil || in.held[d.Branch] {
			v.Doubts = append(v.Doubts, d)
		}
	}
	listed, err := keptSkipped(root)
	v.UnreadNamesUnknown = err != nil
	v.UnreadNames = max(listed.Count, in.unread.n)
	v.BranchesNotRead = max(listed.PastCap, in.pastCap)
	v.unreadKey = listed.Digest + "\x00" + in.unread.digest()
	if err := ctx.Err(); err != nil {
		return v, errors.Join(holdsErr, err)
	}
	if holdsErr != nil {
		// Only the holds failed: answered on its own, so a reader can tell.
		return v, holdsErr
	}
	return v, nil
}

// holdsUnreadError is ReadStatus's answer when only its read of the release
// holds failed: everything else in the view was read.
type holdsUnreadError struct{ err error }

func (e *holdsUnreadError) Error() string { return "reading the release holds: " + e.err.Error() }
func (e *holdsUnreadError) Unwrap() error { return e.err }

// readHolds is ReadStatus's read of the release holds and the lapsed cuts:
// judgedHoldsAndLapsed, which a test replaces with one that fails.
var readHolds = judgedHoldsAndLapsed

// intake is what one intake found.
type intake struct {
	// taken is how many commits it read.
	taken int
	// held is every status branch this checkout holds, by name.
	held map[string]bool
	// unread tallies the status branches this checkout holds whose names are
	// not read.
	unread nameTally
	// pastCap counts the status branches this checkout holds, by names that
	// are read, past maxStatusBranches: none of them is read.
	pastCap int
	// left is each status branch holding new commits this intake did not
	// take in once maxIntakeCommits was spent, by its name and the newest
	// commit not taken in (its tip, for a branch not walked): none of them is
	// marked read at its tip, so a later intake takes in the rest.
	left []string
}

// leftKey is a digest of what in left unread, so the line that counts it is
// shown again once a later intake has read on.
func (in intake) leftKey() string {
	if len(in.left) == 0 {
		return ""
	}
	return hashName(strings.Join(in.left, "\n"))
}

// leftRel names the status branches the last intake left unread, each with
// the turn it was left at (keptLeft).
const leftRel = "intake-left"

// maxLeftBytes bounds what keptLeft reads: a line per branch left, at most
// one per branch an intake reads.
const maxLeftBytes = 128 << 10

// takeIn reads every status branch this checkout holds from where it last
// read it (namespace.since, under ctx), reads each commit the walk added and
// this checkout has not read before, in one bounded batch per branch, keeps
// each as seen at now, then marks the branch read at its tip. It takes in at
// most maxIntakeCommits new commits in all. Every branch the last intake left
// unread is walked first, the one left at the earliest turn first, then every
// other branch in git's ref order; a branch walked and left again is left at
// a turn after every branch already waiting, and one left without a walk
// keeps the turn it was first left at, so no branch waits behind more than
// the branches left before it, however many appear or move after it. A
// branch already left, cut short by the cap or left without a walk, takes in
// half its share, rounded down, from its newest unread commits and the rest
// from its oldest (freshRuns), and is marked read at the newest commit
// beneath which every commit is read, so each unread commit is read within a
// bounded number of intakes while the commits above the mark stay within the
// walk's limit; past it the branch is shown as doubtful.
// A branch whose name is not a lower-case username is tallied, never read and
// never kept. One whose tip is not a commit, or whose walk could not verify
// continuity (it moved back or was rewritten, ran past the walk's limit,
// on a first read as on any other, or passed a merge), is kept as doubtful,
// and every other branch is still read. One whose tip is the commit it was
// last read at is passed over without a walk. One that gained more new
// commits than the intake had left to take in is not marked read at its tip,
// and once nothing is left the branches after it are not walked at all: each
// is counted in left, and a later intake takes in the rest. It writes only
// the local tier, unflushed; a checkout without one keeps nothing and reads
// nothing. When ctx ends, what was read is kept, the branch at hand is not
// marked, and ctx's error is answered.
func takeIn(ctx context.Context, root string, now time.Time) (intake, error) {
	var in intake
	if tierPresent(root) != nil {
		return in, nil
	}
	if err := intakeFailure.Load(); err != nil {
		return in, *err
	}
	refs, err := statusRefs(ctx, root)
	if err != nil {
		return in, err
	}
	in.held = map[string]bool{}
	// The branches read: the first maxStatusBranches whose names are read,
	// in git's ref order; the rest are counted. Those the last intake left
	// are walked first, by the turn each was left at.
	left := keptLeft(root)
	var waiting, rest []statusRef
	for _, r := range refs {
		name := strings.TrimPrefix(r.ref, StatusTrackingPrefix)
		in.held[name] = true
		switch {
		case !statusName(name):
			in.unread.add(name)
		case len(waiting)+len(rest) >= maxStatusBranches:
			in.pastCap++
		default:
			if _, ok := left.at[name]; ok {
				waiting = append(waiting, r)
			} else {
				rest = append(rest, r)
			}
		}
	}
	sort.SliceStable(waiting, func(i, j int) bool {
		return left.at[strings.TrimPrefix(waiting[i].ref, StatusTrackingPrefix)] <
			left.at[strings.TrimPrefix(waiting[j].ref, StatusTrackingPrefix)]
	})
	// This intake's turns: a branch first left without a walk takes turn,
	// and one walked and left again turn+1, after every branch waiting.
	turn := left.turn
	stillLeft := map[string]int64{}
	tree := ""
	for _, r := range append(waiting, rest...) {
		name := strings.TrimPrefix(r.ref, StatusTrackingPrefix)
		if statusBranches.lastRead(root, r.ref) == r.sha {
			continue
		}
		at, wasLeft := left.at[name]
		if in.taken >= maxIntakeCommits {
			// Nothing is left to take in: not walked, not marked, counted.
			in.left = append(in.left, name+" "+r.sha)
			if !wasLeft {
				at = turn
			}
			stillLeft[name] = at
			continue
		}
		h, err := statusBranches.sinceCtx(ctx, root, r.ref, statusWalkLimit)
		if err != nil {
			return in, err
		}
		if h.Tip == "" {
			continue
		}
		if h.Unreadable != "" {
			// Doubtful, and read at that tip: the branch is not walked again
			// until it moves.
			keepDoubt(root, BranchDoubt{Branch: name, Reason: "its tip " + h.Unreadable, Tip: h.Tip, FoundAt: now})
			if err := statusBranches.markRead(root, r.ref, h.Tip); err != nil {
				return in, err
			}
			continue
		}
		// One doubt per branch, the first that applies. A first read past the
		// walk's limit is doubtful whatever its commits carry: within the
		// bounded walk nothing settles whether a line beneath it can still
		// stand (dcn-2610161100416718, as refined).
		reason := ""
		switch {
		case h.Since != "" && h.Rewritten:
			reason = DoubtRewritten
		case h.More:
			reason = DoubtMore
		case h.Merged:
			reason = DoubtMerged
		}
		if reason != "" {
			keepDoubt(root, BranchDoubt{Branch: name, Reason: reason, Tip: h.Tip, FoundAt: now})
		}
		// At most maxIntakeCommits new commits in all. The rest are not read
		// now, and the branch is not marked read at its tip, so a later intake
		// takes them in: a mark at the tip would pass over them for good.
		runs, next, readTo := freshRuns(root, h.Added, maxIntakeCommits-in.taken, wasLeft)
		var fresh []string
		for _, run := range runs {
			fresh = append(fresh, run.commits...)
		}
		if len(fresh) > 0 {
			if tree == "" {
				if tree, err = emptyTree(root); err != nil {
					return in, err
				}
			}
			read, err := readAnnouncements(ctx, root, tree, fresh)
			if err != nil {
				return in, err
			}
			// The order's base is the wall clock's, so a later intake orders
			// after, and each commit takes its place on the branch from it. A
			// run beneath a commit an earlier intake kept is what that intake
			// left of the branch, older than what it took in, so it takes that
			// intake's time and orders beneath that commit, by how far beneath.
			base := time.Now().UnixNano()
			for _, run := range runs {
				at, from := now, base+int64(len(h.Added)-run.at-len(run.commits))
				if run.beneath >= 0 {
					if k, ok := keptAnnouncement(root, h.Added[run.beneath]); ok {
						at, from = k.FirstFetched, k.order-int64(run.at+len(run.commits)-1-run.beneath)
					}
				}
				keepSeen(root, name, run.commits, read, at, from)
			}
			in.taken += len(fresh)
		}
		if err := ctx.Err(); err != nil {
			return in, err
		}
		if next != "" {
			in.left = append(in.left, name+" "+next)
			stillLeft[name] = turn + 1
			if readTo < len(h.Added) {
				// Every commit beneath is read: the next walk starts above it.
				if err := statusBranches.markRead(root, r.ref, h.Added[readTo]); err != nil {
					return in, err
				}
			}
			continue
		}
		if err := statusBranches.markRead(root, r.ref, h.Tip); err != nil {
			return in, err
		}
	}
	if len(stillLeft) > 0 || len(left.at) > 0 {
		keepLeft(root, turn+2, stillLeft)
	}
	// Pruning reads every kept file, so it runs only when something new was
	// read in, never at every prompt.
	if in.taken > 0 {
		pruneStatus(ctx, root, now, refs)
	}
	return in, nil
}

// leftSet is the status branches an intake left unread, each by name with
// the turn it was left at, and the turn the next intake starts its count
// from, past every turn kept.
type leftSet struct {
	turn int64
	at   map[string]int64
}

// keptLeft is the set of branches the last intake left unread, as kept under
// leftRel; empty when none is kept or it does not read. A line that does not
// read, or names a branch whose name is not read, is passed over.
func keptLeft(root string) leftSet {
	s := leftSet{at: map[string]int64{}}
	data := readStatusFile(root, leftRel, maxLeftBytes)
	if data == nil {
		return s
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if n, err := strconv.ParseInt(lines[0], 10, 64); err == nil && n >= 0 {
		s.turn = n
	}
	for _, l := range lines[1:] {
		turn, name, ok := strings.Cut(l, " ")
		n, err := strconv.ParseInt(turn, 10, 64)
		if !ok || err != nil || n < 0 || !statusName(name) {
			continue
		}
		s.at[name] = n
		s.turn = max(s.turn, n+1)
	}
	return s
}

// keepLeft keeps the branches left, each with its turn, and the turn the
// next intake counts from, replacing what was kept, unflushed and best
// effort: a set not kept costs only the order of the next intake's walk.
func keepLeft(root string, turn int64, at map[string]int64) {
	var b strings.Builder
	b.WriteString(strconv.FormatInt(turn, 10) + "\n")
	names := make([]string, 0, len(at))
	for name := range at {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b.WriteString(strconv.FormatInt(at[name], 10) + " " + name + "\n")
	}
	_ = writeUnflushedIn(root, statusBranches.stateDir, leftRel, []byte(b.String()))
}

// freshRun is a stretch of a branch's commits, newest first, this checkout
// has not read and one intake takes in: at is where in the walk's commits it
// begins, and beneath where the nearest commit above it that was read before
// lies (-1 when none is).
type freshRun struct {
	at, beneath int
	commits     []string
}

// freshRuns picks from added (newest first) the commits this checkout has not
// read, at most budget of them: the newest, or, when split, the newest half of
// budget (rounded down) and the oldest for the rest of it, so a split share
// of one takes the oldest and the read mark moves. It answers them in
// runs, each a stretch of added none of whose commits was left out or read
// before; the newest commit it left out ("" when it left none out); and
// readTo, the place in added from which every commit beneath is read, before
// or now (len(added) when the oldest is not).
func freshRuns(root string, added []string, budget int, split bool) (runs []freshRun, next string, readTo int) {
	read := make([]bool, len(added))
	var unread []int
	if r, err := os.OpenRoot(root); err == nil {
		for i, c := range added {
			if read[i] = seenKeptIn(r, c); !read[i] {
				unread = append(unread, i)
			}
		}
		r.Close()
	} else {
		for i := range added {
			unread = append(unread, i)
		}
	}
	take := unread
	if len(unread) > budget {
		newest := budget
		if split {
			newest = budget / 2
		}
		take = append(slices.Clone(unread[:newest]), unread[len(unread)-(budget-newest):]...)
		next = added[unread[newest]]
	}
	taken := make([]bool, len(added))
	for _, i := range take {
		taken[i] = true
	}
	above := -1
	for i, c := range added {
		switch {
		case read[i]:
			above = i
		case !taken[i]:
		case len(runs) > 0 && runs[len(runs)-1].at+len(runs[len(runs)-1].commits) == i:
			runs[len(runs)-1].commits = append(runs[len(runs)-1].commits, c)
		default:
			runs = append(runs, freshRun{at: i, beneath: above, commits: []string{c}})
		}
	}
	readTo = len(added)
	for readTo > 0 && (read[readTo-1] || taken[readTo-1]) {
		readTo--
	}
	return runs, next, readTo
}

// intakeFailure, when a test sets it (IntakeFailsForTest), is what every
// intake answers before it reads anything.
var intakeFailure atomic.Pointer[error]

// IntakeFailsForTest makes every intake answer err until restore is called,
// for a test in another package that needs the read after a fetch to fail:
// every write the intake makes is best effort, so nothing on disk can make
// it fail once the fetch before it succeeded.
func IntakeFailsForTest(err error) (restore func()) {
	intakeFailure.Store(&err)
	return func() { intakeFailure.Store(nil) }
}

// statusRef is one status branch this checkout holds, as for-each-ref names
// it.
type statusRef struct{ ref, sha string }

// statusRefs is every status branch this checkout holds, in git's ref order,
// read as stored, under ctx, line by line (storedRefs): a fetch of the whole
// namespace (`abcd announce`'s) may bring in as many as origin holds, and
// however many that is the listing is read; an intake reads at most
// maxStatusBranches of them. A ref whose name is too long to read whole is
// kept by what was read of it, a name that is not read, and so is counted.
func statusRefs(ctx context.Context, root string) ([]statusRef, error) {
	var refs []statusRef
	err := storedRefs(ctx, root, StatusTrackingPrefix, func(ref, sha string, _ bool) {
		refs = append(refs, statusRef{ref: ref, sha: sha})
	})
	if err != nil {
		return nil, fmt.Errorf("reading the status branches: %s", redacted(err))
	}
	return refs, nil
}

// readAnnouncement is one commit as the intake read it.
type readAnnouncement struct {
	h head
	a announcement
}

// readAnnouncements reads each of commits as an announcement, in two
// commands whatever their number: their types and sizes first (`cat-file
// --batch-check`, which git reads from each object's header), then every one
// that is a commit within the object bound (`cat-file --batch`), each read
// through parseTip with the announcement's parser. A commit past the bound is
// never streamed, and reads as unreadable, as readCommit reads it; so does an
// object that is not a commit. Both commands read history as stored and are
// killed when ctx ends.
func readAnnouncements(ctx context.Context, root, tree string, commits []string) (map[string]readAnnouncement, error) {
	out := make(map[string]readAnnouncement, len(commits))
	stdin := strings.Join(commits, "\n") + "\n"
	check, err := storedGitInput(ctx, root, stdin, len(commits)*(64+len(commits[0]))+64,
		"cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
	if err != nil {
		return nil, fmt.Errorf("reading the status branches: %s", redacted(err))
	}
	var batch []string
	total := 0
	for _, l := range strings.Split(strings.TrimRight(string(check), "\n"), "\n") {
		f := strings.Fields(l)
		if len(f) < 2 || !slices.Contains(commits, f[0]) {
			continue
		}
		c := f[0]
		if len(f) != 3 || f[1] != "commit" {
			out[c] = readAnnouncement{h: head{Commit: c, Unreadable: reasonNoObject}}
			continue
		}
		n, err := strconv.Atoi(f[2])
		if err != nil || n < 0 {
			out[c] = readAnnouncement{h: head{Commit: c, Unreadable: reasonNoObject}}
			continue
		}
		if n > maxObjectBytes {
			out[c] = readAnnouncement{h: head{Commit: c, Unreadable: reasonObjectTooBig}}
			continue
		}
		batch = append(batch, c)
		total += n + len(c) + 32
	}
	if len(batch) > 0 {
		objs, err := storedGitInput(ctx, root, strings.Join(batch, "\n")+"\n", total+64, "cat-file", "--batch")
		if err != nil {
			return nil, fmt.Errorf("reading the status branches: %s", redacted(err))
		}
		// One string, walked by slicing: each object is cut from what is
		// left without copying the rest again.
		rest := string(objs)
		for rest != "" {
			header, body, ok := strings.Cut(rest, "\n")
			if !ok {
				break
			}
			f := strings.Fields(header)
			if len(f) != 3 || f[1] != "commit" {
				return nil, errors.New("reading the status branches: git's batch read did not parse")
			}
			n, err := strconv.Atoi(f[2])
			if err != nil || n < 0 || n > len(body) {
				return nil, errors.New("reading the status branches: git's batch read did not parse")
			}
			h, a := parseTip(f[0], []byte(body[:n]), tree, parseAnnouncement)
			out[f[0]] = readAnnouncement{h: h, a: a}
			rest = strings.TrimPrefix(body[n:], "\n")
		}
	}
	for _, c := range commits {
		if _, ok := out[c]; !ok {
			out[c] = readAnnouncement{h: head{Commit: c, Unreadable: reasonNoObject}}
		}
	}
	return out, nil
}

// seenEntry is one read announcement as the local tier keeps it: the branch
// it was read on, when this checkout first read it, and the validated fields
// or the reason it is unreadable.
type seenEntry struct {
	Branch       string `json:"branch"`
	FirstFetched string `json:"first_fetched"`
	Kind         string `json:"kind,omitempty"`
	Item         string `json:"item,omitempty"`
	Sender       string `json:"sender,omitempty"`
	SentAt       string `json:"sent_at,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	Unreadable   string `json:"unreadable,omitempty"`
	Order        int64  `json:"order,omitempty"`
}

// statusFile is rel, a file under the status state directory.
func statusFile(rel string) string { return statusBranches.stateDir + "/" + rel }

// statusSub is the status state directory's sub-directory sub, or the
// directory itself when sub is "".
func statusSub(sub string) string {
	if sub == "" {
		return statusBranches.stateDir
	}
	return statusFile(sub)
}

// seenKept reports whether commit was already read here: its kept file
// reads back as an entry with the time it was first fetched. A file that
// reads but does not parse as one (a write cut short left it empty or
// partial, since it is written without a flush) is removed, best effort, and
// the commit is read again. A file that does not read at all for now (it may
// not be opened, or the read failed) is left as it is: the commit is read
// again, and its file written again over it.
func seenKept(root, commit string) bool {
	r, err := os.OpenRoot(root)
	if err != nil {
		return false
	}
	defer r.Close()
	return seenKeptIn(r, commit)
}

// seenKeptIn is seenKept through r, the checkout's root.
func seenKeptIn(r *os.Root, commit string) bool {
	rel := statusFile(seenDir + "/" + commit)
	data, err := fsutil.ReadGuardedInRoot(r, rel, 4096)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	if err != nil {
		return false
	}
	var e seenEntry
	if json.Unmarshal(data, &e) == nil {
		if _, err := time.Parse(time.RFC3339, e.FirstFetched); err == nil {
			return true
		}
	}
	_ = r.Remove(rel)
	return false
}

// keepSeen keeps each of commits, read on branch, as first fetched at now,
// through one open of the root, unflushed and best effort: one not kept is
// read again by the next intake. commits is newest first, and the newest
// takes the largest order, from base.
func keepSeen(root, branch string, commits []string, read map[string]readAnnouncement, now time.Time, base int64) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	dir := statusFile(seenDir)
	if err := mkStateDir(r, dir); err != nil {
		return
	}
	for i, c := range commits {
		e := seenEntry{Branch: branch, FirstFetched: stamp(now), Order: base + int64(len(commits)-1-i)}
		if a := read[c]; a.h.Unreadable != "" {
			e.Unreadable = a.h.Unreadable
		} else {
			e.Kind, e.Item, e.Sender, e.SentAt, e.ExpiresAt = a.a.Kind, a.a.Item, a.a.Sender, stamp(a.a.SentAt), stamp(a.a.ExpiresAt)
		}
		data, err := json.Marshal(e)
		if err != nil {
			continue
		}
		_ = writeUnflushedAt(r, dir, c, data)
	}
}

// writeStatusFile writes name under the status state directory's sub, made
// when missing (the tier itself never is), best effort.
func writeStatusFile(root, sub, name string, data []byte) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	dir := statusBranches.stateDir
	if sub != "" {
		dir += "/" + sub
	}
	if err := mkStateDir(r, dir); err != nil {
		return
	}
	_ = fsutil.WriteFileAtomicInRoot(r, dir+"/"+name, data, stateMode)
}

// tmpPrefix starts the temporary file writeUnflushedIn writes before it
// renames it into place.
const tmpPrefix = ".tmp-"

// writeUnflushedIn writes name under dir, a directory under the local tier
// made when missing, through a temporary file renamed into place, with no
// flush to stable storage: for state whose loss costs only a read made again
// or a line shown again, written where the prompt hook's deadline applies.
// A temporary file a write cut off leaves behind is removed by pruneStatus.
func writeUnflushedIn(root, dir, name string, data []byte) error {
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := mkStateDir(r, dir); err != nil {
		return err
	}
	return writeUnflushedAt(r, dir, name, data)
}

// writeUnflushedAt is writeUnflushedIn through r, into dir, which exists.
func writeUnflushedAt(r *os.Root, dir, name string, data []byte) error {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return err
	}
	tmp := dir + "/" + tmpPrefix + hex.EncodeToString(buf[:])
	f, err := r.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, stateMode)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = r.Rename(tmp, dir+"/"+name)
	}
	if werr != nil {
		_ = r.Remove(tmp)
	}
	return werr
}

// readStatusFile is rel under the status state directory, at most max bytes;
// nil when it cannot be read.
func readStatusFile(root, rel string, max int64) []byte {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, statusFile(rel), max)
	if err != nil {
		return nil
	}
	return data
}

// statusEntries is the names of the files under the status state directory's
// sub (the directory itself when sub is ""); nil when it cannot be read.
func statusEntries(root, sub string) []fs.DirEntry {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil
	}
	defer r.Close()
	f, err := r.Open(statusSub(sub))
	if err != nil {
		return nil
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil
	}
	return entries
}

// keptAnnouncements is every announcement kept here, read on a branch in
// held (every branch, when held is nil), that still stands at now, and every
// unreadable one first read in the last day, newest first: by
// when this checkout first read it, then, on one branch, by the order the
// branch holds them in, and across branches by the time each carries. Each kept
// file is read back through the same grammar its fields were held to, so a
// file that does not read is passed over.
func keptAnnouncements(root string, now time.Time, held map[string]bool) []Announcement {
	out, _ := keptAnnouncementsCounted(root, now, held)
	return out
}

// keptAnnouncementsCounted is keptAnnouncements from at most
// maxKeptAnnouncements kept files, the newest kept (by when each was
// written), and how many more were kept past that cap, which it does not read.
func keptAnnouncementsCounted(root string, now time.Time, held map[string]bool) ([]Announcement, int) {
	var out []Announcement
	var entries []fs.DirEntry
	for _, e := range statusEntries(root, seenDir) {
		if gitutil.IsFullSHA(e.Name()) && e.Type().IsRegular() {
			entries = append(entries, e)
		}
	}
	past := 0
	if len(entries) > maxKeptAnnouncements {
		at := make(map[string]time.Time, len(entries))
		for _, e := range entries {
			if info, err := e.Info(); err == nil {
				at[e.Name()] = info.ModTime()
			}
		}
		sort.SliceStable(entries, func(i, j int) bool {
			if a, b := at[entries[i].Name()], at[entries[j].Name()]; !a.Equal(b) {
				return a.After(b)
			}
			return entries[i].Name() < entries[j].Name()
		})
		past = len(entries) - maxKeptAnnouncements
		entries = entries[:maxKeptAnnouncements]
	}
	for _, e := range entries {
		c := e.Name()
		a, ok := keptAnnouncement(root, c)
		if !ok || held != nil && !held[a.Branch] {
			continue
		}
		if a.Unreadable != "" {
			if now.Sub(a.FirstFetched) < MaxAnnouncementLife {
				out = append(out, a)
			}
			continue
		}
		if a.Standing(now) {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].FirstFetched.Equal(out[j].FirstFetched) {
			return out[i].FirstFetched.After(out[j].FirstFetched)
		}
		// On one branch, the order the branch holds them in; across branches,
		// the time each carries.
		if out[i].Branch == out[j].Branch && out[i].order != out[j].order {
			return out[i].order > out[j].order
		}
		if !out[i].SentAt.Equal(out[j].SentAt) {
			return out[i].SentAt.After(out[j].SentAt)
		}
		if out[i].order != out[j].order {
			return out[i].order > out[j].order
		}
		return out[i].Commit < out[j].Commit
	})
	return out, past
}

// keptAnnouncement is commit's kept entry as a reader shows it, its sender
// judged: doubtful when its sender trailer names someone other than the
// branch's owner (C11) or the forge's record names another pusher (C17),
// verified when the forge's record names the owner, claimed otherwise.
func keptAnnouncement(root, commit string) (Announcement, bool) {
	data := readStatusFile(root, seenDir+"/"+commit, 4096)
	if data == nil {
		return Announcement{}, false
	}
	var e seenEntry
	if err := json.Unmarshal(data, &e); err != nil || !statusName(e.Branch) {
		return Announcement{}, false
	}
	first, err := time.Parse(time.RFC3339, e.FirstFetched)
	if err != nil {
		return Announcement{}, false
	}
	a := Announcement{Branch: e.Branch, Commit: commit, FirstFetched: first.UTC(), order: e.Order}
	if e.Unreadable != "" {
		a.Unreadable, a.Check = fixedReason(e.Unreadable), SenderDoubtful
		return a, true
	}
	if !slices.Contains(AnnouncementKinds, e.Kind) || !validUsername(e.Sender) {
		return Announcement{}, false
	}
	if canon, _, err := announcementItem(e.Item); err != nil || canon != e.Item {
		return Announcement{}, false
	}
	sent, err1 := time.Parse(time.RFC3339, e.SentAt)
	exp, err2 := time.Parse(time.RFC3339, e.ExpiresAt)
	life := exp.Sub(sent)
	if err1 != nil || err2 != nil || life <= 0 || life > MaxAnnouncementLife {
		return Announcement{}, false
	}
	a.Kind, a.Item, a.Sender, a.SentAt = e.Kind, e.Item, e.Sender, sent.UTC()
	a.Until = a.FirstFetched.Add(life)
	if exp.Add(announcementSlack).Before(a.FirstFetched) {
		// History a first read brought in, not news: it is not counted anew.
		a.Until = exp.UTC()
	}
	a.Check = SenderClaimed
	switch v := pusherKept(root, commit); {
	case !sameHolder(a.Sender, a.Branch):
		a.Check, a.Doubtful = SenderDoubtful, DoubtSender
	case v.known && sameHolder(v.login, a.Branch):
		a.Check = SenderVerified
	case v.known:
		a.Check, a.Doubtful = SenderDoubtful, DoubtPusher
		if validUsername(v.login) {
			a.Pusher = v.login
		}
	}
	return a, true
}

// fixedReason is reason when it is one of the fixed reasons a commit is
// unreadable, and a fixed stand-in for anything else a file holds.
func fixedReason(reason string) string {
	switch reason {
	case reasonObjectTooBig, reasonMessageTooBig, reasonNotEmptyTree, reasonNoObject, reasonCopy, reasonTime,
		reasonAnnouncementKind, reasonAnnouncementItem, reasonAnnouncementSender, reasonAnnouncementExpiry,
		reasonOutsideTrailers, reasonAnnouncementAttribution:
		return reason
	}
	return "it does not read as an announcement"
}

// keepDoubt keeps d, replacing what was kept for its branch, unflushed and
// best effort: a doubt not kept is found again when its branch next moves. A
// doubt kept already, for the same reason at the same tip, is left as it is,
// so a doubt read again at every prompt is not written again at every prompt;
// only the branch's own file is read to tell.
func keepDoubt(root string, d BranchDoubt) {
	if k, ok := keptDoubt(root, hashName(d.Branch), d.FoundAt); ok &&
		k.Branch == d.Branch && k.Reason == d.Reason && k.Tip == d.Tip && !k.FoundAt.IsZero() && k.Tip != "" {
		return
	}
	data, err := json.Marshal(struct {
		Branch  string `json:"branch"`
		Reason  string `json:"reason"`
		Tip     string `json:"tip,omitempty"`
		FoundAt string `json:"found_at"`
	}{d.Branch, d.Reason, d.Tip, stamp(d.FoundAt)})
	if err != nil {
		return
	}
	writeUnflushed(root, doubtDir, hashName(d.Branch), data)
}

// hashName names a file for a value a writer chose, so it never becomes a
// path.
func hashName(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

// keptDoubts is every status branch kept as doubtful, each found in the last
// day by an intake, sorted by name. Only a branch whose name is read is kept
// as doubtful; a file naming any other (one an earlier build kept) is passed
// over.
func keptDoubts(root string, now time.Time) []BranchDoubt {
	var out []BranchDoubt
	for _, e := range statusEntries(root, doubtDir) {
		if d, ok := keptDoubt(root, e.Name(), now); ok {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Branch < out[j].Branch })
	return out
}

// keptDoubt is the doubt kept in the file name under the doubt directory,
// when it reads back, names a branch whose name is read and is named for
// it, and was found in the day before now.
func keptDoubt(root, name string, now time.Time) (BranchDoubt, bool) {
	data := readStatusFile(root, doubtDir+"/"+name, 4096)
	var d struct {
		Branch  string `json:"branch"`
		Reason  string `json:"reason"`
		Tip     string `json:"tip"`
		FoundAt string `json:"found_at"`
	}
	if data == nil || json.Unmarshal(data, &d) != nil || name != hashName(d.Branch) || !statusName(d.Branch) {
		return BranchDoubt{}, false
	}
	found, err := time.Parse(time.RFC3339, d.FoundAt)
	if err != nil || now.Sub(found) >= MaxAnnouncementLife {
		return BranchDoubt{}, false
	}
	if d.Tip != "" && !gitutil.IsFullSHA(d.Tip) {
		return BranchDoubt{}, false
	}
	return BranchDoubt{Branch: d.Branch, Reason: doubtReason(d.Reason), Tip: d.Tip, FoundAt: found.UTC()}, true
}

// doubtReason is reason when it is one a doubt is kept with, and a fixed
// stand-in for anything else a file holds.
func doubtReason(reason string) string {
	switch reason {
	case DoubtRewritten, DoubtMore, DoubtMerged, "its tip " + reasonNoObject:
		return reason
	}
	return "it could not be read"
}

// skippedState is what origin's last listing held that is not read: how many
// names and their tally's digest, never the names, so its size never grows
// with theirs.
type skippedState struct {
	ListedAt string `json:"listed_at"`
	Count    int    `json:"count"`
	Digest   string `json:"digest,omitempty"`
	// PastCap is how many names that are read the listing held past
	// maxStatusBranches, none of them read: only an announcement's fetches
	// one of them, the sender's own.
	PastCap int `json:"past_cap,omitempty"`
}

// keepSkipped keeps the tally of the names origin's listing held that are
// not read, and how many it held past the cap, replacing the last listing's.
func keepSkipped(root string, t nameTally, past int, now time.Time) {
	data, err := json.Marshal(skippedState{ListedAt: stamp(now), Count: t.n, Digest: t.digest(), PastCap: past})
	if err != nil {
		return
	}
	writeStatusFile(root, "", skippedRel, data)
}

// keptSkipped is origin's last listing's tally of names that are not read:
// the zero state when nothing here fetched the status branches, and an error
// when what was kept cannot be read back, or was not kept at their last
// fetch (it is missing, or older than that fetch: its write failed), so a
// reader says the count is unknown rather than count none or an older count.
// Every fetch of the status branches keeps it from its listing before it
// stamps its time: `abcd status`'s and `abcd announce`'s.
func keptSkipped(root string) (skippedState, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return skippedState{}, err
	}
	defer r.Close()
	fetched := lastFetched(root, statusBranches)
	notKept := errors.New("the count of status branch names not read was not kept at their last fetch")
	data, err := fsutil.ReadGuardedInRoot(r, statusFile(skippedRel), 4096)
	if errors.Is(err, fs.ErrNotExist) {
		if fetched.IsZero() {
			return skippedState{}, nil
		}
		return skippedState{}, notKept
	}
	if err != nil {
		return skippedState{}, err
	}
	var s skippedState
	if err := json.Unmarshal(data, &s); err != nil || s.Count < 0 || s.PastCap < 0 {
		return skippedState{}, errors.New("the count of status branch names not read does not parse")
	}
	listed, err := time.Parse(time.RFC3339, s.ListedAt)
	if err != nil {
		return skippedState{}, errors.New("the count of status branch names not read does not parse")
	}
	if listed.Before(fetched) {
		return skippedState{}, notKept
	}
	return s, nil
}

// pusherVerdict is what the forge said of who pushed one announcement: asked
// is true once its record was placed in full for the branch, and known when it
// named a pusher.
type pusherVerdict struct {
	asked, known bool
	login        string
}

// noPusher is kept for a commit the forge's record, placed in full, did not
// cover: it is not asked about again for as long as its announcement is
// kept (seenKeep past its life), and its sender stays claimed.
const noPusher = "?"

// pusherKept is the forge's answer kept for commit.
func pusherKept(root, commit string) pusherVerdict {
	data := readStatusFile(root, pusherDir+"/"+commit, 128)
	login := strings.TrimSpace(string(data))
	switch {
	case data == nil || login == "":
		return pusherVerdict{}
	case login == noPusher:
		return pusherVerdict{asked: true}
	}
	return pusherVerdict{asked: true, known: true, login: login}
}

// pusherBudget bounds one fetch's questions to the forge and the walks that
// place each answer: each lookup is asked under it as its deadline, so a
// forge call still running when it passes is ended; past it, no further
// branch is asked about, a push not yet placed keeps nothing, and the check
// says so.
var pusherBudget = 10 * time.Second

// maxAskedBranches is the most status branches one fetch asks the forge
// about, the one waiting longest first: the rest are counted and asked by a
// later fetch.
const maxAskedBranches = 10

// askedRel names the turn each status branch waits at to be asked about
// (keptAsked).
const askedRel = "forge-asked"

// checkSenders asks lookup, once per branch, who pushed each standing
// announcement the forge was not yet asked about here, and keeps each login
// the answer names that is a valid username (any other answer is kept as a
// login that is not the owner's), and that it named none (noPusher). At most
// maxAskedBranches branches are asked, and none once pusherBudget has passed;
// the rest are counted, and asked by a later fetch. The branches are asked in
// turn: the one waiting longest first, and among those that began waiting at
// one fetch, the one whose newest such announcement is newest first; a branch
// asked goes behind every branch still waiting, and one left unasked keeps
// its turn, so every branch is asked within a bounded number of fetches. A
// branch leaves the queue once every standing announcement on it is asked
// about: a record read in full, an app's push to the tip held here included,
// keeps a verdict for every commit asked (noPusher for one an app's push
// carried), and one not read in full is asked about again. A push names only
// the tip it left, so each is placed by walking the first-parent line from
// that tip down to the tip it moved from, at most the walk's limit of commits, under pusherBudget:
// every commit of the push is judged by its pusher (placePushes). A commit two
// pushes cover (a rewritten branch) takes a pusher other than the owner over
// the owner. A branch whose record could not be placed in full keeps nothing
// of it, and one whose record was not read in full (ErrPartialRecord), or not
// as far as the tip held here, or held against no tip because the branches
// held here could not be read or do not hold this one, keeps only another
// account named as a pusher.
// It returns why the forge could not be asked, or its record could not be
// read or placed in full, "" when it could.
func checkSenders(root string, now time.Time, lookup PusherLookup, held map[string]bool) string {
	ask := map[string][]string{}
	var branches []string // newest first, as keptAnnouncements orders them
	for _, a := range keptAnnouncements(root, now, held) {
		if a.Unreadable == "" && a.Standing(now) && !pusherKept(root, a.Commit).asked {
			if _, ok := ask[a.Branch]; !ok {
				branches = append(branches, a.Branch)
			}
			ask[a.Branch] = append(ask[a.Branch], a.Commit)
		}
	}
	if len(branches) == 0 {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), pusherBudget)
	defer cancel()
	var failed []string
	// The tip this checkout holds of each branch, which a record read in
	// full names as some push's After.
	tips := map[string]string{}
	refs, err := senderRefs(ctx, root)
	tipsKnown := err == nil
	if tipsKnown {
		for _, r := range refs {
			tips[strings.TrimPrefix(r.ref, StatusTrackingPrefix)] = r.sha
		}
	} else {
		failed = append(failed, "the status branches held here could not be read ("+termsafe.Sanitize(redacted(err))+
			"), so whether the forge's record reaches each tip is not known, and a commit it does not name another account for stays claimed and is asked about again")
	}
	kept := keptAsked(root)
	turn := kept.turn
	turnOf := func(b string) int64 {
		if n, ok := kept.at[b]; ok {
			return n
		}
		return turn
	}
	queue := branches
	sort.SliceStable(queue, func(i, j int) bool { return turnOf(queue[i]) < turnOf(queue[j]) })
	next := map[string]int64{}
	unasked := 0
	for i, b := range queue {
		if i >= maxAskedBranches || ctx.Err() != nil {
			unasked = len(queue) - i
			for _, b := range queue[i:] {
				next[b] = turnOf(b)
			}
			break
		}
		// Asked: behind every branch still waiting.
		next[b] = turn + 1
		pushed, err := lookup(ctx, root, b)
		partial := errors.Is(err, ErrPartialRecord)
		if err != nil && !partial {
			failed = append(failed, "abcd-status/"+b+": "+termsafe.Sanitize(redacted(err)))
			continue
		}
		// The forge records a push some time after it lands: a record that
		// names no push to the tip held here has not caught up yet, and is
		// read as one not read in full. With no tip known, or none held of
		// this branch, nothing is held against one, and the record is read
		// as not read in full too, with nothing said of a tip.
		tipHeld := tips[b] != ""
		var atTip *Push
		for i := range pushed {
			if tipHeld && pushed[i].After == tips[b] {
				atTip = &pushed[i]
				break
			}
		}
		short := partial
		behind := tipsKnown && tipHeld && !partial && atTip == nil
		partial = partial || behind || !tipsKnown || !tipHeld
		byCommit, err := placePushes(ctx, root, b, pushed)
		if err != nil {
			failed = append(failed, "abcd-status/"+b+": the forge's record of pushes was not placed in full: "+termsafe.Sanitize(redacted(err)))
			continue
		}
		const askedAgain = ", so a commit it does not name another account for stays claimed and is asked about again"
		switch {
		case behind:
			failed = append(failed, "abcd-status/"+b+": the forge's record does not yet name the push of the tip held here"+askedAgain)
		case short && tipsKnown:
			failed = append(failed, "abcd-status/"+b+": "+ErrPartialRecord.Error()+askedAgain)
		}
		for _, c := range ask[b] {
			login, ok := byCommit[c]
			if partial && (!ok || sameHolder(login, b)) {
				// An older push past what was read may still name another
				// account: nothing is kept that it could change.
				continue
			}
			if !ok {
				writeStatusFile(root, pusherDir, c, []byte(noPusher+"\n"))
				continue
			}
			if !validUsername(login) {
				login = "-"
			}
			writeStatusFile(root, pusherDir, c, []byte(login+"\n"))
		}
	}
	keepAsked(root, turn+2, next)
	if unasked > 0 {
		what := fmt.Sprintf("%d status branches not asked about", unasked)
		if unasked == 1 {
			what = "1 status branch not asked about"
		}
		failed = append(failed, fmt.Sprintf("%s (one fetch asks about at most %d, the longest waiting first, within %s); the next `abcd status` asks them",
			what, maxAskedBranches, pusherBudget))
	}
	return strings.Join(failed, "; ")
}

// askedSet is what askedRel keeps: the turn each branch waits at, by name,
// and the turn the next fetch counts from, past every turn kept.
type askedSet struct {
	turn int64
	at   map[string]int64
}

// maxAskedBytes bounds what keptAsked reads: a line per branch.
const maxAskedBytes = 256 << 10

// keptAsked is the set askedRel keeps; empty when none is kept or it does not
// read. A line that does not read, or names a branch whose name is not read,
// is passed over.
func keptAsked(root string) askedSet {
	s := askedSet{at: map[string]int64{}}
	data := readStatusFile(root, askedRel, maxAskedBytes)
	if data == nil {
		return s
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if n, err := strconv.ParseInt(lines[0], 10, 64); err == nil && n >= 0 {
		s.turn = n
	}
	for _, l := range lines[1:] {
		turn, name, ok := strings.Cut(l, " ")
		n, err := strconv.ParseInt(turn, 10, 64)
		if !ok || err != nil || n < 0 || !statusName(name) {
			continue
		}
		s.at[name] = n
		s.turn = max(s.turn, n+1)
	}
	return s
}

// keepAsked keeps each branch's turn, and the turn the next fetch counts
// from, replacing what was kept, unflushed and best effort: a set not kept
// costs only the order of the next fetch's questions.
func keepAsked(root string, turn int64, at map[string]int64) {
	var b strings.Builder
	b.WriteString(strconv.FormatInt(turn, 10) + "\n")
	names := make([]string, 0, len(at))
	for name := range at {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b.WriteString(strconv.FormatInt(at[name], 10) + " " + name + "\n")
	}
	_ = writeUnflushedIn(root, statusBranches.stateDir, askedRel, []byte(b.String()))
}

// senderRefs is checkSenders' read of the status branches this checkout
// holds: statusRefs, which a test replaces with one that fails.
var senderRefs = statusRefs

// placePushes maps each commit the pushes to abcd-status/<branch> carried to
// its pusher, read from this checkout's copy under ctx: the first-parent
// commits from each push's After down to its Before, at most the walk's
// limit per push. A push that made the branch (Before the zero name) carried
// its whole line, as far as the walk reads it. A push whose Before is not on
// that line, as far as the walk reads it (a forced push, or one that carried
// more than the walk reads), carried commits from its After down to a place
// the walk cannot find: another account's then takes every commit the walk
// read, since any of them may be its, and the owner's vouches for its After
// alone. A commit two pushes carried takes the pusher that is not the owner.
// An app's push judges no commit and is passed over, as is a push whose
// After this checkout does not hold; one that
// ends ctx stops the placing, and ctx's error is answered with nothing
// placed, since a commit an older push also carried may then read as the
// owner's when it is not.
func placePushes(ctx context.Context, root, branch string, pushes []Push) (map[string]string, error) {
	ref := statusBranches.track(branch)
	out := map[string]string{}
	place := func(c, pusher string) {
		if prev, ok := out[c]; !ok || (sameHolder(prev, branch) && !sameHolder(pusher, branch)) {
			out[c] = pusher
		}
	}
	for i, p := range pushes {
		if h := placeHook; h != nil {
			h(i)
		}
		if p.App || !gitutil.IsFullSHA(p.After) {
			// An app's push judges no commit it carried.
			continue
		}
		held, err := hasCommitCtx(ctx, root, p.After)
		if err != nil {
			return nil, err
		}
		if !held {
			continue
		}
		line, err := firstParentsCtx(ctx, root, ref, p.After, statusWalkLimit)
		if err != nil {
			return nil, err
		}
		line, _ = upTo(line, statusWalkLimit)
		carried := line
		if p.Before != zeroCommit {
			if i := slices.Index(line, p.Before); i >= 0 {
				carried = line[:i]
			} else if sameHolder(p.Pusher, branch) {
				carried = line[:1]
			}
		}
		for _, c := range carried {
			place(c, p.Pusher)
		}
	}
	return out, nil
}

// placeHook, when a test sets it, runs before placePushes places push i, so a
// test can end the budget between two pushes.
var placeHook func(i int)

// zeroCommit is the forge's name for no commit: the tip a push that made the
// branch moved it from.
var zeroCommit = strings.Repeat("0", 40)

// tmpKeep is how old a temporary file in the status state must be before the
// prune takes it for one a write cut off left behind.
const tmpKeep = time.Minute

// pruneStatus removes, best effort: every temporary file a write cut off left
// behind (older than tmpKeep, by the wall clock), in the state directory
// itself or any directory under it; every kept announcement whose life
// ended more than seenKeep ago, with its shown marker and its pusher; every
// other shown marker older than shownKeep (a block's file of the lines one
// prompt showed, a line's own marker an earlier build kept, or an
// announcement's whose announcement is no longer kept); every doubt the reader no longer shows (found more than a day ago, or
// not reading); and the read mark of every status branch not among refs, the
// branches this checkout holds. It runs inside the read, so it stops when ctx
// ends, and what it did not reach is pruned by a later intake.
func pruneStatus(ctx context.Context, root string, now time.Time, refs []statusRef) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	for _, sub := range []string{"", seenDir, shownDir, doubtDir, pusherDir, path.Base(statusBranches.readDir())} {
		for _, e := range statusEntries(root, sub) {
			if ctx.Err() != nil {
				return
			}
			name := e.Name()
			if !strings.HasPrefix(name, tmpPrefix) && !strings.HasPrefix(name, ".abcd-tmp-") {
				continue
			}
			// By the wall clock: a write running beside this one renames its
			// file within moments, so only one older than tmpKeep was cut off.
			if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > tmpKeep {
				_ = r.Remove(statusSub(sub) + "/" + name)
			}
		}
	}
	for _, e := range statusEntries(root, seenDir) {
		if ctx.Err() != nil {
			return
		}
		c := e.Name()
		if !gitutil.IsFullSHA(c) {
			continue
		}
		data := readStatusFile(root, seenDir+"/"+c, 4096)
		var s seenEntry
		if data == nil || json.Unmarshal(data, &s) != nil {
			continue
		}
		first, err := time.Parse(time.RFC3339, s.FirstFetched)
		if err != nil || now.Sub(first) < MaxAnnouncementLife+seenKeep {
			continue
		}
		for _, sub := range []string{seenDir, shownDir, pusherDir} {
			_ = r.Remove(statusFile(sub + "/" + c))
		}
	}
	for _, e := range statusEntries(root, shownDir) {
		if ctx.Err() != nil {
			return
		}
		if gitutil.IsFullSHA(e.Name()) && seenKept(root, e.Name()) {
			continue
		}
		if info, err := e.Info(); err == nil && now.Sub(info.ModTime()) > shownKeep {
			_ = r.Remove(statusFile(shownDir + "/" + e.Name()))
		}
	}
	for _, e := range statusEntries(root, doubtDir) {
		if ctx.Err() != nil {
			return
		}
		data := readStatusFile(root, doubtDir+"/"+e.Name(), 4096)
		var d struct {
			Branch  string `json:"branch"`
			FoundAt string `json:"found_at"`
		}
		if data != nil && json.Unmarshal(data, &d) == nil && statusName(d.Branch) {
			if found, err := time.Parse(time.RFC3339, d.FoundAt); err == nil && now.Sub(found) < MaxAnnouncementLife {
				continue
			}
		}
		_ = r.Remove(statusFile(doubtDir + "/" + e.Name()))
	}
	held := map[string]bool{}
	for _, ref := range refs {
		held[path.Base(statusBranches.markRel(ref.ref))] = true
	}
	for _, e := range statusEntries(root, path.Base(statusBranches.readDir())) {
		if ctx.Err() != nil {
			return
		}
		if !held[e.Name()] {
			_ = r.Remove(statusBranches.readDir() + "/" + e.Name())
		}
	}
}

// noteFetchFailure keeps, best effort, that a fetch of ns failed at now, and
// git's reason, for every reader to say (C10); each namespace keeps its own.
// A checkout without its local tier keeps nothing.
func noteFetchFailure(root string, ns namespace, now time.Time, reason string) {
	if tierPresent(root) != nil {
		return
	}
	reason = strings.Join(strings.Fields(termsafe.Sanitize(reason)), " ")
	if r := []rune(reason); len(r) > 400 {
		reason = string(r[:397]) + "..."
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	if err := mkStateDir(r, ns.stateDir); err != nil {
		return
	}
	_ = fsutil.WriteFileAtomicInRoot(r, ns.stateDir+"/"+failedRel, []byte(stamp(now)+"\n"+reason+"\n"), stateMode)
}

// clearFetchFailure removes ns's kept failure after a fetch that succeeded.
func clearFetchFailure(root string, ns namespace) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	_ = r.Remove(ns.stateDir + "/" + failedRel)
}

// unreadOf is ns's failed fetch as kept, nil when none is: a failure kept
// before the namespace's last stamped fetch is passed over.
func unreadOf(root string, ns namespace) *Unread {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, ns.stateDir+"/"+failedRel, 1024)
	if err != nil {
		return nil
	}
	at, reason, _ := strings.Cut(strings.TrimRight(string(data), "\n"), "\n")
	failed, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return nil
	}
	since := lastFetched(root, ns)
	if since.After(failed) {
		return nil
	}
	return &Unread{Since: since, FailedAt: failed.UTC(), Reason: termsafe.Sanitize(reason)}
}
