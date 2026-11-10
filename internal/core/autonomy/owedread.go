package autonomy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// StaleWindow bounds ownership through the reflog or a recorded creation: such
// a branch is the session's own only while it holds a commit of this
// checkout's made within it, or a live reservation links it
// (dcn-2610181003463367).
const StaleWindow = 14 * 24 * time.Hour

// ReadOptions carry what ReadOwed reads; every seam's zero value reads the
// machine.
type ReadOptions struct {
	// Root is the session's working copy: the root of the worktree the
	// session started in.
	Root string
	// Record is the away record as the stop check read it, nil when there is
	// none. The forge is read only while it holds at Now; its queue and
	// HoldPushes decide what a queued push hands back.
	Record *Record
	// Home is the home directory the away store lives under; "" means the
	// process's own.
	Home string
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// Forge runs the forge client; nil means landing.GHContext.
	Forge Forge
	// ForgeDeadline bounds the listing; zero means ForgeDeadline.
	ForgeDeadline time.Duration
	// RulesetDeadline bounds each read that decides whether a merge queue
	// keeps a behind pull request current (forge.go): the default branch's
	// live rules, and its armed or queued pull requests; zero means
	// RulesetDeadline.
	RulesetDeadline time.Duration
	// Reservations reads the reservations as last fetched, with no network;
	// nil means claim.View.
	Reservations func(root string, now time.Time) (claim.Snapshot, error)
	// Person is the person's forge name, which tells their pull requests
	// from another person's; "" means the holder name reservations resolve
	// without a network read (claim.ReaderHolder), and when that is unknown
	// too the listing asks for the person's own pull requests alone.
	Person string
	// Email is this checkout's author email, which tells its commits from
	// another author's; "" means the email git would author a commit with.
	Email string
	// SessionID is the stopping session's id, as the Stop payload carries it.
	// With one ValidSessionID accepts, an item another live session holds is
	// left out of the owed work and named in a note (holders.go); "" leaves
	// every item owed as this copy's, as does any item whose holder the
	// evidence cannot tell.
	SessionID string
	// LiveSessions reads the sessions running on the machine, each with the
	// checkout it started in; nil means occupancy.Live under Home with the
	// machine's process table. It is read at most once a check, and only
	// when an item may be another session's.
	LiveSessions func() ([]occupancy.Session, error)
}

func (o ReadOptions) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// ReadOwed is the owed work of the working copy at o.Root
// (spc-2610180914469489, "Owed work"): the stop check's one reading.
//
// While the person is present (o.Record does not hold) it reads disk only:
// renewals due and unpushed own branches, and never calls the forge. While
// away it adds one listing of the open pull requests under ForgeDeadline,
// for failing, behind, stacked and uncarried work; a listing that fails or
// is late is named in Owed.Notes and the check judges from disk alone
// (adr-2610181003541997). A behind pull request into the default branch is
// judged by its merge queue, read from the forge's live rules under
// RulesetDeadline (queueKeepsCurrent). With --hold-pushes, an unpushed own
// branch a queued step names, whose head contains its base's last-fetched tip
// and has a
// preflight receipt, is handed back: moved out of Owed.Items into
// Owed.HandedBack, for the verdict to accept and the stop hook to record.
// Owed.NextRenewal is the earliest renewal due among the live reservations
// under this copy's key, due or not yet, present or away.
//
// The error is a root that is not a working copy; every other failure to read
// something is a note, and the set is judged without it.
func ReadOwed(ctx context.Context, o ReadOptions) (Owed, error) {
	if o.Root == "" {
		return Owed{}, errors.New("autonomy: no working copy to read the owed work of")
	}
	root, err := gitutil.CheckoutRootContext(ctx, o.Root, StoreName)
	if err != nil {
		return Owed{}, err
	}
	home := o.Home
	if home == "" {
		if home, err = os.UserHomeDir(); err != nil {
			home = ""
		}
	}
	r := &owedReader{
		ctx:      ctx,
		root:     root,
		home:     home,
		now:      o.now(),
		email:    o.Email,
		rec:      o.Record,
		me:       o.SessionID,
		liveRead: o.LiveSessions,
		heldRes:  map[string]string{},
	}
	if r.liveRead == nil {
		r.liveRead = func() ([]occupancy.Session, error) {
			return occupancy.Live(home, sessionname.Live().ReadTable)
		}
	}
	r.away = o.Record.Holds(r.now)
	if r.email == "" {
		if id, err := identity.EffectiveIdentityContext(ctx, root); err == nil {
			r.email = id.Email
		}
	}
	r.readRefs()
	r.readReservations(o)
	if r.away {
		r.person = o.Person
		if r.person == "" {
			r.person = claim.ReaderHolderContext(ctx, root, claim.Options{Home: home})
		}
		forge := o.Forge
		if forge == nil {
			forge = landing.GHContext
		}
		deadline := o.ForgeDeadline
		if deadline <= 0 {
			deadline = ForgeDeadline
		}
		r.forge, r.rulesDeadline = forge, o.RulesetDeadline
		if r.rulesDeadline <= 0 {
			r.rulesDeadline = RulesetDeadline
		}
		r.list = listOpen(ctx, forge, root, r.person, deadline)
		if !r.list.read {
			r.note("the pull requests were not read (%s), so the check judged from disk alone", r.list.unread)
			// The check's own budget, not the forge's, ended the listing:
			// what it leaves unjudged is a reading cut short.
			if r.expired() {
				r.ranOut("the open pull requests")
			}
		}
		if r.list.truncated {
			r.note("the forge listed %d open pull requests, its limit, so the listing may be truncated: a reservation no listed pull request carries is not counted as uncarried", len(r.list.prs))
		}
	}
	r.receiptsMinted = receiptsMinted(ctx, root)
	r.readLanes()
	r.findOwn()
	r.scope()
	owned := r.ownedRefs()
	r.branchItems()
	r.reservationItems()
	r.handBack()
	r.breachNote()
	if r.cut {
		r.ranOut("the rest of this copy's work")
	}
	sort.SliceStable(r.items, func(i, j int) bool {
		a, b := r.items[i], r.items[j]
		if ka, kb := kindRank(a.Kind), kindRank(b.Kind); ka != kb {
			return ka < kb
		}
		if a.Branch != b.Branch {
			return a.Branch < b.Branch
		}
		return a.Record < b.Record
	})
	return Owed{Items: r.items, Notes: r.notes, NextRenewal: r.nextRenewal, WakeBy: r.wakeBy, HandedBack: r.handed, Incomplete: r.incomplete, Owned: owned}, nil
}

