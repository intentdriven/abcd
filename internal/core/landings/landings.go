package landings

// landings.go is the reader (spc-2610301950480609 approach 5): one Row per
// commit on the default branch's first-parent line in a Range, newest first,
// each classed by how it landed. History decides the shape (which commits a
// merge brought, which pull request a subject names, the facilitator mode the
// first parent committed, the admission a branch's records commit carries);
// the forge decides what history cannot hold (who approved which head, a
// squash's commits), and only through forge.go; the local tier adds the
// technical facilitator's recorded approvals. Once every row is classed, the
// revert trace (revert.go) marks the rows a later commit reverts, which
// changes no row's class. It writes nothing to stdout and nothing outside its
// forge cache.
//
// Every input but the object names git itself computes is text someone typed:
// a subject, a trailer, a committer's name, a forge login. Each is read to a
// strict shape or dropped, and each that a row carries is masked for a
// terminal here, where it is read, as is every error list returns. No trailer
// a branch's author typed lifts a row to person-approved: that class rests on
// a recorded approval, a forge review of the merged head, or the human mode
// the first parent committed, and the recorded approval and the mode only for
// a merge the forge's answer was read for. A subject names a pull request
// only as far as the forge confirms it merged as that row's commit; where the
// forge answered and says otherwise, the row is read as the push it is, and
// none of that pull request's evidence is read for it.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The four classes, decided in this order.
const (
	// ClassPersonPush is a first-parent commit whose subject names no pull
	// request, or names one the forge answered for and says did not merge as
	// that commit: the shape of a commit pushed to the default branch, which a
	// rebase merge's commits share. The class confirms no person, in either
	// mode: the row is unattended unless a recorded approval of the commit
	// itself is found.
	ClassPersonPush = "person-push"
	// ClassPersonApproved is a merge a person approved: a recorded approval of
	// its merged head, a forge review approving that head by someone other
	// than the pull request's author, or the human mode its first parent
	// committed, each for a merge the forge's answer was read for and says
	// merged as that commit.
	ClassPersonApproved = "person-approved"
	// ClassDrain is a merge under auto with no approval seen whose branch
	// carries the drain's admission, or such a merge in either mode whose
	// forge answer was not read.
	ClassDrain = "drain"
	// ClassAutomatic is every other merge under auto with no approval seen,
	// and every other merge in either mode with no approval seen whose forge
	// answer was not read.
	ClassAutomatic = "automatic"
)

// LaneUnknown is the lane kind of a row whose branch carries no admission, or
// two that differ, or whose commits the forge could not give.
const LaneUnknown = "unknown"

// Where a row's approval was read.
const (
	SourceForge       = "forge"
	SourceLocalTier   = "local tier"
	SourceNoneSeen    = "none seen"
	SourceForgeUnread = "forge unread"
)

// The modes a row records beside landing.ModeHuman and landing.ModeAuto.
const (
	// ModeModeless is a first parent that holds no .abcd/config.json, or a
	// row with no first parent; it reads as human, the bundled default.
	ModeModeless = "modeless"
	// ModeUnreadable is a first parent whose .abcd/config.json cannot be read
	// or judged; it never reads as human, so no row is a person's by it.
	ModeUnreadable = "unreadable"
)

// RuleMode is the kind of a Rule that names facilitator.mode.
const RuleMode = landing.KeyFacilitatorMode

// RuleForgeUnread is the kind, and the whole name, of the Rule of a merge
// under the human mode whose forge answer was not read: no rule read admits
// it, and the human mode attends only a merge the forge's answer confirms.
const RuleForgeUnread = "forge unread"

// Range is the stretch of the default branch a listing reads: From exclusive
// ("" for the whole history up to To) and To inclusive, full object names. Tip
// and Since are how the caller named them (the remote-tracking ref To was read
// from, and the tag or date From was taken from), and Note what resolve has to
// say of how it found From; list copies the three into the listing, masked,
// and reads none of them.
type Range struct {
	From  string
	To    string
	Tip   string
	Since string
	Note  string
}

