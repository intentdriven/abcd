package claim

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
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

// PusherLookup asks the forge which account pushed each update to the status
// branch abcd-status/<branch> on origin (C17): the answer maps each commit
// that a push made the branch's tip to the pusher's login. A commit the answer
// does not name is one the forge cannot say about, and its sender stays
// claimed; an error leaves every sender claimed, with the reason. The front
// door wires it to the forge adapter; nil asks nothing.
type PusherLookup func(root, branch string) (map[string]string, error)

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
	// UnreadNames counts the status branches whose names are not read: no
	// such name is carried, since a pusher chose it (unreadNames keys the
	// line that counts them, and never leaves this package).
	UnreadNames int
	unreadNames []string
	// Reservations is the reservations as last fetched (View), and Holds every
	// release being cut, judged as the merge hold judges them (JudgedHolds).
	Reservations Snapshot
	Holds        []Hold
}

// StatusFetch is what `abcd status`'s fetch could not read: Gap is why
// origin's default branch could not be read, and SendersUnchecked why the
// forge could not be asked who pushed, each "" when it could.
type StatusFetch struct {
	Gap              string
	SendersUnchecked string
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
// and answers its *FetchError.
func FetchStatus(root string, lookup PusherLookup, o Options) (StatusFetch, error) {
	now := o.now()
	ok, err := HasOrigin(root)
	if err != nil {
		return StatusFetch{}, refuse("could not list this checkout's remotes: %s; nothing was fetched", redacted(err))
	}
	if !ok {
		return StatusFetch{}, ErrNoOrigin
	}
	names, skipped, err := listStatusBranches(root)
	if err != nil {
		noteFetchFailure(root, statusBranches, now, redacted(err))
		return StatusFetch{}, &FetchError{Reason: redacted(err), what: statusBranches.noun, outcome: "nothing was fetched"}
	}
	status := statusBranches
	status.only = names
	gap, err := fetch(root, now, "nothing was fetched", reservations, status)
	if err != nil {
		return StatusFetch{}, err
	}
	keepSkipped(root, skipped, now)
	keepFirstFetched(root, now, nil)
	var res StatusFetch
	res.Gap = gap
	if _, err := takeIn(context.Background(), root, now); err != nil {
		// The reader takes it in again and names what it could not read.
		return res, nil
	}
	if lookup != nil {
		res.SendersUnchecked = checkSenders(root, now, lookup)
	}
	return res, nil
}

// listStatusBranches asks origin which status branches it holds (`git
// ls-remote origin refs/heads/abcd-status/*`): the names that are lower-case
// GitHub usernames, sorted, and every other name, which is not fetched.
func listStatusBranches(root string) (names, skipped []string, err error) {
	args := []string{"ls-remote", "--refs", Remote, StatusRefPrefix + "*"}
	trace(args...)
	out, err := gitutil.Net(root, args...)
	if err != nil {
		return nil, nil, errors.New(redacted(err))
	}
	names = []string{}
	for _, line := range strings.Split(out, "\n") {
		sha, ref, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || !gitutil.IsFullSHA(sha) {
			continue
		}
		name, ok := strings.CutPrefix(ref, StatusRefPrefix)
		if !ok || name == "" {
			continue
		}
		if statusName(name) {
			names = append(names, name)
		} else {
			skipped = append(skipped, name)
		}
	}
	sort.Strings(names)
	sort.Strings(skipped)
	return names, skipped, nil
}

// statusName reports whether name, after abcd-status/, is a branch the reader
// reads: a GitHub username in lower case.
func statusName(name string) bool { return validUsername(name) && name == strings.ToLower(name) }

// ReadStatus is everyone's status as this checkout last fetched it, from
// disk alone: no fetch and no push (invariant 7). It first takes in whatever a
// fetch brought since the last read, under ctx, then reads what is kept. A ctx
// that ends first answers its error.
func ReadStatus(ctx context.Context, root string, now time.Time, o Options) (StatusView, error) {
	return readStatus(ctx, root, now, o, true)
}

// readStatus is ReadStatus, reading the reservations themselves only when
// full is true: the prompt hook shows the release holds alone, and reads no
// other reservation's tip.
func readStatus(ctx context.Context, root string, now time.Time, o Options, full bool) (StatusView, error) {
	now = now.UTC().Truncate(time.Second)
	ok, err := HasOrigin(root)
	if err != nil {
		return StatusView{}, err
	}
	if !ok {
		return StatusView{}, nil
	}
	v := StatusView{Origin: true, FetchedAt: lastFetched(root, statusBranches)}
	v.Unread = unreadOf(root, statusBranches)
	v.ReservationsUnread = unreadOf(root, reservations)
	if _, err := takeIn(ctx, root, now); err != nil {
		return StatusView{}, err
	}
	v.Announcements = keptAnnouncements(root, now)
	for _, d := range keptDoubts(root, now) {
		if statusName(d.Branch) {
			v.Doubts = append(v.Doubts, d)
		} else {
			v.unreadNames = append(v.unreadNames, d.Branch)
		}
	}
	v.UnreadNames = len(v.unreadNames)
	if full {
		if v.Reservations, err = View(root, now); err != nil {
			return StatusView{}, err
		}
	}
	if v.Holds, err = JudgedHolds(root, now, o); err != nil {
		return StatusView{}, err
	}
	return v, ctx.Err()
}

// intake is what one intake found.
type intake struct {
	// taken is how many commits it read.
	taken int
}

// takeIn reads every status branch this checkout holds from where it last
// read it (namespace.since, under ctx), reads each commit the walk added and
// this checkout has not read before, in one bounded batch, keeps each as seen
// at now, then marks the branch read at its tip. A branch whose name is not a
// lower-case username, whose tip is not a commit, or whose walk could not
// verify continuity (it moved back or was rewritten, ran past the walk's
// limit since the mark, or passed a merge) is kept as doubtful, and every
// other branch is still read. It writes only the local tier; a checkout
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
	tree := ""
	for _, r := range refs {
		name := strings.TrimPrefix(r.ref, StatusTrackingPrefix)
		if !statusName(name) {
			keepDoubt(root, BranchDoubt{Branch: name, Reason: DoubtBranchName, Tip: r.sha, FoundAt: now})
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
			keepDoubt(root, BranchDoubt{Branch: name, Reason: "its tip " + h.Unreadable, Tip: h.Tip, FoundAt: now})
			continue
		}
		switch {
		case h.Since != "" && h.Rewritten:
			keepDoubt(root, BranchDoubt{Branch: name, Reason: DoubtRewritten, Tip: h.Tip, FoundAt: now})
		case h.Since != "" && h.More:
			keepDoubt(root, BranchDoubt{Branch: name, Reason: DoubtMore, Tip: h.Tip, FoundAt: now})
		case h.Merged:
			keepDoubt(root, BranchDoubt{Branch: name, Reason: DoubtMerged, Tip: h.Tip, FoundAt: now})
		}
		var fresh []string
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
			read, err := readAnnouncements(ctx, root, tree, fresh)
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
		pruneStatus(root, now)
	}
	return in, nil
}

