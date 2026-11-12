package landings

// landings.go is the reader (spc-2610301950480609 approach 5): one Row per
// commit on the default branch's first-parent line in a Range, newest first,
// each classed by how it landed. History decides the shape (which commits a
// merge brought, which pull request a subject names, the facilitator mode the
// first parent committed, the admission a branch's records commit carries);
// the forge decides what history cannot hold (who approved which head, a
// squash's commits), and only through forge.go; the local tier adds the
// technical facilitator's recorded approvals. It writes nothing to stdout and
// nothing outside its forge cache.
//
// Every input but the object names git itself computes is text someone typed:
// a subject, a trailer, a committer's name, a forge login or body. Each is read
// to a strict shape or dropped, and each that a row carries is masked for a
// terminal here, where it is read. No trailer a branch's author typed lifts a
// row to person-approved: that class rests on a recorded approval, a forge
// review of the merged head, or the human mode the first parent committed.

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
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The four classes, decided in this order.
const (
	// ClassPersonPush is a first-parent commit whose subject names no pull
	// request: someone pushed it to the default branch.
	ClassPersonPush = "person-push"
	// ClassPersonApproved is a merge a person approved: a recorded approval of
	// its merged head, a forge review approving that head by someone other
	// than the pull request's author, or the human mode its first parent
	// committed.
	ClassPersonApproved = "person-approved"
	// ClassDrain is a merge under auto with no approval seen whose branch
	// carries the drain's admission.
	ClassDrain = "drain"
	// ClassAutomatic is every other merge under auto with no approval seen.
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

// Range is the stretch of the default branch a listing reads: From exclusive
// ("" for the whole history up to To) and To inclusive, full object names. Tip
// and Since are how the caller named them (the remote-tracking ref To was read
// from, and the tag or date From was taken from); list copies them into the
// listing and reads neither.
type Range struct {
	From  string
	To    string
	Tip   string
	Since string
}

// Options are list's seams.
type Options struct {
	// Forge runs the forge client; nil is landing.GHContext bound to list's
	// ctx, run in repoRoot.
	Forge landing.ForgeCall
	// Worktrees are where recorded approvals are looked for; nil lists the
	// worktrees git knows for repoRoot, once.
	Worktrees []gitutil.Worktree
}

// Rule is what admitted a row: a drain rule record by its id and path, or
// facilitator.mode by its value as .abcd/config.json held it at At (the first
// parent). A person's push names none.
type Rule struct {
	Kind string `json:"kind,omitempty"`
	ID   string `json:"id,omitempty"`
	Path string `json:"path,omitempty"`
	At   string `json:"at,omitempty"`
}

// String is the rule as a listing names it: "drain adr-N" or
// "facilitator.mode: auto"; "" for none.
func (r Rule) String() string {
	switch r.Kind {
	case "":
		return ""
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
	// PR is the pull request the subject names, 0 for none.
	PR int `json:"pr,omitempty"`
	// MergedHead is the head the merge brought: the second parent, or the
	// forge's head for a one-parent row that names a pull request.
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
	// Unattended marks a row no evidence places with a person.
	Unattended bool `json:"unattended"`
	// The revert trace (approach 7) marks a row it undoes here.
}

// Listing is list's answer.
type Listing struct {
	Tip   string `json:"tip"`
	From  string `json:"from"`
	To    string `json:"to"`
	Since string `json:"since"`
	Rows  []Row  `json:"rows"`
	// ForgeUnread says why the forge did not answer for some row, "" when it
	// answered for every row that names a pull request.
	ForgeUnread string `json:"forge_unread,omitempty"`
	// CacheUnwritten says why an answer could not be kept in the forge cache.
	CacheUnwritten string `json:"cache_unwritten,omitempty"`
}

// Bounds on what list reads from git.
const (
	// maxHistoryBytes bounds the log of the range, messages and all: a range
	// past it is refused whole, never read in part.
	maxHistoryBytes = 64 << 20
	// maxListBytes bounds the range's commit listing and a tree listing.
	maxListBytes = 16 << 20
	// maxTextBytes bounds a name a row carries.
	maxTextBytes = 200
)

// list is the reader's entry (the spec's List): it stays unexported until a
// front door calls it, the verb and the spot-check, since the core reach
// audit refuses an exported function no production code reaches.
//
// list reads every first-parent commit of r on the repository at repoRoot,
// newest first, and classes each. It fetches nothing; it writes only its forge
// cache in repoRoot's local tier. Every git call and every forge call is bound
// to ctx, and each forge batch also to landing.ForgeTimeout.
func list(ctx context.Context, repoRoot string, r Range, o Options) (Listing, error) {
	if !gitutil.IsFullSHA(r.To) || (r.From != "" && !gitutil.IsFullSHA(r.From)) {
		return Listing{}, errors.New("a landings range names its ends by full object names")
	}
	h, err := readHistory(ctx, repoRoot, r)
	if err != nil {
		return Listing{}, err
	}
	l := &lister{ctx: ctx, root: repoRoot, h: h, modes: map[string]string{}, rules: map[string][]string{}}
	rows := h.firstParent(r.To)
	h.assign(rows)
	out := Listing{Tip: termsafe.Sanitize(r.Tip), From: r.From, To: r.To, Since: termsafe.Sanitize(r.Since), Rows: []Row{}}

	facts := make([]rowFacts, len(rows))
	var asks []prAsk
	for i, sha := range rows {
		f, err := l.historyFacts(sha)
		if err != nil {
			return Listing{}, err
		}
		facts[i] = f
		if f.row.PR != 0 {
			asks = append(asks, prAsk{pr: f.row.PR, commit: sha})
		}
	}

	forge := o.Forge
	if forge == nil {
		forge = func(args ...string) (string, error) { return landing.GHContext(ctx, repoRoot, args...) }
	}
	answers := readForge(ctx, repoRoot, forge, asks)
	if err := ctx.Err(); err != nil {
		return Listing{}, err
	}
	out.CacheUnwritten = answers.cacheWhy

	wts := o.Worktrees
	if wts == nil {
		if wts, err = gitutil.ListWorktreesContext(ctx, repoRoot, gitutil.MaxWorktreeListing); err != nil {
			return Listing{}, fmt.Errorf("listing the repository's worktrees: %w", err)
		}
	}
	var unread []string
	for i := range facts {
		row, why, err := l.classify(facts[i], answers.got[facts[i].row.PR], answers.why[facts[i].row.PR], wts)
		if err != nil {
			return Listing{}, err
		}
		if why != "" && !slices.Contains(unread, why) {
			unread = append(unread, why)
		}
		out.Rows = append(out.Rows, row)
	}
	out.ForgeUnread = boundText(strings.Join(unread, "; "), 4000)
	return out, nil
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
}

// readHistory reads every commit From..To reaches in one log, and refuses an
// answer whose commits do not match git's own listing of the range one for
// one: a message is text its author chose, and one that carries the log's
// separators must never be read as another commit.
func readHistory(ctx context.Context, repoRoot string, r Range) (*history, error) {
	spec := r.To
	if r.From != "" {
		spec = r.From + ".." + r.To
	}
	raw, err := gitutil.RunCappedBytesContext(ctx, repoRoot, maxHistoryBytes,
		"log", "-z", "--no-show-signature", "--format=%H%x1f%P%x1f%ct%x1f%cn%x1f%B", spec, "--")
	if err != nil {
		return nil, fmt.Errorf("reading the default branch's history: %w", err)
	}
	listed, err := gitutil.RunCappedContext(ctx, repoRoot, maxListBytes, "rev-list", spec, "--")
	if err != nil {
		return nil, fmt.Errorf("listing the default branch's commits: %w", err)
	}
	want := strings.Fields(listed)
	h := &history{commits: map[string]*commitInfo{}, onLine: map[string]bool{}, branch: map[string][]string{}}
	recs := bytes.Split(raw, []byte{0})
	if n := len(recs); n > 0 && len(bytes.TrimSpace(recs[n-1])) == 0 {
		recs = recs[:n-1]
	}
	refuse := errors.New("the default branch's history does not read apart into its commits (a commit message carries the log's separators), so it is not listed")
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
			for j := len(ci.parents) - 1; j >= 0; j-- {
				stack = append(stack, ci.parents[j])
			}
		}
	}
}