// Options are list's seams.
type Options struct {
	// Forge runs the forge client; nil is landing.GHAnswerContext bound to
	// list's ctx, run in repoRoot.
	Forge landing.ForgeCall
	// Worktrees are where recorded approvals are looked for; nil lists the
	// worktrees git knows for repoRoot, once.
	Worktrees []gitutil.Worktree
}

// Rule is what admitted a row: a drain rule record by its id and path, or
// facilitator.mode by its value as .abcd/config.json held it at At (the first
// parent). A person's push names none, and a merge under the human mode whose
// forge answer was not read names RuleForgeUnread alone.
type Rule struct {
	Kind string `json:"kind,omitempty"`
	ID   string `json:"id,omitempty"`
	Path string `json:"path,omitempty"`
	At   string `json:"at,omitempty"`
}

// String is the rule as a listing names it: "drain adr-N",
// "facilitator.mode: auto" or "forge unread"; "" for none.
func (r Rule) String() string {
	switch r.Kind {
	case "", RuleForgeUnread:
		return r.Kind
	case RuleMode:
		return r.Kind + ": " + r.ID
	}
	return r.Kind + " " + r.ID
}

// Approval is the approval a row's class rests on, and where it was looked
// for. Seen is true for a recorded approval or a counted forge review; By and
// At are the forge reviewer's login and the review's time.
type Approval struct {
	Seen   bool      `json:"seen"`
	By     string    `json:"by,omitempty"`
	At     time.Time `json:"at,omitzero"`
	Source string    `json:"source,omitempty"`
}

// Row is one first-parent commit and how it landed.
type Row struct {
	Commit string    `json:"commit"`
	When   time.Time `json:"when"`
	// PR is the pull request the subject names, 0 for none. On a person-push
	// row the forge contradicted (ForgeSays set), it is the number the subject
	// named, not a merged pull request.
	PR int `json:"pr,omitempty"`
	// MergedHead is the head the merge brought: the second parent, or the
	// forge's head for a one-parent row that names a pull request; "" on a row
	// the forge contradicted.
	MergedHead string `json:"merged_head,omitempty"`
	// Committer is git's committer name, for a person's push.
	Committer string   `json:"committer,omitempty"`
	Class     string   `json:"class"`
	Rule      Rule     `json:"rule"`
	Lane      string   `json:"lane,omitempty"`
	Approval  Approval `json:"approval"`
	Mode      string   `json:"mode"`
	// Resolves, Delivers and AssistedBy are the trailers of the commits the
	// row brought, once each.
	Resolves   []string `json:"resolves,omitempty"`
	Delivers   []string `json:"delivers,omitempty"`
	AssistedBy []string `json:"assisted_by,omitempty"`
	// ForgeSays is what the forge answered against the pull request the
	// subject names, for a person-push row whose subject names one: "" for
	// every other row.
	ForgeSays string `json:"forge_says,omitempty"`
	// ForgeUnreadable says why the forge's answer for the pull request the
	// subject names was not read, whatever the cause (an answer too large to
	// read even asked alone among them): no mode attends such a row. "" for
	// every other row.
	ForgeUnreadable string `json:"forge_unreadable,omitempty"`
	// Unattended marks a row no evidence places with a person.
	Unattended bool `json:"unattended"`
	// Reverted marks a row a later commit on the default branch reverts
	// (revert.go), nil for none; it changes nothing else on the row.
	Reverted *Revert `json:"reverted,omitempty"`
	// Reverts are the revert lines the row's commits carry, and its
	// untraceable reverts, each with the row it marks or why it marks none.
	Reverts []RevertLine `json:"reverts,omitempty"`
}