// statusRef is one status branch this checkout holds, as for-each-ref names
// it.
type statusRef struct{ ref, sha string }

// statusRefs is every status branch this checkout holds, read in one command
// as stored.
func statusRefs(ctx context.Context, root string) ([]statusRef, error) {
	out, err := storedGitCtx(ctx, root, 1<<20, "for-each-ref", "--format=%(refname) %(objectname)", StatusTrackingPrefix)
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

// keptAnnouncements is every announcement kept here that still stands at
// now, and every unreadable one first read in the last day, newest first: by
// when this checkout first read it, then, on one branch, by the order the
// branch holds them in, and across branches by the time each carries. Each kept
// file is read back through the same grammar its fields were held to, so a
// file that does not read is passed over.
func keptAnnouncements(root string, now time.Time) []Announcement {
	var out []Announcement
	for _, e := range statusEntries(root, seenDir) {
		c := e.Name()
		if !gitutil.IsFullSHA(c) || !e.Type().IsRegular() {
			continue
		}
		a, ok := keptAnnouncement(root, c)
		if !ok {
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
	return out
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
		if k.Branch == boundedName(d.Branch) && k.Reason == d.Reason && k.Tip == d.Tip && !k.FoundAt.IsZero() && k.Tip != "" {
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

// maxNameRunes bounds a branch name a doubt keeps and a render shows.
const maxNameRunes = 80

// boundedName is a branch name terminal-safe and bounded, for keeping and
// showing.
func boundedName(n string) string {
	n = termsafe.Sanitize(n)
	if r := []rune(n); len(r) > maxNameRunes {
		n = string(r[:maxNameRunes-3]) + "..."
	}
	return n
}

// keptDoubts is every status branch kept as doubtful: each found in the last
// day by an intake, and each name origin's last listing held that is not
// read, sorted by name.
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
		if data == nil || json.Unmarshal(data, &d) != nil || e.Name() != hashName(d.Branch) {
			continue
		}
		found, err := time.Parse(time.RFC3339, d.FoundAt)
		if err != nil || now.Sub(found) >= MaxAnnouncementLife {
			continue
		}
		if d.Tip != "" && !gitutil.IsFullSHA(d.Tip) {
			continue
		}
		byName[d.Branch] = BranchDoubt{Branch: boundedName(d.Branch), Reason: doubtReason(d.Reason), Tip: d.Tip, FoundAt: found.UTC()}
	}
	for _, s := range keptSkipped(root) {
		if _, ok := byName[s.Branch]; !ok {
			byName[s.Branch] = s
		}
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
	case DoubtBranchName, DoubtRewritten, DoubtMore, DoubtMerged, "its tip " + reasonNoObject:
		return reason
	}
	return "it could not be read"
}

// keepSkipped keeps the names origin's listing held that are not read,
// replacing the last listing's.
func keepSkipped(root string, names []string, now time.Time) {
	kept := make([]string, 0, len(names))
	for _, n := range names {
		kept = append(kept, boundedName(n))
	}
	data, err := json.Marshal(struct {
		ListedAt string   `json:"listed_at"`
		Names    []string `json:"names"`
	}{stamp(now), kept})
	if err != nil {
		return
	}
	writeStatusFile(root, "", skippedRel, data)
}

// keptSkipped is origin's last listing's names that are not read, as doubts.
func keptSkipped(root string) []BranchDoubt {
	data := readStatusFile(root, skippedRel, 64<<10)
	var s struct {
		ListedAt string   `json:"listed_at"`
		Names    []string `json:"names"`
	}
	if data == nil || json.Unmarshal(data, &s) != nil {
		return nil
	}
	at, _ := time.Parse(time.RFC3339, s.ListedAt)
	var out []BranchDoubt
	for _, n := range s.Names {
		out = append(out, BranchDoubt{Branch: boundedName(n), Reason: DoubtBranchName, FoundAt: at.UTC()})
	}
	return out
}

// pusherVerdict is what the forge said of who pushed one announcement.
type pusherVerdict struct {
	known bool
	login string
}

// pusherKept is the forge's answer kept for commit.
func pusherKept(root, commit string) pusherVerdict {
	data := readStatusFile(root, pusherDir+"/"+commit, 128)
	login := strings.TrimSpace(string(data))
	if data == nil || login == "" {
		return pusherVerdict{}
	}
	return pusherVerdict{known: true, login: login}
}

// checkSenders asks lookup, once per branch, who pushed each standing
// announcement whose pusher is not kept here, and keeps each login the
// answer names that is a valid username (any other answer is kept as a
// login that is not the owner's). It returns why the forge could not be
// asked, "" when it could.
func checkSenders(root string, now time.Time, lookup PusherLookup) string {
	ask := map[string][]string{}
	for _, a := range keptAnnouncements(root, now) {
		if a.Unreadable == "" && a.Standing(now) && !pusherKept(root, a.Commit).known {
			ask[a.Branch] = append(ask[a.Branch], a.Commit)
		}
	}
	branches := make([]string, 0, len(ask))
	for b := range ask {
		branches = append(branches, b)
	}
	sort.Strings(branches)
	var failed []string
	for _, b := range branches {
		pushed, err := lookup(root, b)
		if err != nil {
			failed = append(failed, "abcd-status/"+b+": "+termsafe.Sanitize(redacted(err)))
			continue
		}
		for _, c := range ask[b] {
			login, ok := pushed[c]
			if !ok {
				continue
			}
			if !validUsername(login) {
				login = "-"
			}
			writeStatusFile(root, pusherDir, c, []byte(login+"\n"))
		}
	}
	return strings.Join(failed, "; ")
}

// pruneStatus removes, best effort, every kept announcement whose life ended
// more than seenKeep ago, with its shown marker and its pusher, and every
// other shown marker older than shownKeep.
func pruneStatus(root string, now time.Time) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return
	}
	defer r.Close()
	for _, e := range statusEntries(root, seenDir) {
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
		if gitutil.IsFullSHA(e.Name()) {
			continue
		}
		if info, err := e.Info(); err == nil && now.Sub(info.ModTime()) > shownKeep {
			_ = r.Remove(statusFile(shownDir + "/" + e.Name()))
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