// ownedRefs is the work this session owns as the reading found it, after
// another live session's share is taken out (scope): each own branch, with
// its open pull request while the forge was read (away only), and each live
// reservation under this copy's key that no other live session's run holds.
// They are spelled as a checklist item's owner_refs names them.
func (r *owedReader) ownedRefs() []OwnedRef {
	var out []OwnedRef
	for _, name := range r.ownNames() {
		ref := OwnedRef{Ref: "branch:" + name}
		if b := r.branches[name]; b != nil {
			ref.Changed = b.committed
		}
		if r.list.read {
			if p, ok := r.list.byHead(name, r.person); ok && p.Number > 0 {
				ref.Aliases = []string{"pr:" + strconv.Itoa(p.Number)}
			}
		}
		out = append(out, ref)
	}
	for _, res := range r.mine {
		if r.heldRes[res.Record] != "" {
			continue
		}
		out = append(out, OwnedRef{Ref: res.Record})
	}
	return out
}

// owedReader is one ReadOwed's state.
type owedReader struct {
	ctx        context.Context
	root, home string
	now        time.Time
	away       bool
	email      string
	person     string
	rec        *Record

	def     string // the default branch's full ref as last fetched
	defName string // its branch name
	defTip  string

	branches map[string]*localBranch
	mine     []claim.Reservation
	list     listing
	lanes    map[string]laneState

	// forge and rulesDeadline make the reads that decide whether a merge
	// queue keeps a behind pull request current, each at most once a check
	// and only while away: the default branch's live rules (queue), and
	// its armed or queued pull requests (armedInto).
	forge         Forge
	rulesDeadline time.Duration
	queueRead     bool
	queue         queueRule
	armedRead     bool
	armedInto     armedSet

	// me is the stopping session (ReadOptions.SessionID), and the rest of
	// this block what holders.go reads to tell its work from another live
	// session's.
	me         string
	liveRead   func() ([]occupancy.Session, error)
	live       map[string]string // a live session's id -> the checkout it started in
	liveTried  bool
	liveOK     bool
	recordedBy map[string]string   // a recorded branch -> the session that created it, "" when none was kept
	runDrivers map[string][]string // a live run's key -> each unfinished run's driving session, "" when none
	elsewhere  map[string]string   // a branch -> the other worktree it is checked out in
	heldRes    map[string]string   // a reservation -> the live session whose run it is for
	held       map[string]*holding // by session: what another live session holds

	receiptsMinted bool
	own            map[string]bool
	baseNames      []string          // the own branches as findOwn found them, before scope: what stackIdx indexes
	stacks         *stackIndex       // read once, for diskStackBase
	incomplete     bool              // the time budget ran out before everything was read
	cut            bool              // a git read failed once the budget had run out
	stackBase      map[string]string // a branch's base's tip on disk, for the hand-back

	items       []OwedItem
	handed      []OwedItem // handed back by a queued step, for the verdict to accept
	nextRenewal time.Time  // the earliest renewal due among r.mine
	wakeBy      time.Time  // the latest a wake may fire for r.mine (Owed.WakeBy)
	notes       []string
}

type localBranch struct {
	name, head   string
	upstream     string
	upstreamGone bool
	// committed is the head commit's committer time, zero when git gave none.
	committed time.Time
}

type laneState struct {
	key     string
	reviews []laneReview
	// driver is the session its run's control record names as the run's
	// driver, "" when it names none.
	driver string
}

type laneReview struct {
	head string
	pass bool
}

func (r *owedReader) note(format string, a ...any) {
	r.notes = append(r.notes, fmt.Sprintf(format, a...))
}

// git runs one git command bound to the check's context: past its deadline
// git is killed and the call fails.
// A failure once the deadline has passed is marked, so a reader that treats
// a failed read as "nothing there" cannot pass a cut-short reading as whole.
func (r *owedReader) git(args ...string) (string, error) {
	out, err := gitutil.RunLimitedContext(r.ctx, r.root, 1<<20, args...)
	if err != nil && r.expired() {
		r.cut = true
	}
	return out, err
}

// ancestor reports whether a is an ancestor of b; false when git cannot say,
// the deadline passed included.
func (r *owedReader) ancestor(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	_, err := r.git("merge-base", "--is-ancestor", a, b)
	return err == nil
}

// expired reports whether the check's time budget has run out.
func (r *owedReader) expired() bool { return r.ctx.Err() != nil }

// ranOut marks the reading incomplete, once noting what was not read: the
// verdict then allows the stop, naming it (spc-2610180914469489, "Diff review
// round 1", major 6).
func (r *owedReader) ranOut(what string) {
	if !r.incomplete {
		r.note("the check ran out of its time budget before reading %s", what)
	}
	r.incomplete = true
}