// Listing is list's answer.
type Listing struct {
	Tip   string `json:"tip"`
	From  string `json:"from"`
	To    string `json:"to"`
	Since string `json:"since"`
	// SinceNote says what a reader of the range should know of how From was
	// found (Range.Note), "" for nothing.
	SinceNote string `json:"since_note,omitempty"`
	Rows      []Row  `json:"rows"`
	// ForgeUnread says why the forge did not answer for some row, "" when it
	// answered for every row that names a pull request; it opens with the
	// count of rows whose answer was not read (unreadNote).
	ForgeUnread string `json:"forge_unread,omitempty"`
	// CacheUnwritten says why an answer could not be kept in the forge cache.
	CacheUnwritten string `json:"cache_unwritten,omitempty"`
	// ModeUnread says why the facilitator mode some first parent committed
	// could not be read, "" when every one read; such a row's mode is
	// ModeUnreadable.
	ModeUnread string `json:"mode_unread,omitempty"`
}

// Bounds on what list reads from git.
const (
	// maxListBytes bounds the range's commit listing and a tree listing.
	maxListBytes = 16 << 20
	// maxTextBytes bounds a name a row carries.
	maxTextBytes = 200
)

// maxHistoryBytes bounds the log of the range, messages and all: a range past
// it is refused whole, never read in part. A variable only so a test can hold
// a small history to it.
var maxHistoryBytes = 64 << 20

// listBound is the time list allows itself when its caller's ctx sets no
// deadline: every git and forge call it starts ends by then.
const listBound = 10 * time.Minute

// bounded is ctx with listBound as its deadline when it carries none.
func bounded(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, listBound)
}

// The reader's seams onto git and the landing package, variables so a test
// can hand it a log git would never write or count its reads of the mode.
var (
	historyLog    = gitutil.RunCappedBytesContext
	committedMode = landing.CommittedMode
)

// maskedError is a failure list returns, its text masked as a row's text is
// (whyText) and its cause kept for errors.Is.
type maskedError struct {
	msg   string
	cause error
}

func (e *maskedError) Error() string { return e.msg }
func (e *maskedError) Unwrap() error { return e.cause }

// masked is err as list returns it: what failed, then err's text masked.
func masked(what string, err error) error {
	return &maskedError{msg: what + ": " + whyText(err), cause: err}
}

// list is the reader's entry (the spec's List). It stays unexported for this
// step: step 4 exports it when it wires the verb.
//
// list reads every first-parent commit of r on the repository at repoRoot,
// newest first, and classes each. It fetches nothing; it writes only its forge
// cache in repoRoot's local tier. Every git call and every forge call is bound
// to ctx, or to listBound when ctx sets no deadline, and each forge batch also
// to landing.ForgeTimeout. A ctx that ends before the listing is done is the
// error, whatever the read it ended was.
func list(ctx context.Context, repoRoot string, r Range, o Options) (Listing, error) {
	ctx, cancel := bounded(ctx)
	defer cancel()
	l, err := listRows(ctx, repoRoot, r, o)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return Listing{}, ctxErr
	}
	return l, err
}

