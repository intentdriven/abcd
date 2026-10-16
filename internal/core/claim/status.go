package claim

import (
	"context"
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
	DoubtBranchName = "its name is not a lower-case GitHub username, so it is not read"
	DoubtRewritten  = "continuity not verified: the branch was rewritten rather than added to, so an announcement may be missing or shown twice"
	DoubtMore       = "continuity not verified: more was added than one read covers, so an announcement may be missing or shown twice"
	DoubtMerged     = "continuity not verified: a merge was passed, and what it brought in is not read"
	DoubtSender     = "it names a sender other than the branch's owner"
	DoubtPusher     = "the forge's record names another account as its pusher"
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
// made the branch) and to (After), and the account that pushed.
type Push struct {
	Before, After, Pusher string
}

// PusherLookup asks the forge who pushed each update to the status branch
// abcd-status/<branch> on origin (C17): its record of pushes, newest first. A
// commit no push in the answer covers is one the forge cannot say about, and
// its sender stays claimed; an error leaves every sender claimed, with the
// reason. The front door wires it to the forge adapter; nil asks nothing.
type PusherLookup func(root, branch string) ([]Push, error)

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
	// Branch is the name after abcd-status/, as origin or this checkout holds
	// it; a render quotes it, since it may be any name git admits.
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
	// Holds is every release being cut, judged as the merge hold judges them
	// (JudgedHolds).
	Holds []Hold
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
// (decision 10), so it is kept by name for the reader to show as doubtful
// (C11). Then it reads in what the fetch brought (the intake), and asks the
// forge, through lookup when it is given, who pushed each standing
// announcement whose pusher is not yet known here (C17). A checkout with no
// remote named origin answers ErrNoOrigin. A listing or a fetch that fails is
// kept in the local tier, its time and reason, for every reader to say (C10),
// and answers its *FetchError. Before the fetch, every tracking ref under the
// status namespace that is not among the names it fetches is deleted
// (dropUnlisted): a branch deleted on origin, one whose name is not read, or
// one past the cap leaves nothing to read; a deletion that fails twice is
// said in NotDropped, and the fetch goes on.
func FetchStatus(root string, lookup PusherLookup, o Options) (StatusFetch, error) {
	now := o.now()
	ok, err := HasOrigin(root)
	if err != nil {
		return StatusFetch{}, refuse("could not list this checkout's remotes: %s; nothing was fetched", redacted(err))
	}
	if !ok {
		return StatusFetch{}, ErrNoOrigin
	}
	names, skipped, past, err := listStatusBranches(root)
	if err != nil {
		noteFetchFailure(root, statusBranches, now, redacted(err))
		return StatusFetch{}, &FetchError{Reason: redacted(err), what: statusBranches.noun, outcome: "nothing was fetched"}
	}
	// Only the names origin lists, and of them only those the fetch reads,
	// are kept: the named refspecs prune nothing, and a branch left from an
	// earlier fetch would read as fetched now.
	notDropped := statusBranches.dropUnlisted(root, names)
	status := statusBranches
	status.only = names
	gap, err := fetch(root, now, "nothing was fetched", reservations, status)
	if err != nil {
		return StatusFetch{}, err
	}
	keepSkipped(root, skipped, past, now)
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
// not fetched; and the tally of every other name, which is not fetched and
// is never kept.
func listStatusBranches(root string) (names []string, skipped nameTally, past int, err error) {
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
// holds `abcd status` and the prompt hook make: neither reads them again. A
// read that fails, or a ctx that ends first, answers its error with what was
// read before it: the view's origin, its fetch's date and its failed fetches
// always, and the holds when they were read.
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
	if v.Holds, err = JudgedHolds(root, now, o); err != nil {
		return v, err
	}
	in, err := takeIn(ctx, root, now)
	if err != nil {
		return v, err
	}
	v.Announcements, v.AnnouncementsNotRead = keptAnnouncementsCounted(root, now, in.held)
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
	return v, ctx.Err()
}

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
}

// takeIn reads every status branch this checkout holds from where it last
// read it (namespace.since, under ctx), reads each commit the walk added and
// this checkout has not read before, in one bounded batch, keeps each as seen
// at now, then marks the branch read at its tip. A branch whose name is not a
// lower-case username is tallied, never read and never kept. One whose tip is
// not a commit, or whose walk could not verify continuity (it moved back or
// was rewritten, ran past the walk's limit since the mark, or on a first read
// while something beneath what it read could still stand, or passed a merge)
// is kept as doubtful, and every other branch is still read. It writes only
// the local tier; a checkout
// without one keeps nothing and reads nothing. When ctx ends, what was read
// is kept, the branch at hand is not marked, and ctx's error is answered.
func takeIn(ctx context.Context, root string, now time.Time) (intake, error) {
	var in intake
	if tierPresent(root) != nil {
		return in, nil
	}
	refs, err := statusRefs(ctx, root)
	if err != nil {
		return in, err
	}
	in.held = map[string]bool{}
	for _, r := range refs {
		in.held[strings.TrimPrefix(r.ref, StatusTrackingPrefix)] = true
	}
	tree := ""
	taken := 0
	for _, r := range refs {
		name := strings.TrimPrefix(r.ref, StatusTrackingPrefix)
		if !statusName(name) {
			in.unread.add(name)
			continue
		}
		if taken++; taken > maxStatusBranches {
			in.pastCap++
			continue
		}
		if statusBranches.lastRead(root, r.ref) == r.sha {
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
		switch {
		case h.Since != "" && h.Rewritten:
			keepDoubt(root, BranchDoubt{Branch: name, Reason: DoubtRewritten, Tip: h.Tip, FoundAt: now})
		case h.More && h.Since != "":
			keepDoubt(root, BranchDoubt{Branch: name, Reason: DoubtMore, Tip: h.Tip, FoundAt: now})
		case h.Merged:
			keepDoubt(root, BranchDoubt{Branch: name, Reason: DoubtMerged, Tip: h.Tip, FoundAt: now})
		}
		var fresh []string
		var read map[string]readAnnouncement
		for _, c := range h.Added {
			if !seenKept(root, c) {
				fresh = append(fresh, c)
			}
		}
		if len(fresh) > 0 {
			if tree == "" {
				if tree, err = emptyTree(root); err != nil {
					return in, err
				}
			}
			read, err = readAnnouncements(ctx, root, tree, fresh)
			if err != nil {
				return in, err
			}
			// fresh is newest first, so the newest takes the largest order;
			// the base is the wall clock's, so a later intake orders after.
			base := time.Now().UnixNano()
			for i, c := range fresh {
				keepSeen(root, name, c, read[c], now, base+int64(len(fresh)-1-i))
				in.taken++
			}
		}
		if h.More && h.Since == "" && mayStandBeneath(root, read, h.Added[len(h.Added)-1], now) {
			// A first read: a live announcement may sit under more than one
			// read covers (dcn-2610161100416718), but only while what lies
			// beneath the oldest commit read could still stand.
			keepDoubt(root, BranchDoubt{Branch: name, Reason: DoubtMore, Tip: h.Tip, FoundAt: now})
		}
		if err := ctx.Err(); err != nil {
			return in, err
		}
		if err := statusBranches.markRead(root, r.ref, h.Tip); err != nil {
			return in, err
		}
	}
	// Pruning reads every kept file, so it runs only when something new was
	// read in, never at every prompt.
	if in.taken > 0 {
		pruneStatus(ctx, root, now, refs)
	}
	return in, nil
}

// mayStandBeneath reports whether a commit beneath oldest, the oldest commit a
// first read took in, could still stand at now. A branch only grows, so each
// commit beneath was sent before oldest, and an announcement stands at most
// MaxAnnouncementLife, counted from its first fetch only while its own expiry
// is less than announcementSlack past (C9): when oldest was sent, by the time
// it carries, more than both ago, nothing beneath it can stand. The same slack
// covers a sender's clock that ran a little ahead of an earlier one. When
// oldest does not read as an announcement, what lies beneath it cannot be
// dated, and it may stand. read is the batch this intake read; oldest is
// looked up in what is kept when it is not there.
func mayStandBeneath(root string, read map[string]readAnnouncement, oldest string, now time.Time) bool {
	var sent time.Time
	if r, ok := read[oldest]; ok {
		if r.h.Unreadable != "" {
			return true
		}
		sent = r.a.SentAt
	} else {
		data := readStatusFile(root, seenDir+"/"+oldest, 4096)
		var e seenEntry
		if data == nil || json.Unmarshal(data, &e) != nil || e.Unreadable != "" {
			return true
		}
		t, err := time.Parse(time.RFC3339, e.SentAt)
		if err != nil {
			return true
		}
		sent = t
	}
	return now.Before(sent.Add(MaxAnnouncementLife + announcementSlack))
}

// statusRef is one status branch this checkout holds, as for-each-ref names
// it.
type statusRef struct{ ref, sha string }

// statusRefs is every status branch this checkout holds, in git's ref order,
// read in one command as stored, under ctx. A fetch of the whole namespace
// (`abcd announce`'s) may bring in as many as origin holds, so the read is
// bounded only as dropUnlisted's is; an intake reads at most
// maxStatusBranches of them.
func statusRefs(ctx context.Context, root string) ([]statusRef, error) {
	out, err := storedGitCtx(ctx, root, 16<<20, "for-each-ref", "--format=%(refname) %(objectname)", StatusTrackingPrefix)
	if err != nil {
		return nil, fmt.Errorf("reading the status branches: %s", redacted(err))
	}
	var refs []statusRef
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) != 2 || !strings.HasPrefix(f[0], StatusTrackingPrefix) || !gitutil.IsFullSHA(f[1]) {
			continue
		}
		refs = append(refs, statusRef{ref: f[0], sha: f[1]})
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

// seenKept reports whether commit was already read here.
func seenKept(root, commit string) bool {
	r, err := os.OpenRoot(root)
	if err != nil {
		return false
	}
	defer r.Close()
	_, err = r.Lstat(statusFile(seenDir + "/" + commit))
	return err == nil
}

// keepSeen keeps commit, read on branch, as first fetched at now, best
// effort: one not kept is read again by the next intake.
func keepSeen(root, branch, commit string, read readAnnouncement, now time.Time, order int64) {
	e := seenEntry{Branch: branch, FirstFetched: stamp(now), Order: order}
	if read.h.Unreadable != "" {
		e.Unreadable = read.h.Unreadable
	} else {
		a := read.a
		e.Kind, e.Item, e.Sender, e.SentAt, e.ExpiresAt = a.Kind, a.Item, a.Sender, stamp(a.SentAt), stamp(a.ExpiresAt)
	}
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	writeStatusFile(root, seenDir, commit, data)
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
// sub; nil when it cannot be read.
func statusEntries(root, sub string) []fs.DirEntry {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil
	}
	defer r.Close()
	f, err := r.Open(statusFile(sub))
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

// keepDoubt keeps d, replacing what was kept for its branch. A doubt kept
// already, for the same reason at the same tip, is left as it is, so a doubt
// read again at every prompt is not written again at every prompt.
func keepDoubt(root string, d BranchDoubt) {
	for _, k := range keptDoubts(root, d.FoundAt) {
		if k.Branch == d.Branch && k.Reason == d.Reason && k.Tip == d.Tip && !k.FoundAt.IsZero() && k.Tip != "" {
			return
		}
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
	writeStatusFile(root, doubtDir, hashName(d.Branch), data)
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
	byName := map[string]BranchDoubt{}
	for _, e := range statusEntries(root, doubtDir) {
		data := readStatusFile(root, doubtDir+"/"+e.Name(), 4096)
		var d struct {
			Branch  string `json:"branch"`
			Reason  string `json:"reason"`
			Tip     string `json:"tip"`
			FoundAt string `json:"found_at"`
		}
		if data == nil || json.Unmarshal(data, &d) != nil || e.Name() != hashName(d.Branch) || !statusName(d.Branch) {
			continue
		}
		found, err := time.Parse(time.RFC3339, d.FoundAt)
		if err != nil || now.Sub(found) >= MaxAnnouncementLife {
			continue
		}
		if d.Tip != "" && !gitutil.IsFullSHA(d.Tip) {
			continue
		}
		byName[d.Branch] = BranchDoubt{Branch: d.Branch, Reason: doubtReason(d.Reason), Tip: d.Tip, FoundAt: found.UTC()}
	}
	out := make([]BranchDoubt, 0, len(byName))
	for _, d := range byName {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Branch < out[j].Branch })
	return out
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
	// maxStatusBranches, none of them fetched.
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
// the zero state when nothing here listed origin, and an error when what was
// kept cannot be read back, so a reader says the count is unknown rather than
// count none.
func keptSkipped(root string) (skippedState, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return skippedState{}, err
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, statusFile(skippedRel), 4096)
	if errors.Is(err, fs.ErrNotExist) {
		return skippedState{}, nil
	}
	if err != nil {
		return skippedState{}, err
	}
	var s skippedState
	if err := json.Unmarshal(data, &s); err != nil || s.Count < 0 || s.PastCap < 0 {
		return skippedState{}, errors.New("the count of status branch names not read does not parse")
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
// place each answer: past it, no further branch is asked about, a push not
// yet placed keeps nothing, and the check says so.
var pusherBudget = 10 * time.Second

// maxAskedBranches is the most status branches one fetch asks the forge
// about, the newest first: the rest are counted and asked by a later fetch.
const maxAskedBranches = 10

// checkSenders asks lookup, once per branch, who pushed each standing
// announcement the forge was not yet asked about here, and keeps each login
// the answer names that is a valid username (any other answer is kept as a
// login that is not the owner's), and that it named none (noPusher). At most
// maxAskedBranches branches are asked, the one whose newest such
// announcement is newest first, and none once pusherBudget has passed; the
// rest are counted, and asked by a later fetch. A push names only the tip it left, so each
// is placed by walking the first-parent line from that tip down to the tip
// it moved from, at most the walk's limit of commits, under pusherBudget:
// every commit of the push is judged by its pusher (placePushes). A commit two
// pushes cover (a rewritten branch) takes a pusher other than the owner over
// the owner. A branch whose record could not be placed in full keeps nothing
// of it. It returns why the forge could not be asked or its record could not
// be placed in full, "" when it could.
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
	ctx, cancel := context.WithTimeout(context.Background(), pusherBudget)
	defer cancel()
	var failed []string
	unasked := 0
	for i, b := range branches {
		if i >= maxAskedBranches || ctx.Err() != nil {
			unasked = len(branches) - i
			break
		}
		pushed, err := lookup(root, b)
		if err != nil {
			failed = append(failed, "abcd-status/"+b+": "+termsafe.Sanitize(redacted(err)))
			continue
		}
		byCommit, err := placePushes(ctx, root, b, pushed)
		if err != nil {
			failed = append(failed, "abcd-status/"+b+": the forge's record of pushes was not placed in full: "+termsafe.Sanitize(redacted(err)))
			continue
		}
		for _, c := range ask[b] {
			login, ok := byCommit[c]
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
	if unasked > 0 {
		what := fmt.Sprintf("%d status branches not asked about", unasked)
		if unasked == 1 {
			what = "1 status branch not asked about"
		}
		failed = append(failed, fmt.Sprintf("%s (one fetch asks about at most %d, the newest first, within %s); the next `abcd status` asks them",
			what, maxAskedBranches, pusherBudget))
	}
	return strings.Join(failed, "; ")
}

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
// A push whose After this checkout does not hold is passed over; one that
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
		if !gitutil.IsFullSHA(p.After) {
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

// pruneStatus removes, best effort: every kept announcement whose life
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