// readRefs reads the default branch as last fetched and every local branch
// with its head and upstream.
func (r *owedReader) readRefs() {
	r.def = gitutil.DefaultRefContext(r.ctx, r.root)
	switch {
	case strings.HasPrefix(r.def, "refs/remotes/"):
		parts := strings.SplitN(strings.TrimPrefix(r.def, "refs/remotes/"), "/", 2)
		if len(parts) == 2 {
			r.defName = parts[1]
		}
	case strings.HasPrefix(r.def, "refs/heads/"):
		r.defName = strings.TrimPrefix(r.def, "refs/heads/")
	}
	if r.def != "" {
		r.defTip, _ = r.git("rev-parse", "--verify", "--quiet", r.def+"^{commit}")
		r.defTip = strings.TrimSpace(r.defTip)
	}
	r.branches = map[string]*localBranch{}
	out, err := r.git("for-each-ref", "--format=%(refname:short)%00%(objectname)%00%(upstream)%00%(upstream:track)%00%(committerdate:unix)", "refs/heads")
	if err != nil {
		// A listing the deadline cut short is a reading not done, not a
		// copy with no branches: the verdict must fail open and say so.
		if r.expired() {
			r.ranOut("the local branches")
			return
		}
		r.note("the local branches could not be listed")
		return
	}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, "\x00")
		if len(f) != 5 || f[0] == "" {
			continue
		}
		b := &localBranch{name: f[0], head: f[1], upstream: f[2], upstreamGone: strings.Contains(f[3], "gone")}
		if sec, err := strconv.ParseInt(f[4], 10, 64); err == nil && sec > 0 {
			b.committed = time.Unix(sec, 0)
		}
		r.branches[f[0]] = b
	}
}

// readReservations keeps the live reservations under this copy's key.
func (r *owedReader) readReservations(o ReadOptions) {
	view := o.Reservations
	if view == nil {
		view = claim.View
	}
	snap, err := view(r.root, r.now)
	if err != nil {
		r.note("the reservations as last fetched could not be read")
		return
	}
	for _, res := range snap.Reservations {
		if res.Copy == claim.CopyThis && res.State == claim.StateLive {
			r.mine = append(r.mine, res)
		}
	}
}

func (r *owedReader) liveMine(id string) bool {
	for _, res := range r.mine {
		if res.Record == id {
			return true
		}
	}
	return false
}

// receiptsMinted reports whether the repository mints preflight receipts: a
// receipt folder exists in one of its worktrees.
func receiptsMinted(ctx context.Context, root string) bool {
	wts, err := gitutil.ListWorktreesContext(ctx, root, gitutil.MaxWorktreeListing)
	if err != nil {
		return false
	}
	for _, wt := range wts {
		if wt.Bare {
			continue
		}
		if fi, err := os.Lstat(filepath.Join(wt.Path, filepath.FromSlash(landing.ReceiptsRelDir))); err == nil && fi.IsDir() {
			return true
		}
	}
	return false
}

func (r *owedReader) hasReceipt(head string) bool {
	if head == "" {
		return false
	}
	p, err := landing.Receipt(r.root, head)
	return err == nil && p != ""
}

// runIDRe is a run's directory name (loop.ValidRunID's shape).
var runIDRe = regexp.MustCompile(`^run-[0-9]{16}$`)

// runFile is what the owed reading needs of a run's state file, read
// leniently: the loop owns the file, and this reads only its lanes.
type runFile struct {
	Key     string            `json:"key"`
	Pending []json.RawMessage `json:"pending"`
	Lanes   []struct {
		Branch     string `json:"branch"`
		Stage      string `json:"stage"`
		Validation []struct {
			HeadSHA    string `json:"head_sha"`
			Validators []struct {
				Pass bool `json:"pass"`
			} `json:"validators"`
		} `json:"validation"`
	} `json:"lanes"`
}

const runRelDir = ".abcd/.work.local/run"

// readLanes keeps the open lanes of every unfinished run in this copy's local
// tier whose reservation is live under this copy's key.
func (r *owedReader) readLanes() {
	r.lanes = map[string]laneState{}
	r.runDrivers = map[string][]string{}
	rt, err := os.OpenRoot(r.root)
	if err != nil {
		return
	}
	defer rt.Close()
	entries, err := fs.ReadDir(rt.FS(), runRelDir)
	if err != nil {
		return
	}
	// A run git tracks anything in came with a commit, so none of its lanes
	// is this copy's to owe: asked once for every run listed, inside the
	// budget, never once a run.
	var runs []string
	for _, e := range entries {
		if e.IsDir() && runIDRe.MatchString(e.Name()) {
			runs = append(runs, runRelDir+"/"+e.Name())
		}
	}
	if err := localtier.UntrackedEach(r.ctx, r.root, runs...); err != nil {
		if r.expired() {
			r.cut = true
		}
		r.note("the runs' lanes were not read: %v", err)
		return
	}
	for _, e := range entries {
		if !e.IsDir() || !runIDRe.MatchString(e.Name()) {
			continue
		}
		raw, err := fsutil.ReadGuardedInRoot(rt, runRelDir+"/"+e.Name()+"/state.json", 8<<20)
		if err != nil {
			continue
		}
		var run runFile
		if json.Unmarshal(raw, &run) != nil || !r.liveMine(run.Key) {
			continue
		}
		finished := len(run.Pending) == 0
		for _, l := range run.Lanes {
			if l.Stage != "done" && l.Stage != "discarded" {
				finished = false
			}
		}
		if finished {
			continue
		}
		driver := runDriver(rt, runRelDir+"/"+e.Name())
		r.runDrivers[run.Key] = append(r.runDrivers[run.Key], driver)
		for _, l := range run.Lanes {
			if l.Branch == "" || l.Stage == "done" || l.Stage == "discarded" || l.Stage == "handed-back" {
				continue
			}
			ls := laneState{key: run.Key, driver: driver}
			for _, v := range l.Validation {
				pass := len(v.Validators) > 0
				for _, val := range v.Validators {
					pass = pass && val.Pass
				}
				ls.reviews = append(ls.reviews, laneReview{head: v.HeadSHA, pass: pass})
			}
			r.lanes[l.Branch] = ls
		}
	}
}