// listRows is list's work, under the ctx list bounds.
func listRows(ctx context.Context, repoRoot string, r Range, o Options) (Listing, error) {
	if !gitutil.IsFullSHA(r.To) || (r.From != "" && !gitutil.IsFullSHA(r.From)) {
		return Listing{}, errors.New("a landings range names its ends by full object names")
	}
	h, err := readHistory(ctx, repoRoot, r)
	if err != nil {
		return Listing{}, err
	}
	l := &lister{ctx: ctx, root: repoRoot, h: h, rules: map[string][]string{}}
	rows := h.firstParent(r.To)
	h.assign(rows)
	modes, err := l.modes(r, rows)
	if err != nil {
		return Listing{}, err
	}
	out := Listing{Tip: boundText(termsafe.Sanitize(r.Tip), maxTextBytes), From: r.From, To: r.To,
		Since: boundText(termsafe.Sanitize(r.Since), maxTextBytes), SinceNote: termsafe.CleanProseLine(r.Note, 600), Rows: []Row{},
		ModeUnread: boundText(strings.Join(l.modeWhy, "; "), 4000)}

	facts := make([]rowFacts, len(rows))
	var asks []prAsk
	for i, sha := range rows {
		facts[i] = l.historyFacts(sha, modes[sha])
		if facts[i].row.PR != 0 {
			asks = append(asks, prAsk{pr: facts[i].row.PR, commit: sha, commits: !facts[i].twoParent})
		}
	}

	forge := o.Forge
	if forge == nil {
		forge = func(args ...string) (string, error) { return landing.GHAnswerContext(ctx, repoRoot, args...) }
	}
	answers := readForge(ctx, repoRoot, forge, asks)
	out.CacheUnwritten = answers.cacheWhy

	wts := o.Worktrees
	if wts == nil {
		if wts, err = gitutil.ListWorktreesContext(ctx, repoRoot, gitutil.MaxWorktreeListing); err != nil {
			return Listing{}, masked("listing the repository's worktrees", err)
		}
		if wts == nil {
			// Listed once: an empty listing must not read as "list them".
			wts = []gitutil.Worktree{}
		}
	}
	var unread []string
	unreadRows := 0
	for i := range facts {
		pr := facts[i].row.PR
		row, why, err := l.classify(facts[i], answers.got[pr], answers.why[pr], answers.notPR[pr], wts)
		if err != nil {
			return Listing{}, err
		}
		if why != "" && !slices.Contains(unread, why) {
			unread = append(unread, why)
		}
		// Every row the forge's answer was not read for is unattended
		// (classify: no approval is read for it, and no mode attends it), so
		// the count needs no second test.
		if row.ForgeUnreadable != "" {
			unreadRows++
		}
		out.Rows = append(out.Rows, row)
	}
	out.ForgeUnread = boundText(unreadNote(unreadRows, unread), 4000)

	// The revert trace asks a pull request's body only where its answer was
	// read and confirmed the row (classify): a row the forge contradicted, or
	// whose answer was not read, carries no forge revert line. The body is
	// asked in a document of its own, after every row is classed, within
	// what the listing's bound on asking again still holds, so no row's class
	// rests on it.
	accepted := map[string]bool{}
	for _, row := range out.Rows {
		if row.PR != 0 && answers.got[row.PR] != nil && row.ForgeSays == "" && row.ForgeUnreadable == "" {
			accepted[row.Commit] = true
		}
	}
	l.bodies = func(asks []prAsk) bodyAnswers { return readBodies(ctx, forge, asks, answers.left) }
	l.traceReverts(out.Rows, accepted)
	return out, nil
}

// unreadNote is the listing's forge note: how many rows the forge's answer
// was not read for that are left unattended, first, so the bound never cuts
// it, then each reason once.
func unreadNote(rows int, reasons []string) string {
	note := strings.Join(reasons, "; ")
	if rows == 0 {
		return note
	}
	noun := "rows"
	if rows == 1 {
		noun = "row"
	}
	return fmt.Sprintf("the forge's answer was not read for %d %s, which no mode attends, and a later listing asks about them again: %s", rows, noun, note)
}

// commitInfo is one commit as the range's log gives it.
type commitInfo struct {
	parents   []string
	when      int64
	committer string
	message   string
}

// history is every commit the range reaches, and which first-parent row
// brought each.
type history struct {
	commits map[string]*commitInfo
	onLine  map[string]bool
	// branch is each row's commits, the second parents' walk, in walk order.
	branch map[string][]string
	// rowOf is the row that brought each commit assigned: a first-parent
	// commit's own, and the row whose walk took each other.
	rowOf map[string]string
}