// lister holds one list's reads.
type lister struct {
	ctx   context.Context
	root  string
	h     *history
	modes map[string]string
	// rules memoises the drain rule records' file names at a commit.
	rules map[string][]string
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
// head, mode at the first parent, and the commits it brought.
func (l *lister) historyFacts(sha string) (rowFacts, error) {
	c := l.h.commits[sha]
	f := rowFacts{row: Row{Commit: sha, When: time.Unix(c.when, 0).UTC(), PR: subjectPR(c.message)}}
	if len(c.parents) > 0 {
		f.firstPar = c.parents[0]
	}
	f.twoParent = len(c.parents) > 1
	if f.twoParent {
		f.row.MergedHead = c.parents[1]
	}
	mode, err := l.modeAt(f.firstPar)
	if err != nil {
		return rowFacts{}, err
	}
	f.row.Mode = mode
	switch {
	case f.row.PR == 0:
		f.messages = []string{c.message}
	case f.twoParent:
		for _, b := range l.h.branch[sha] {
			f.messages = append(f.messages, l.h.commits[b].message)
		}
	}
	return f, nil
}

// modeAt is the row's mode as its first parent committed it, memoised.
func (l *lister) modeAt(commit string) (string, error) {
	if commit == "" {
		return ModeModeless, nil
	}
	if m, ok := l.modes[commit]; ok {
		return m, nil
	}
	mode, found, err := landing.CommittedMode(l.ctx, l.root, commit)
	if ctxErr := l.ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}
	switch {
	case err != nil:
		mode = ModeUnreadable
	case !found:
		mode = ModeModeless
	}
	l.modes[commit] = mode
	return mode, nil
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
// gave none, with why), and the recorded approvals in wts. The why it returns
// is a reason the forge was not read for this row, for the listing's note.
func (l *lister) classify(f rowFacts, a *prAnswer, why string, wts []gitutil.Worktree) (Row, string, error) {
	row := f.row
	if row.PR == 0 {
		row.Class = ClassPersonPush
		row.Committer = boundText(termsafe.Sanitize(l.h.commits[row.Commit].committer), maxTextBytes)
		row.Resolves, row.Delivers, row.AssistedBy = trailers(f.messages)
		return row, "", nil
	}
	if a != nil && why == "" {
		why = a.landedAs(row.PR, row.Commit)
	}
	if why != "" {
		a = nil
	}
	messages := f.messages
	commitsRead := f.twoParent
	if !f.twoParent && a != nil {
		row.MergedHead = a.HeadRefOid
		if a.Commits.TotalCount <= len(a.Commits.Nodes) {
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

	row.Approval.Source = SourceNoneSeen
	if row.MergedHead != "" {
		p, err := landing.ApprovalRecord(l.ctx, l.root, row.MergedHead, wts)
		if ctxErr := l.ctx.Err(); ctxErr != nil {
			return Row{}, "", ctxErr
		}
		// A record that cannot be counted (one git tracks, say) counts for
		// nothing, as the merge gate counts it: the row falls to the forge and
		// the mode, which never read it as a person's on less.
		if err == nil && p != "" {
			row.Approval = Approval{Seen: true, Source: SourceLocalTier}
		}
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
	case human(row.Mode):
		row.Class, row.Rule = ClassPersonApproved, f.modeRule()
	case adm != nil && adm.Kind == KindDrain:
		row.Class = ClassDrain
		row.Rule = Rule{Kind: KindDrain, ID: adm.Record}
		p, err := l.rulePath(f.firstPar, adm.Record)
		if err != nil {
			return Row{}, "", err
		}
		row.Rule.Path = p
	default:
		row.Class, row.Rule = ClassAutomatic, f.modeRule()
	}
	row.Unattended = row.Class == ClassDrain || row.Class == ClassAutomatic ||
		(row.Class != ClassPersonApproved && !human(row.Mode) && (row.Lane == LaneUnknown || row.Approval.Source == SourceForgeUnread))
	return row, why, nil
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
		if ctxErr := l.ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
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
	trailerRe    = regexp.MustCompile(`(?i)^(resolves|delivers|assisted-by)\s*:\s*(\S+)\s*$`)
	issueRe      = regexp.MustCompile(`^iss-[0-9]{1,20}$`)
	assistedByRe = regexp.MustCompile(`^(None|[A-Za-z][A-Za-z0-9._-]{0,63}:[A-Za-z0-9][A-Za-z0-9._+-]{0,63})$`)
)

// trailers reads the Resolves:, Delivers: and Assisted-by: trailers of every
// message's trailer block (its last paragraph), once each, sorted. A value not
// in the shape its key takes is dropped.
func trailers(messages []string) (resolves, delivers, assisted []string) {
	add := func(list []string, v string) []string {
		if slices.Contains(list, v) {
			return list
		}
		return append(list, v)
	}
	for _, m := range messages {
		paras := strings.Split(strings.TrimRight(strings.ReplaceAll(m, "\r\n", "\n"), "\n"), "\n\n")
		for line := range strings.SplitSeq(paras[len(paras)-1], "\n") {
			t := trailerRe.FindStringSubmatch(strings.TrimSpace(line))
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