// maxControlBytes bounds the read of a run's control record (the loop's own
// bound on it).
const maxControlBytes = 1 << 20

// runDriver is the session a run's control record names as its live driver
// (the loop's ClaimDriver writes it as each step starts), read leniently: ""
// when the record is absent, unreadable, or names no session abcd keys on,
// which says nothing of whose the run is.
func runDriver(rt *os.Root, runDir string) string {
	raw, err := fsutil.ReadGuardedInRoot(rt, runDir+"/control.json", maxControlBytes)
	if err != nil {
		return ""
	}
	var c struct {
		Driver *struct {
			Session string `json:"session"`
		} `json:"driver"`
	}
	if json.Unmarshal(raw, &c) != nil || c.Driver == nil || !ValidSessionID(c.Driver.Session) {
		return ""
	}
	return c.Driver.Session
}

// reflogBranches are the branches this working copy's HEAD reflog has
// checked out, and the one checked out now.
func (r *owedReader) reflogBranches() []string {
	var out []string
	if cur, err := r.git("symbolic-ref", "--quiet", "--short", "HEAD"); err == nil && strings.TrimSpace(cur) != "" {
		out = append(out, strings.TrimSpace(cur))
	}
	log, err := r.git("reflog", "show", "-n", "5000", "--format=%gs", "HEAD", "--")
	if err != nil {
		return out
	}
	for _, line := range strings.Split(log, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "checkout: moving from ")
		if !ok {
			continue
		}
		from, to, ok := strings.Cut(rest, " to ")
		if !ok {
			continue
		}
		out = append(out, from, to)
	}
	return out
}

// recordedBranches are the branches this copy's sessions created, as the
// guard's pre-tool hook recorded them; r.recordedBy keeps the session that
// created each, where the record kept one.
func (r *owedReader) recordedBranches() []string {
	r.recordedBy = map[string]string{}
	key := claim.OwnCopyKey(r.root)
	if key == "" || r.home == "" {
		return nil
	}
	st, err := openStoreContext(r.ctx, r.home, r.root)
	if err != nil {
		return nil
	}
	list, err := st.CreatedBranches(key)
	if err != nil {
		r.note("this copy's created-branch list could not be read (%s)", err)
		return nil
	}
	var out []string
	for _, b := range list {
		out = append(out, b.Name)
		if ValidSessionID(b.Session) {
			r.recordedBy[b.Name] = b.Session
		}
	}
	return out
}

// authoredRecently reports whether branch holds, beyond the default branch, a
// commit of this checkout's author email made or rewritten within
// StaleWindow.
func (r *owedReader) authoredRecently(b *localBranch) bool {
	if r.email == "" {
		return false
	}
	args := []string{"log", "-n", "200", "--format=%ae%x09%at%x09%ct", b.head}
	if r.defTip != "" {
		args = append(args, "--not", r.defTip)
	}
	out, err := r.git(append(args, "--")...)
	if err != nil {
		return false
	}
	cutoff := r.now.Add(-StaleWindow).Unix()
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 3 || !strings.EqualFold(f[0], r.email) {
			continue
		}
		at, _ := strconv.ParseInt(f[1], 10, 64)
		ct, _ := strconv.ParseInt(f[2], 10, 64)
		if max(at, ct) >= cutoff {
			return true
		}
	}
	return false
}

// linksByName reports whether a branch's name carries the id of a live
// reservation under this copy's key.
func (r *owedReader) linksByName(branch string) bool {
	for _, res := range r.mine {
		if mentionsID(branch, res.Record) {
			return true
		}
	}
	return false
}

// prLinks reports whether p carries the work of the reservation id: the id in
// its head branch's name or on a Resolves: or Delivers: line of its body, or
// on a Refs: line when its head is already own.
func (r *owedReader) prLinks(p pullRequest, id string, headOwn bool) bool {
	if mentionsID(p.HeadRefName, id) || trailerMentions(p.Body, closingKeys, id) {
		return true
	}
	return headOwn && trailerMentions(p.Body, refsKeys, id)
}

// findOwn names the session's own branches (dcn-2610180936568358,
// dcn-2610181003463367): the reflog's and the recorded ones while live, the
// live runs' lanes, and the heads of the person's open pull requests linked
// to a live reservation under this copy's key; never the default branch, nor
// a branch whose open pull request another person authored.
func (r *owedReader) findOwn() {
	r.own = map[string]bool{}
	var linkedHeads []string
	if r.list.read {
		for _, p := range r.list.prs {
			if p.IsCrossRepository || !r.list.authoredBy(p, r.person) {
				continue
			}
			for _, res := range r.mine {
				if r.prLinks(p, res.Record, false) {
					linkedHeads = append(linkedHeads, p.HeadRefName)
					break
				}
			}
		}
	}
	linked := map[string]bool{}
	for _, h := range linkedHeads {
		linked[h] = true
	}
	reflog, recorded := r.reflogBranches(), r.recordedBranches()
	isRecorded := map[string]bool{}
	for _, n := range recorded {
		isRecorded[n] = true
	}
	r.elsewhere = r.checkedOutElsewhere()
	elsewhere := r.elsewhere
	candidates := append(reflog, recorded...)
	for i, name := range candidates {
		b, ok := r.branches[name]
		if !ok || r.own[name] {
			continue
		}
		// A branch this copy only checked out once, which another worktree
		// has checked out now, is that worktree's: sibling copies on one
		// machine share the author email, so the email cannot tell. One
		// this copy's sessions recorded creating, or a worktree they made
		// for it, is still this copy's.
		if elsewhere[name] != "" && !isRecorded[name] && !linked[name] {
			continue
		}
		if r.expired() {
			r.ranOut(fmt.Sprintf("which branches are this copy's own (%d of %d not checked)", len(candidates)-i, len(candidates)))
			break
		}
		if linked[name] || r.linksByName(name) || r.authoredRecently(b) {
			r.own[name] = true
		}
	}
	for name := range r.lanes {
		if _, ok := r.branches[name]; ok {
			r.own[name] = true
		}
	}
	for _, h := range linkedHeads {
		r.own[h] = true
	}
	delete(r.own, r.defName)
	if r.list.read && !r.list.mine && r.person != "" {
		for name := range r.own {
			if p, ok := r.list.byHead(name, r.person); ok && !r.list.authoredBy(p, r.person) {
				delete(r.own, name)
			}
		}
	}
	r.baseNames = r.ownNames()
}