// readHistory reads every commit From..To reaches in one log, NUL-separated,
// so a message carrying the unit or record separator is still one commit's
// message (git ends a message at a NUL). It refuses an answer whose commits do
// not match git's own listing of the range one for one, in number, name and
// order: whatever the log holds, nothing in it is read as a commit git did not
// list.
func readHistory(ctx context.Context, repoRoot string, r Range) (*history, error) {
	spec := r.To
	if r.From != "" {
		spec = r.From + ".." + r.To
	}
	raw, err := historyLog(ctx, repoRoot, maxHistoryBytes,
		"log", "-z", "--no-show-signature", "--format=%H%x1f%P%x1f%ct%x1f%cn%x1f%B", spec, "--")
	if err != nil {
		return nil, masked("reading the default branch's history", err)
	}
	listed, err := gitutil.RunCappedContext(ctx, repoRoot, maxListBytes, "rev-list", spec, "--")
	if err != nil {
		return nil, masked("listing the default branch's commits", err)
	}
	want := strings.Fields(listed)
	h := &history{commits: map[string]*commitInfo{}, onLine: map[string]bool{}, branch: map[string][]string{}, rowOf: map[string]string{}}
	recs := bytes.Split(raw, []byte{0})
	if n := len(recs); n > 0 && len(bytes.TrimSpace(recs[n-1])) == 0 {
		recs = recs[:n-1]
	}
	refuse := errors.New("the default branch's history does not read apart into the commits git lists for the range, so it is not listed")
	if len(recs) != len(want) {
		return nil, refuse
	}
	for i, rec := range recs {
		f := strings.SplitN(strings.TrimPrefix(string(rec), "\n"), "\x1f", 5)
		if len(f) != 5 || f[0] != want[i] || !gitutil.IsFullSHA(f[0]) || h.commits[f[0]] != nil {
			return nil, refuse
		}
		ps := strings.Fields(f[1])
		for _, p := range ps {
			if !gitutil.IsFullSHA(p) {
				return nil, refuse
			}
		}
		when, err := strconv.ParseInt(f[2], 10, 64)
		if err != nil {
			return nil, refuse
		}
		h.commits[f[0]] = &commitInfo{parents: ps, when: when, committer: f[3], message: f[4]}
	}
	return h, nil
}

// firstParent is the first-parent line from to, newest first, within the
// range.
func (h *history) firstParent(to string) []string {
	var rows []string
	for c := to; h.commits[c] != nil && !h.onLine[c]; {
		rows = append(rows, c)
		h.onLine[c] = true
		h.rowOf[c] = c
		ps := h.commits[c].parents
		if len(ps) == 0 {
			break
		}
		c = ps[0]
	}
	return rows
}

// assign gives each commit the range reaches off the first-parent line to the
// row whose second-parent walk first reaches it, oldest row first, stopping at
// the line and at commits already assigned: the row that brought it onto the
// default branch.
func (h *history) assign(rows []string) {
	taken := map[string]bool{}
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		stack := slices.Clone(h.commits[row].parents[min(1, len(h.commits[row].parents)):])
		for len(stack) > 0 {
			c := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			ci := h.commits[c]
			if ci == nil || h.onLine[c] || taken[c] {
				continue
			}
			taken[c] = true
			h.branch[row] = append(h.branch[row], c)
			h.rowOf[c] = row
			for j := len(ci.parents) - 1; j >= 0; j-- {
				stack = append(stack, ci.parents[j])
			}
		}
	}
}

// lister holds one list's reads.
type lister struct {
	ctx  context.Context
	root string
	h    *history
	// rules memoises the drain rule records' file names at a commit.
	rules map[string][]string
	// modeWhy is why each mode that could not be read was not.
	modeWhy []string
	// bodies asks the forge the bodies the revert trace reads; nil asks
	// nothing, and every body reads as not asked.
	bodies func([]prAsk) bodyAnswers
}

// rowFacts are what history says of one row before the forge is asked.
type rowFacts struct {
	row       Row
	firstPar  string
	twoParent bool
	// messages are the commits the row brought, for its lane and trailers.
	messages []string
}

// The subjects that name a pull request: the forge's merge commit, and a
// squash's trailing number. A number past nine digits is no pull request the
// forge's 32-bit Int can name.
var (
	mergeSubjectRe  = regexp.MustCompile(`^Merge pull request #([1-9][0-9]{0,8}) from \S`)
	squashSubjectRe = regexp.MustCompile(`\(#([1-9][0-9]{0,8})\)$`)
)