// checkedOutElsewhere are the branches a worktree other than this copy has
// checked out, each with that worktree's path.
func (r *owedReader) checkedOutElsewhere() map[string]string {
	out := map[string]string{}
	wts, err := gitutil.ListWorktreesContext(r.ctx, r.root, gitutil.MaxWorktreeListing)
	if err != nil {
		return out
	}
	here, _ := os.Stat(r.root)
	for _, wt := range wts {
		name, ok := strings.CutPrefix(wt.Branch, "refs/heads/")
		if !ok {
			continue
		}
		if fi, err := os.Stat(wt.Path); err == nil && here != nil && os.SameFile(fi, here) {
			continue
		}
		out[name] = wt.Path
	}
	return out
}

func (r *owedReader) ownNames() []string {
	names := make([]string, 0, len(r.own))
	for n := range r.own {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// fetchedAt names when the default branch was last fetched.
func (r *owedReader) fetchedAt() string {
	if r.def != "" {
		if out, err := r.git("reflog", "show", "-n", "1", "--format=%ct", r.def, "--"); err == nil {
			if s, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64); err == nil && s > 0 {
				return showTime(time.Unix(s, 0))
			}
		}
	}
	if t := claim.LastFetchedContext(r.ctx, r.root); !t.IsZero() {
		return showTime(t)
	}
	return "an unknown time"
}

// stackWalk bounds how far back diskStackBase looks in a branch's own
// history (its commits not in the last-fetched default branch).
const stackWalk = 1000

// stackIndex is what diskStackBase reads once a check, so finding every own
// branch's base takes a fixed number of git calls a branch rather than one a
// pair: the own branches by their current head, and by each tip their reflog
// set other than by creating the branch. It indexes the own branches as
// findOwn found them, before scope left out what another live session holds:
// a branch a peer holds is not owed here, but this session's branch cut from
// it is still stacked on it, and judged by the clearing rule.
type stackIndex struct {
	heads map[string][]string // a commit -> the own branches whose head it is
	tips  map[string][]string // a commit -> the own branches it was a set tip of
}

func (r *owedReader) stackIdx() *stackIndex {
	if r.stacks != nil {
		return r.stacks
	}
	idx := &stackIndex{heads: map[string][]string{}, tips: map[string][]string{}}
	for _, name := range r.baseNames {
		c, ok := r.branches[name]
		if !ok {
			continue
		}
		if r.expired() {
			// An index cut short can miss a base, which reads as none.
			r.cut = true
			continue
		}
		idx.heads[c.head] = append(idx.heads[c.head], name)
		out, err := r.git("reflog", "show", "-n", "1000", "--format=%H%x09%gs", "refs/heads/"+name, "--")
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, line := range strings.Split(out, "\n") {
			h, msg, _ := strings.Cut(line, "\t")
			if h == "" || seen[h] || strings.HasPrefix(msg, "branch: Created from") {
				continue
			}
			seen[h] = true
			idx.tips[h] = append(idx.tips[h], name)
		}
	}
	r.stacks = idx
	return idx
}

// diskStackBase is the nearest other own branch whose tip is in b's own
// history (an ancestor of b's head not in the last-fetched default branch),
// or nil. A base that has moved on since b was cut no longer has its tip
// there, so it is found too by a commit of b's own history that was the
// other branch's tip (its reflog set it there other than by creating the
// branch, which tells the base from a branch cut from it) and is still in
// its history. One walk of b's own history, newest first in topological
// order, finds the nearest; the reflogs are read once a check (stackIdx).
func (r *owedReader) diskStackBase(b *localBranch) *localBranch {
	args := []string{"rev-list", "--topo-order", "-n", strconv.Itoa(stackWalk), b.head}
	if r.defTip != "" {
		args = append(args, "--not", r.defTip)
	}
	out, err := r.git(append(args, "--")...)
	if err != nil {
		return nil
	}
	idx := r.stackIdx()
	pick := func(names []string) *localBranch {
		for _, n := range names {
			if c, ok := r.branches[n]; ok && n != b.name && c.head != b.head {
				return c
			}
		}
		return nil
	}
	history := strings.Fields(out)
	for _, h := range history {
		if c := pick(idx.heads[h]); c != nil {
			return c
		}
	}
	for _, h := range history {
		if h == b.head {
			continue
		}
		for _, n := range idx.tips[h] {
			c, ok := r.branches[n]
			if !ok || n == b.name || r.ancestor(b.head, c.head) || !r.ancestor(h, c.head) {
				continue
			}
			return c
		}
	}
	return nil
}

// baseTipOnDisk is a base branch's tip as it stands on disk: the local branch,
// else its remote-tracking ref.
func (r *owedReader) baseTipOnDisk(base string) string {
	if b, ok := r.branches[base]; ok {
		return b.head
	}
	out, err := r.git("rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+base+"^{commit}")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func (r *owedReader) lanePassed(branch, head string) (isLane, passed bool) {
	ls, ok := r.lanes[branch]
	if !ok {
		return false, false
	}
	for _, v := range ls.reviews {
		if v.head == head && v.pass {
			return true, true
		}
	}
	return true, false
}

func (r *owedReader) add(i OwedItem) { r.items = append(r.items, i) }

// branchItems judges each own branch: failing and behind pull requests,
// stacked branches by the clearing rule, and unpushed commits.
func (r *owedReader) branchItems() {
	r.stackBase = map[string]string{}
	names := r.ownNames()
	for i, name := range names {
		if r.expired() {
			r.ranOut(fmt.Sprintf("%d of %d own branches (from %s on)", len(names)-i, len(names), name))
			return
		}
		b := r.branches[name]
		head := ""
		if b != nil {
			head = b.head
		}
		var p pullRequest
		hasPR := false
		if r.away && r.list.read {
			p, hasPR = r.list.byHead(name, r.person)
			if hasPR && head == "" {
				head = p.HeadRefOid
			}
		}
		if r.away {
			r.forgeItems(name, b, head, p, hasPR)
		}
		if b != nil && r.unpushed(b) {
			r.add(OwedItem{Kind: OwedUnpushed, Capped: true, Branch: name, Head: head,
				Reason: fmt.Sprintf("branch %s has commits not pushed", name)})
		}
	}
}

// forgeItems is the away half of a branch's judgement.
func (r *owedReader) forgeItems(name string, b *localBranch, head string, p pullRequest, hasPR bool) {
	// Stacked: its pull request's base is not the default branch, or, with no
	// pull request read, its nearest own ancestor branch is not in the
	// default branch.
	base, baseTip := "", ""
	switch {
	case hasPR && p.BaseRefName != "" && p.BaseRefName != r.defName:
		base, baseTip = p.BaseRefName, r.baseTipOnDisk(p.BaseRefName)
	case !hasPR && b != nil:
		if sb := r.diskStackBase(b); sb != nil {
			base, baseTip = sb.name, sb.head
		}
	}
	if base != "" {
		r.stackBase[name] = baseTip
		r.stacked(name, head, base, baseTip, p, hasPR)
		return
	}
	if hasPR {
		if p.checks() == checksFailing {
			r.add(OwedItem{Kind: OwedFailing, Capped: true, Branch: name, PR: p.Number, Head: head,
				Reason: fmt.Sprintf("pull request #%d (%s) is failing its checks: fix them", p.Number, name)})
		}
		switch strings.ToUpper(p.MergeStateStatus) {
		case "BEHIND":
			reason := fmt.Sprintf("pull request #%d (%s) is behind its base: merge it in", p.Number, name)
			switch r.queueKeepsCurrent(p) {
			case queueArmed:
				return
			case queueUnarmed:
				reason = fmt.Sprintf("pull request #%d (%s) is behind its base, which a merge queue guards, and auto-merge is not armed: arm it", p.Number, name)
			}
			r.add(OwedItem{Kind: OwedBehind, Capped: true, Branch: name, PR: p.Number, Head: head, Reason: reason})
		case "DIRTY":
			r.add(OwedItem{Kind: OwedBehind, Capped: true, Branch: name, PR: p.Number, Head: head,
				Reason: fmt.Sprintf("pull request #%d (%s) conflicts with its base: merge it in and resolve the conflicts", p.Number, name)})
		case "UNKNOWN", "":
			r.note("the merge state of pull request #%d (%s) was not read (the forge has not computed it)", p.Number, name)
		}
		return
	}
	// The listing failed: a pushed branch is compared on disk with the
	// last-fetched default branch, naming when it was fetched.
	if !r.list.read && b != nil && b.upstream != "" && !b.upstreamGone && r.defTip != "" && !r.ancestor(r.defTip, b.head) {
		r.add(OwedItem{Kind: OwedBehind, Capped: true, Branch: name, Head: head,
			Reason: fmt.Sprintf("%s is behind %s as last fetched (fetched %s), compared on disk because the pull requests were not read: merge it in",
				name, gitutil.ShortRef(r.def), r.fetchedAt())})
	}
}

// queueStanding is what a merge queue makes of a behind pull request.
type queueStanding int

const (
	// queueNone: no queue is known to keep it current, so it is owed a merge.
	queueNone queueStanding = iota
	// queueArmed: the queue tests it against the latest base itself.
	queueArmed
	// queueUnarmed: a queue guards its base, but nobody has armed it.
	queueUnarmed
)

// queueKeepsCurrent decides a behind pull request in a repository whose
// default branch a merge queue guards (iss-2610311339274372,
// dcn-2611081501502395). A queue tests every entry against the latest base, so
// merging the base in only moves the head and voids its receipts: a pull
// request into the default branch, not a draft, that is armed or already
// queued is not owed a merge (queueArmed), and one nobody armed is owed its
// arming instead (queueUnarmed). A draft, any other base, and a queue the
// forge's live rules do not prove read as before (queueNone), each read that
// fails, is late or cannot be read named in a note; a queued listing that
// fails leaves the pull request unarmed, so it stays owed. A stacked base never
// reaches here: stacked judges it, and a conflict and failing checks are owed
// apart from it.
func (r *owedReader) queueKeepsCurrent(p pullRequest) queueStanding {
	if p.IsDraft || r.defName == "" || p.BaseRefName != r.defName || r.forge == nil {
		return queueNone
	}
	if !r.queueRead {
		r.queueRead = true
		r.queue = readQueueRule(r.ctx, r.forge, r.root, r.defName, r.rulesDeadline)
		if r.queue.unread != "" {
			r.note("whether a merge queue guards %s was not read from the forge (%s), so a pull request behind it is owed a merge as before", r.defName, r.queue.unread)
		}
	}
	if !r.queue.queue {
		return queueNone
	}
	if p.armed() {
		return queueArmed
	}
	if !r.armedRead {
		r.armedRead = true
		r.armedInto = readArmedInto(r.ctx, r.forge, r.root, r.defName, r.rulesDeadline)
		switch {
		case r.armedInto.unread != "":
			r.note("which pull requests into %s are already in its merge queue was not read from the forge (%s), so each behind one not armed is owed its arming", r.defName, r.armedInto.unread)
		case !r.armedInto.complete:
			r.note("the forge listed a page of the pull requests into %s and more follow, so a behind one not on it and not armed is owed its arming", r.defName)
		}
	}
	if r.armedInto.armed[p.Number] {
		return queueArmed
	}
	return queueUnarmed
}

// stacked applies the clearing rule (dcn-2610180936571036,
// dcn-2610181003488355): a branch stacked on a change still open is owed while
// it is behind its base, failing, with a review outstanding, or not verified;
// otherwise only its landing waits for its base.
func (r *owedReader) stacked(name, head, base, baseTip string, p pullRequest, hasPR bool) {
	var needs []string
	capped := false
	need := func(what string, forgeOnly bool) {
		needs = append(needs, what)
		capped = capped || !forgeOnly
	}
	notVerified := ""
	checks := checksNone
	if hasPR {
		checks = p.checks()
		switch strings.ToUpper(p.MergeStateStatus) {
		case "BEHIND", "DIRTY":
			need("the merge from its base", false)
		case "UNKNOWN", "":
			r.note("the merge state of pull request #%d (%s) was not read (the forge has not computed it)", p.Number, name)
		}
		if checks == checksFailing {
			need("the fixes for its failing checks", false)
		}
		if strings.EqualFold(p.ReviewDecision, "CHANGES_REQUESTED") {
			need("the review (changes are requested)", false)
		}
	} else if baseTip != "" && !r.ancestor(baseTip, head) {
		need(fmt.Sprintf("the merge from its base (%s as it stands on disk)", base), false)
	}
	if isLane, passed := r.lanePassed(name, head); isLane && !passed {
		need("the review of its head", false)
	}
	switch {
	case r.receiptsMinted:
		if !r.hasReceipt(head) {
			need("the preflight (no receipt names its head)", false)
		}
	case hasPR && checks == checksPending:
		need("its checks, still running", true)
	case hasPR && checks == checksNone:
		need("verification (no checks reported, and no receipts are minted here)", false)
	case !hasPR:
		notVerified = "it was not verified: no receipts are minted here and it has no pull request read"
	}
	if len(needs) == 0 {
		if notVerified != "" {
			r.note("%s is waiting on its base %s; %s", name, base, notVerified)
		}
		return
	}
	reason := fmt.Sprintf("%s is stacked on %s, still open, and needs now: %s; only its landing waits for its base",
		name, base, strings.Join(needs, ", "))
	if notVerified != "" {
		reason += "; " + notVerified
	}
	r.add(OwedItem{Kind: OwedStacked, Capped: capped, Branch: name, PR: p.Number, Head: head, Reason: reason})
}

// unpushed reports whether b holds commits not on its upstream, or, with no
// upstream configured, not in any remote-tracking ref and not already landed
// in the default branch (r.landed). A branch whose upstream is gone never
// counts.
func (r *owedReader) unpushed(b *localBranch) bool {
	if b.upstream != "" {
		if b.upstreamGone {
			return false
		}
		out, err := r.git("rev-list", "--count", b.upstream+".."+b.head, "--")
		n, _ := strconv.Atoi(strings.TrimSpace(out))
		return err == nil && n > 0
	}
	out, err := r.git("rev-list", "-n", "1", b.head, "--not", "--remotes", "--")
	if err != nil || strings.TrimSpace(out) == "" {
		return false
	}
	switch r.landed(b) {
	case landedYes:
		return false
	case landedUnknown:
		if r.expired() {
			r.ranOut("whether " + b.name + " has landed")
			return false
		}
		r.note("%s has commits in no remote branch and no upstream, and could not be told landed in %s or not (there is no default branch to compare with, or its change conflicts with it and its commits' patches are not all there), so it is not counted as unpushed",
			b.name, r.defLabel())
		return false
	}
	return true
}

// landedness is whether a branch's work is already in the default branch.
type landedness int

const (
	landedNo landedness = iota
	landedYes
	landedUnknown
)

// maxLandedOutput bounds the merge and patch reads of r.landed.
const maxLandedOutput = 1 << 20

// landed decides whether a branch with no upstream and commits in no remote
// branch has nonetheless landed in the default branch as last fetched: the
// usual shape of a branch pushed without -u, squash- or rebase-merged, and
// its remote branch deleted and pruned. Merging it into the default branch
// (`merge-tree --write-tree`, writing nothing, with the repository's merge
// drivers replaced by git's built-in merge, as the conflict probe pins them)
// changing nothing is landed, which covers a squash of several commits; a
// clean merge that changes the tree is not landed. A merge that conflicts,
// or fails, falls to `git cherry`: every commit already upstream by patch id
// ('-' lines only) is landed, which covers a single commit main has edited
// over since; otherwise it cannot be told. No default branch to compare with
// cannot be told either.
func (r *owedReader) landed(b *localBranch) landedness {
	if r.defTip == "" || r.expired() {
		return landedUnknown
	}
	if overrides, err := gitutil.MergeDriverOverridesContext(r.ctx, r.root); err == nil {
		args := append(overrides, "-c", "merge.union.driver="+gitutil.BuiltinMergeDriver(), "-c", "merge.renormalize=false",
			"merge-tree", "--write-tree", "--name-only", "--no-messages", "-z", r.defTip, b.head)
		out, conflicted, err := gitutil.RunCappedExitOneContext(r.ctx, r.root, maxLandedOutput, args...)
		if err == nil && !conflicted {
			tree, _, _ := strings.Cut(string(out), "\x00")
			tree = strings.TrimSpace(tree)
			defTree, terr := r.git("rev-parse", "--verify", "--quiet", r.defTip+"^{tree}")
			if gitutil.IsFullSHA(tree) && terr == nil {
				if tree == strings.TrimSpace(defTree) {
					return landedYes
				}
				return landedNo
			}
		}
	}
	out, err := gitutil.RunLimitedContext(r.ctx, r.root, maxLandedOutput, "cherry", r.defTip, b.head)
	if err != nil {
		return landedUnknown
	}
	lines := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines++
		if !strings.HasPrefix(line, "- ") {
			return landedUnknown
		}
	}
	if lines == 0 {
		return landedUnknown
	}
	return landedYes
}

// defLabel names the default branch as last fetched, for a note.
func (r *owedReader) defLabel() string {
	if r.def == "" {
		return "the default branch"
	}
	return gitutil.ShortRef(r.def) + " as last fetched"
}

// reservationItems judges this copy's live reservations: a renewal due
// halfway between the latest claim or renewal and the expiry, and, while away
// with the listing read, work no open pull request carries (a release
// subject, which no pull request carries, aside).
func (r *owedReader) reservationItems() {
	for _, res := range r.mine {
		// A reservation only another live session's runs are for is that
		// session's to renew and to carry (holders.go).
		if r.heldRes[res.Record] != "" {
			continue
		}
		half := res.ClaimedAt.Add(res.EffectiveExpiry.Sub(res.ClaimedAt) / 2)
		if r.nextRenewal.IsZero() || half.Before(r.nextRenewal) {
			r.nextRenewal = half
		}
		need := half
		if !r.now.Before(half) {
			need = res.EffectiveExpiry
		}
		if r.wakeBy.IsZero() || need.Before(r.wakeBy) {
			r.wakeBy = need
		}
		if !r.now.Before(half) {
			r.add(OwedItem{Kind: OwedRenewal, Record: res.Record,
				Reason: fmt.Sprintf("the reservation of %s is due for renewal (it lapses %s): renew it with `abcd reserve %s`",
					res.Record, showTime(res.EffectiveExpiry), shellWord(res.Record))})
		}
		if !r.away || !r.list.read || strings.HasPrefix(res.Record, "rel-") {
			continue
		}
		carried := false
		for _, p := range r.list.prs {
			// Only a pull request findOwn would count carries: in this
			// repository, and the person's.
			if p.IsCrossRepository || !r.list.authoredBy(p, r.person) {
				continue
			}
			if r.prLinks(p, res.Record, r.own[p.HeadRefName]) {
				carried = true
				break
			}
		}
		if !carried && !r.list.truncated {
			r.add(OwedItem{Kind: OwedUncarried, Capped: true, Record: res.Record,
				Reason: fmt.Sprintf("no open pull request carries the work of %s yet: finish it and open one", res.Record)})
		}
	}
}

// handBack is the one hand-back (dcn-2610181035355741): only with
// --hold-pushes, a queued step naming an own branch hands back that branch's
// unpushed item, while its head contains its base's last-fetched tip and a
// preflight receipt names it where the repository mints receipts. Each is
// moved from the owed items to r.handed for the verdict, which accepts it;
// the stop hook records what the verdict accepted, once per branch and head,
// so the reader records nothing.
func (r *owedReader) handBack() {
	if !r.away || !r.rec.HoldPushes || len(r.rec.Queue) == 0 || !r.receiptsMinted {
		return
	}
	kept := r.items[:0]
	for _, it := range r.items {
		if it.Kind != OwedUnpushed || !r.queued(it.Branch) {
			kept = append(kept, it)
			continue
		}
		baseTip := r.defTip
		if t, ok := r.stackBase[it.Branch]; ok {
			baseTip = t
		}
		if baseTip == "" || !r.ancestor(baseTip, it.Head) || !r.hasReceipt(it.Head) {
			kept = append(kept, it)
			continue
		}
		r.handed = append(r.handed, it)
	}
	r.items = kept
}

// queued reports whether a queued step names branch as a whole name.
func (r *owedReader) queued(branch string) bool {
	for _, q := range r.rec.Queue {
		if namesBranch(q.Step, branch) {
			return true
		}
	}
	return false
}

// mentionsID reports whether text names the record id on word boundaries:
// iss-5 is never found in iss-50, and is found in feat/iss-5-x.
func mentionsID(text, id string) bool {
	if id == "" {
		return false
	}
	return regexp.MustCompile(`(?:^|[^A-Za-z0-9])` + regexp.QuoteMeta(id) + `(?:$|[^A-Za-z0-9])`).MatchString(text)
}

// namesBranch reports whether text names branch as a whole name: the
// characters around it cannot continue a branch name, so feat/p is never
// found in feat/p-2 or feat/p2.
func namesBranch(text, branch string) bool {
	if branch == "" {
		return false
	}
	return regexp.MustCompile(`(?:^|[^A-Za-z0-9._/-])` + regexp.QuoteMeta(branch) + `(?:$|[^A-Za-z0-9._/-]|[.](?:$|\s))`).MatchString(text)
}

var (
	closingKeys = []string{"resolves", "delivers"}
	refsKeys    = []string{"refs"}
)

// trailerMentions reports whether body has a line keyed by one of keys
// (`Resolves: …`, any case) naming id on word boundaries.
func trailerMentions(body string, keys []string, id string) bool {
	for _, line := range strings.Split(body, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		for _, want := range keys {
			if strings.EqualFold(strings.TrimSpace(k), want) && mentionsID(v, id) {
				return true
			}
		}
	}
	return false
}