// subjectPR is the pull request a commit's subject (its message's first
// line) names, 0 for none.
func subjectPR(message string) int {
	subject, _, _ := strings.Cut(strings.TrimLeft(message, "\n"), "\n")
	subject = strings.TrimSpace(subject)
	m := mergeSubjectRe.FindStringSubmatch(subject)
	if m == nil {
		m = squashSubjectRe.FindStringSubmatch(subject)
	}
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// historyFacts reads one row's facts from history: its pull request, merged
// head, the commits it brought, and mode, its first parent's (modes).
func (l *lister) historyFacts(sha, mode string) rowFacts {
	c := l.h.commits[sha]
	f := rowFacts{row: Row{Commit: sha, When: time.Unix(c.when, 0).UTC(), PR: subjectPR(c.message), Mode: mode}}
	if len(c.parents) > 0 {
		f.firstPar = c.parents[0]
	}
	f.twoParent = len(c.parents) > 1
	if f.twoParent {
		f.row.MergedHead = c.parents[1]
	}
	switch {
	case f.row.PR == 0:
		f.messages = []string{c.message}
	case f.twoParent:
		for _, b := range l.h.branch[sha] {
			f.messages = append(f.messages, l.h.commits[b].message)
		}
	}
	return f
}

// modes is each row's mode as its first parent committed it. The commits on
// the first-parent line that change .abcd/config.json against their first
// parent are read in one log, and the file is read and judged only at the
// oldest row's first parent and after each of those, so a range in which the
// mode never changes reads it once.
func (l *lister) modes(r Range, rows []string) (map[string]string, error) {
	spec := r.To
	if r.From != "" {
		spec = r.From + ".." + r.To
	}
	out, err := gitutil.RunCappedContext(l.ctx, l.root, maxListBytes, "log", "--first-parent", "--no-show-signature", "--format=%H", spec, "--", layered.Config.RepoRel)
	if err != nil {
		return nil, masked("reading where the facilitator mode changed", err)
	}
	changed := map[string]bool{}
	for _, c := range strings.Fields(out) {
		changed[c] = true
	}
	modes := make(map[string]string, len(rows))
	mode := ""
	for i := len(rows) - 1; i >= 0; i-- {
		if i == len(rows)-1 || changed[rows[i+1]] {
			ps := l.h.commits[rows[i]].parents
			if len(ps) == 0 {
				mode = ModeModeless
			} else {
				mode = l.modeAt(ps[0])
			}
		}
		modes[rows[i]] = mode
	}
	return modes, nil
}

// modeAt is the mode as commit committed it: modeless for a commit with no
// .abcd/config.json, and unreadable for one whose file cannot be read or
// judged, the reason kept for the listing's note.
func (l *lister) modeAt(commit string) string {
	mode, found, err := committedMode(l.ctx, l.root, commit)
	switch {
	case err != nil:
		// Each first parent is read once (modes), so each reason is said once.
		l.modeWhy = append(l.modeWhy, "the facilitator mode "+commit[:12]+" committed cannot be read: "+whyText(err))
		return ModeUnreadable
	case !found:
		return ModeModeless
	}
	return mode
}

// human says whether a row's mode reads as human: committed so, or not
// committed at all (the bundled default).
func human(mode string) bool { return mode == landing.ModeHuman || mode == ModeModeless }

// modeRule is facilitator.mode as the row's first parent held it.
func (f rowFacts) modeRule() Rule {
	id := f.row.Mode
	if id == ModeModeless {
		id = landing.ModeHuman
	}
	r := Rule{Kind: RuleMode, ID: id, At: f.firstPar}
	if f.row.Mode != ModeModeless {
		r.Path = ".abcd/config.json"
	}
	return r
}

// classify decides one row's class, rule, lane, approval and trailers from its
// history facts, the forge's answer for its pull request (nil when the forge
// gave none, with why, and notPR when the forge says the number names no pull
// request), and the recorded approvals in wts. The why it returns is a reason
// the forge was not read for this row, for the listing's note.
func (l *lister) classify(f rowFacts, a *prAnswer, why string, notPR bool, wts []gitutil.Worktree) (Row, string, error) {
	row := f.row
	if row.PR == 0 {
		return l.push(row, "", wts), "", nil
	}
	// The forge contradicts the subject when it answered and says the number
	// names no pull request, or that the pull request did not merge as this
	// commit: the row is then the push it is, in either mode, and nothing of
	// that pull request is read for it. A forge whose answer was not read,
	// whatever the cause, leaves the row to the subject, and no mode attends
	// it: the row carries why.
	if notPR {
		return l.push(row, why, wts), "", nil
	}
	if a != nil && why == "" {
		if said := a.landedAs(row.PR, row.Commit); said != "" {
			return l.push(row, said, wts), "", nil
		}
	}
	if why != "" {
		a = nil
	}
	if a == nil {
		if why == "" {
			why = fmt.Sprintf("the forge gave no answer for pull request #%d", row.PR)
		}
		row.ForgeUnreadable = termsafe.CleanProseLine(why, 300)
	}
	messages := f.messages
	commitsRead := f.twoParent
	if !f.twoParent && a != nil {
		row.MergedHead = a.HeadRefOid
		if !a.commitsUnread && a.Commits.TotalCount <= len(a.Commits.Nodes) {
			commitsRead = true
			for _, c := range a.Commits.Nodes {
				messages = append(messages, c.Commit.Message)
			}
		}
	}
	row.Resolves, row.Delivers, row.AssistedBy = trailers(messages)
	var adm *Admitted
	row.Lane = LaneUnknown
	if commitsRead {
		if one, ok := admission(messages); ok {
			adm = &one
			row.Lane = one.Kind
		}
	}

	// A recorded approval of the merged head counts only beside a forge answer
	// read for the pull request, and confirmed above as merged as this commit.
	row.Approval.Source = SourceNoneSeen
	if row.MergedHead != "" && a != nil {
		row.Approval = l.recorded(row.MergedHead, wts)
	}
	if !row.Approval.Seen {
		switch {
		case a == nil:
			row.Approval.Source = SourceForgeUnread
		default:
			appr, pageWhy := a.approval(row.MergedHead)
			if pageWhy != "" {
				row.Approval.Source, why = SourceForgeUnread, pageWhy
			} else if appr.Seen {
				row.Approval = appr
			}
		}
	}

	switch {
	case row.Approval.Seen:
		row.Class, row.Rule = ClassPersonApproved, f.modeRule()
	case human(row.Mode) && a != nil:
		// The human mode attends a merge the forge's answer was read for and
		// says merged as this commit (landedAs above), and no other.
		row.Class, row.Rule = ClassPersonApproved, f.modeRule()
	case adm != nil && adm.Kind == KindDrain:
		row.Class = ClassDrain
		row.Rule = Rule{Kind: KindDrain, ID: adm.Record}
		p, err := l.rulePath(f.firstPar, adm.Record)
		if err != nil {
			return Row{}, "", err
		}
		row.Rule.Path = p
	case human(row.Mode):
		// An unattended merge never carries the human mode's rule, so a
		// grouping by rule never places it with the merges the mode attends.
		row.Class, row.Rule = ClassAutomatic, Rule{Kind: RuleForgeUnread}
	default:
		row.Class, row.Rule = ClassAutomatic, f.modeRule()
	}
	row.Unattended = row.Class == ClassDrain || row.Class == ClassAutomatic
	return row, why, nil
}

// push is row read as a person's push: its committer, the trailers of its own
// message, and unattended unless the technical facilitator's recorded
// approval of the commit itself is found, whatever the mode. said is what the
// forge answered against the pull request its subject names, "" for a subject
// that names none; where the forge said so, the merged head is cleared, the
// forge having denied the merge it would name.
func (l *lister) push(row Row, said string, wts []gitutil.Worktree) Row {
	c := l.h.commits[row.Commit]
	row.Class = ClassPersonPush
	if said != "" {
		row.MergedHead = ""
	}
	row.Committer = boundText(termsafe.Sanitize(c.committer), maxTextBytes)
	row.Resolves, row.Delivers, row.AssistedBy = trailers([]string{c.message})
	row.ForgeSays = termsafe.CleanProseLine(said, 300)
	row.Approval = l.recorded(row.Commit, wts)
	row.Unattended = !row.Approval.Seen
	return row
}

// recorded is the technical facilitator's recorded approval of sha in wts as
// a row's approval: Seen from the local tier, or none seen. A record that
// cannot be counted (one git tracks, say) counts for nothing, as the merge
// gate counts it.
func (l *lister) recorded(sha string, wts []gitutil.Worktree) Approval {
	if p, err := landing.ApprovalRecord(l.ctx, l.root, sha, wts); err == nil && p != "" {
		return Approval{Seen: true, Source: SourceLocalTier}
	}
	return Approval{Source: SourceNoneSeen}
}

// admission is the one admission the commits' messages carry: none, or two
// that differ, is no admission.
func admission(messages []string) (Admitted, bool) {
	var found *Admitted
	for _, m := range messages {
		a, ok := ParseAdmitted(m)
		if !ok {
			continue
		}
		if found != nil && *found != a {
			return Admitted{}, false
		}
		found = &a
	}
	if found == nil {
		return Admitted{}, false
	}
	return *found, true
}

// rulePath is the path of the drain rule record id in the decision store as
// commit holds it, "" when it holds none by that id.
func (l *lister) rulePath(commit, id string) (string, error) {
	if commit == "" {
		return "", nil
	}
	names, ok := l.rules[commit]
	if !ok {
		out, err := gitutil.RunCappedContext(l.ctx, l.root, maxListBytes, "ls-tree", "-z", "--name-only", commit, "--", drainrule.ADRsRelDir+"/")
		if err == nil {
			for n := range strings.SplitSeq(out, "\x00") {
				if n != "" {
					names = append(names, n)
				}
			}
		}
		l.rules[commit] = names
	}
	for _, n := range names {
		if path.Dir(n) == drainrule.ADRsRelDir && recordid.ADRFileID(path.Base(n)) == id {
			return boundText(termsafe.Sanitize(n), maxTextBytes), nil
		}
	}
	return "", nil
}

// The trailers a row carries, each to the shape the repository writes.
var (
	trailerRe    = regexp.MustCompile(`(?i)^(resolves|delivers|assisted-by)\s*:\s*(\S+)$`)
	issueRe      = regexp.MustCompile(`^iss-[0-9]{1,20}$`)
	assistedByRe = regexp.MustCompile(`^(None|[A-Za-z][A-Za-z0-9._-]{0,63}:[A-Za-z0-9][A-Za-z0-9._+-]{0,63})$`)
)

// trailers reads the Resolves:, Delivers: and Assisted-by: trailers of every
// message's trailer block (trailerBlock, the reader ParseAdmitted reads the
// admission through), once each, sorted. A value not in the shape its key
// takes is dropped.
func trailers(messages []string) (resolves, delivers, assisted []string) {
	add := func(list []string, v string) []string {
		if slices.Contains(list, v) {
			return list
		}
		return append(list, v)
	}
	for _, m := range messages {
		for _, line := range trailerBlock(m) {
			t := trailerRe.FindStringSubmatch(line)
			if t == nil {
				continue
			}
			switch v := t[2]; strings.ToLower(t[1]) {
			case "resolves":
				if issueRe.MatchString(v) {
					resolves = add(resolves, v)
				}
			case "delivers":
				if recordid.ValidIntentID(v) && len(v) <= 24 {
					delivers = add(delivers, v)
				}
			case "assisted-by":
				if assistedByRe.MatchString(v) {
					assisted = add(assisted, v)
				}
			}
		}
	}
	slices.Sort(resolves)
	slices.Sort(delivers)
	slices.Sort(assisted)
	return resolves, delivers, assisted
}

// boundText cuts s to at most n bytes, never inside a rune.
func boundText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// whyText masks a failure's text for a listing's note: home-redacted,
// credential-free, one line and bounded.
func whyText(err error) string {
	return termsafe.CleanProseLine(fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error())), 300)
}
